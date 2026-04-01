package simulation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"ui/internal/config"
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
func (o *Orchestrator) Run(uavs []UAV, isLocal bool) (string, error) {
	simDir, resDir, err := o.createSimulationDirs()
	if err != nil {
		return "", err
	}

	writer := NewResourceWriter(resDir)
	builder := newComposeBuilder()
	builder.AddNetworkSimulator()

	for _, uav := range uavs {
		if err := o.appendUAV(uav, builder, writer); err != nil {
			return "", fmt.Errorf("uav %s: %w", uav.ID, err)
		}
	}

	composePath := filepath.Join(simDir, "docker-compose.yaml")
	if err := os.WriteFile(composePath, []byte(builder.Build()), 0644); err != nil {
		return "", fmt.Errorf("write docker-compose.yaml: %w", err)
	}

	return composePath, nil
}

// createSimulationDirs creates the timestamped simulation directory and the
// resources subdirectory inside it, returning both paths.
func (o *Orchestrator) createSimulationDirs() (simDir, resDir string, err error) {
	timestamp := time.Now().Format("20060102_150405")
	simDir = filepath.Join(o.paths.SimulationsDir, timestamp)
	resDir = filepath.Join(simDir, "resources")

	if err = os.MkdirAll(resDir, 0755); err != nil {
		return "", "", fmt.Errorf("create simulation dirs: %w", err)
	}
	return simDir, resDir, nil
}

// appendUAV adds all service blocks for a single UAV to the builder.
func (o *Orchestrator) appendUAV(uav UAV, builder *composeBuilder, writer *ResourceWriter) error {
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
	builder.AddUAVController(uav.ID, ucFileName)

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
		svcFileName, err := writer.Write(svc.ServiceId+"_config", svc.Config)
		if err != nil {
			return fmt.Errorf("service %q config: %w", svc.ServiceId, err)
		}
		builder.AddAlgorithmService(uav.ID, svc, svcFileName)
	}

	return nil
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
