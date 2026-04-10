package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ui/internal/config"
	"ui/internal/geo"
	"ui/internal/simulation/formation"
)

// Orchestrator coordinates a full simulation run: creates the directory tree,
// writes deduplicated configs, generates Docker Compose YAML, and optionally
// launches Docker.
type Orchestrator struct {
	paths config.Paths
}

// NewOrchestrator creates an Orchestrator using the provided resolved paths.
func NewOrchestrator(paths config.Paths) *Orchestrator {
	return &Orchestrator{paths: paths}
}

// Run generates the simulation directory and Docker Compose file for the given
// fleet, then launches Docker if isLocal is true.
// Returns the path to the generated docker-compose.yaml.
func (o *Orchestrator) Run(uavs []UAV, generalConfig GeneralConfig, activeMode string, isLocal bool, simDir string) (string, error) {
	resDir := filepath.Join(simDir, "resources")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return "", fmt.Errorf("create simulation dirs: %w", err)
	}

	// Capture and save the full simulation state for later reloading.
	o.saveSimulationState(uavs, generalConfig, activeMode, simDir)

	// Generate custom ArduPilot parameters based on the template and UI configuration.
	paramFileName, err := o.generateCustomParams(generalConfig, resDir)
	if err != nil {
		return "", fmt.Errorf("generate custom params: %w", err)
	}

	writer := NewResourceWriter(resDir)
	builder := newComposeBuilder()
	builder.AddNetworkSimulator()

	// Calculate ground formation offsets (Strategy Pattern)
	f := formation.GetFormation(generalConfig.GroundFormation)
	offsets := f.CalculateOffsets(len(uavs), generalConfig.FormationSpacing)

	for i, uav := range uavs {
		if err := o.appendUAV(uav, paramFileName, builder, writer, generalConfig, offsets[i]); err != nil {
			return "", fmt.Errorf("uav %s: %w", uav.ID, err)
		}
	}

	composePath := filepath.Join(simDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(builder.Build()), 0644); err != nil {
		return "", fmt.Errorf("write docker-compose.yaml: %w", err)
	}

	return composePath, nil
}



// appendUAV adds all service blocks for a single UAV to the builder.
func (o *Orchestrator) appendUAV(uav UAV, paramFileName string, builder *composeBuilder, writer *ResourceWriter, config GeneralConfig, offset formation.Offset) error {
	uavNum, _ := strconv.Atoi(uav.ID)

	builder.AddUAVNetwork(uav.ID)
	builder.AddCommunicationModule(uav.ID)

	appFileName, err := o.writeTemplateConfig("application_config", nil, writer)
	if err != nil {
		return err
	}
	builder.AddApplication(uav.ID, appFileName)

	ucFileName, err := o.writeTemplateConfig("uav_controller_config", nil, writer)
	if err != nil {
		return err
	}

	// Calculate absolute home location for this UAV
	homeLat, homeLon := geo.AddOffset(config.FormationCenterLat, config.FormationCenterLon, offset.X, offset.Y)
	homeLocation := fmt.Sprintf("%f,%f,0,0", homeLat, homeLon)

	builder.AddUAVController(uav.ID, ucFileName, paramFileName, homeLocation)

	ecOverrides := map[string]interface{}{
		// uav_id and simulator coordinates must be unique per UAV instance.
		"uav_id":        uavNum,
		"simulator_ip":  "network_simulator",
		"simulator_port": 3000,
	}
	ecFileName, err := o.writeTemplateConfig("external_comms_config", ecOverrides, writer)
	if err != nil {
		return err
	}
	builder.AddExternalComms(uav.ID, ecFileName)

	for _, svc := range uav.Services {
		var extraVolumes []VolumeMount
		// Clone config to avoid modifying common map across drones
		cfg := make(map[string]interface{})
		for k, v := range svc.Config {
			cfg[k] = v
		}

		if schema, err := o.getServiceSchema(svc.ServiceId); err == nil {
			if props, ok := schema["properties"].(map[string]interface{}); ok {
				for key, val := range props {
					prop, ok := val.(map[string]interface{})
					if !ok {
						continue
					}

					// We treat fields with format "kml" as auxiliary files that need mounting.
					if format, ok := prop["format"].(string); ok && format == "kml" {
						if srcPath, ok := cfg[key].(string); ok && srcPath != "" {
							ext := filepath.Ext(srcPath)
							// To avoid collisions in the flat resources dir, prefix with service ID
							hostName := fmt.Sprintf("%s_%s%s", svc.ServiceId, key, ext)

							// Copy the source file to the simulation resources directory
							if data, err := os.ReadFile(srcPath); err == nil {
								if err := os.WriteFile(filepath.Join(writer.outputDir, hostName), data, os.ModePerm); err == nil {
									// Update the config to point to the path inside the container.
									// We use the property key as the filename for stability.
									containerPath := fmt.Sprintf("/app/%s%s", key, ext)
									cfg[key] = containerPath

									extraVolumes = append(extraVolumes, VolumeMount{
										HostPath:      hostName,
										ContainerPath: containerPath,
									})
								} else {
									fmt.Printf("[orchestrator] failed to write auxiliary file: %v\n", err)
								}
							} else {
								fmt.Printf("[orchestrator] auxiliary file not found at %q: %v\n", srcPath, err)
							}
						}
					}
				}
			}
		}

		svcFileName, err := writer.Write(svc.ServiceId+"_config", cfg)
		if err != nil {
			return fmt.Errorf("service %q config: %w", svc.ServiceId, err)
		}
		builder.AddAlgorithmService(uav.ID, svc, svcFileName, extraVolumes)
	}

	return nil
}

// generateCustomParams reads the base copter.parm and appends values from generalConfig.
func (o *Orchestrator) generateCustomParams(config GeneralConfig, resDir string) (string, error) {
	// Base copter.parm is located in uav_controller/ardupilot4_5_3/ardupilot/
	// o.paths.ResourcesDir is ui/../resources
	baseParmPath := filepath.Join(o.paths.ResourcesDir, "..", "uav_controller", "ardupilot4_5_3", "ardupilot", "copter.parm")
	content, err := os.ReadFile(baseParmPath)
	if err != nil {
		return "", fmt.Errorf("read base parm: %w", err)
	}

	params := string(content)
	if !strings.HasSuffix(params, "\n") {
		params += "\n"
	}

	// Append custom parameters
	if !config.LoggingEnabled {
		params += "LOG_BITMASK 0\n"
	}

	if config.BatteryRestricted {
		params += fmt.Sprintf("BATT_CAPACITY %d\n", config.BatteryCapacity)
		// Set failsafe to 20% of capacity
		params += fmt.Sprintf("FS_BATT_MAH %d\n", config.BatteryCapacity*20/100)
		params += "FS_BATT_ENABLE 2\n" // RTL on battery failsafe
	}

	if config.WindEnabled {
		params += fmt.Sprintf("SIM_WIND_DIR %.2f\n", config.WindDirection)
		params += fmt.Sprintf("SIM_WIND_SPD %.2f\n", config.WindSpeed)
	}

	fileName := "custom_params.param"
	destPath := filepath.Join(resDir, fileName)
	if err := os.WriteFile(destPath, []byte(params), 0644); err != nil {
		return "", fmt.Errorf("write custom params: %w", err)
	}

	return fileName, nil
}

// writeTemplateConfig loads a base JSON config from the resources directory,
// applies any overrides on top of it, and delegates writing to the ResourceWriter.
func (o *Orchestrator) writeTemplateConfig(baseName string, overrides map[string]interface{}, writer *ResourceWriter) (string, error) {
	templatePath := filepath.Join(o.paths.ResourcesDir, baseName+".json")

	cfg := make(map[string]interface{})
	if rawData, err := os.ReadFile(templatePath); err == nil {
		// Non-fatal: if the template is missing we start from an empty config.
		_ = json.Unmarshal(rawData, &cfg)
	}

	for key, value := range overrides {
		cfg[key] = value
	}

	return writer.Write(baseName, cfg)
}

// saveSimulationState writes a simulation.json to the simulation root directory,
// enabling the UI to reconstruct its state if the user loads this run later.
func (o *Orchestrator) saveSimulationState(uavs []UAV, config GeneralConfig, mode string, simDir string) {
	state := SimulationState{
		UAVs:          uavs,
		GeneralConfig: config,
		ActiveMode:    mode,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		fmt.Printf("[orchestrator] failed to marshal state: %v\n", err)
		return
	}

	statePath := filepath.Join(simDir, "simulation.json")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		fmt.Printf("[orchestrator] failed to save state: %v\n", err)
	}
}

func (o *Orchestrator) getServiceSchema(serviceID string) (map[string]interface{}, error) {
	schemaPath := filepath.Join(o.paths.AlgorithmsDir, serviceID, "schema.json")
	rawData, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(rawData, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}
