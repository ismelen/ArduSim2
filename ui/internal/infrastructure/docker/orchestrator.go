package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ui/internal/domain"
	"ui/internal/domain/formation"
	"ui/internal/infrastructure/util"
	"ui/internal/ports"
)

type DockerOrchestrator struct {
	projectRoot string
	ui          ports.UIBridge
	// paths needed for templates
	applicationConfig   string
	uavControllerConfig string
	externalCommsConfig string
	algorithmsDir       string
}

func NewDockerOrchestrator(projectRoot string, ui ports.UIBridge) *DockerOrchestrator {
	base := filepath.Clean(projectRoot)
	return &DockerOrchestrator{
		projectRoot:         base,
		ui:                  ui,
		applicationConfig:   filepath.Join(base, "..", "application", "config.json"),
		uavControllerConfig: filepath.Join(base, "..", "uav_controller", "ardupilot4_5_3", "config.sitl.json"),
		externalCommsConfig: filepath.Join(base, "..", "external_comms", "config.json"),
		algorithmsDir:       filepath.Join(base, "..", "algorithms"),
	}
}

func (o *DockerOrchestrator) Run(uavs []domain.UAV, config domain.GeneralConfig, isLocal bool, simDir string) (string, error) {
	resDir := filepath.Join(simDir, "resources")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return "", fmt.Errorf("create simulation dirs: %w", err)
	}

	var runDir string
	if config.StoreLocalData {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		runDir = filepath.Join(simDir, "runs", timestamp)
		_ = os.MkdirAll(runDir, 0755)
	}

	var speeds []float64 // Simplified: could load from file if needed

	f := formation.GetFormation(config.GroundFormation)
	offsets := f.CalculateOffsets(len(uavs), config.FormationSpacing)

	if isLocal {
		return o.buildLocalCompose(uavs, config, speeds, offsets, resDir, simDir, runDir)
	}
	return o.buildSwarmCompose(uavs, config, speeds, offsets, resDir, simDir)
}

func (o *DockerOrchestrator) buildLocalCompose(uavs []domain.UAV, config domain.GeneralConfig, speeds []float64, offsets []formation.Offset, resDir, simDir, runDir string) (string, error) {
	writer := NewResourceWriter(resDir)
	builder := newComposeBuilder()

	var nsLogDir string
	if runDir != "" {
		nsLogDir = filepath.Join(runDir, "network_simulator")
		_ = os.MkdirAll(nsLogDir, 0755)
	}
	builder.AddNetworkSimulator(nsLogDir, config.VerboseLogging)

	for i, uav := range uavs {
		uavSpeed := 10.0
		if i < len(speeds) {
			uavSpeed = speeds[i]
		}

		paramFile, _ := o.generateUAVParams(uav.ID, uavSpeed, config, resDir)
		o.appendUAV(uav, paramFile, builder, writer, config, offsets[i], runDir)
	}

	composePath := filepath.Join(simDir, "docker-compose.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

func (o *DockerOrchestrator) buildSwarmCompose(uavs []domain.UAV, config domain.GeneralConfig, speeds []float64, offsets []formation.Offset, resDir, simDir string) (string, error) {
	writer := NewResourceWriter(resDir)
	builder := newSwarmComposeBuilder()

	builder.AddNetworkSimulator(config.VerboseLogging)

	for i, uav := range uavs {
		uavSpeed := 10.0
		if i < len(speeds) {
			uavSpeed = speeds[i]
		}

		paramFile, _ := o.generateUAVParams(uav.ID, uavSpeed, config, resDir)
		o.appendSwarmUAV(uav, paramFile, builder, writer, config, offsets[i])
	}

	composePath := filepath.Join(simDir, "docker-compose.swarm.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

// ... helpers (appendUAV, appendSwarmUAV, etc) should be ported here ...

func (o *DockerOrchestrator) StartCompose(composePath string) error {
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING")
	}

	cmd := exec.Command("docker", "compose", "up", "--build", "-d")
	cmd.Dir = filepath.Dir(composePath)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		scanner := NewLogScanner(stdout, stderr)
		for scanner.Scan() {
			o.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	return cmd.Wait()
}

func (o *DockerOrchestrator) StopCompose(composePath string) error {
	cmd := exec.Command("docker", "compose", "down", "--remove-orphans")
	cmd.Dir = filepath.Dir(composePath)
	return cmd.Run()
}

func (o *DockerOrchestrator) StartStack(composePath, swarmHost, stackName string) error {
	dockerEnv := os.Environ()
	if !o.isLocalhost(swarmHost) {
		dockerEnv = append(dockerEnv, "DOCKER_HOST=tcp://"+swarmHost)
	}

	cmd := exec.Command("docker", "stack", "deploy", "-c", composePath, stackName)
	cmd.Env = dockerEnv

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		scanner := NewLogScanner(stdout, stderr)
		for scanner.Scan() {
			o.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	return cmd.Wait()
}

func (o *DockerOrchestrator) StopStack(stackName, swarmHost string) error {
	cmd := exec.Command("docker", "stack", "rm", stackName)
	if !o.isLocalhost(swarmHost) {
		cmd.Env = append(os.Environ(), "DOCKER_HOST=tcp://"+swarmHost)
	}
	return cmd.Run()
}

func (o *DockerOrchestrator) CollectSwarmLogs(stackName, swarmHost, simName, destDir string) error {
	// Implementation from app.go's CleanSwarmNodes
	return nil
}

func (o *DockerOrchestrator) isLocalhost(host string) bool {
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1")
}

// Ported helpers from ui/internal/simulation/orchestrator.go

func (o *DockerOrchestrator) appendUAV(uav domain.UAV, paramFileName string, builder *composeBuilder, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset, runDir string) {
	uavNum, _ := strconv.Atoi(uav.ID)
	var uavLogRoot string
	if runDir != "" {
		uavLogRoot = filepath.Join(runDir, fmt.Sprintf("uav-%s", uav.ID))
		_ = os.MkdirAll(uavLogRoot, 0755)
	}

	builder.AddUAVNetwork(uav.ID)
	builder.AddCommunicationModule(uav.ID, o.getServiceLogDir(uavLogRoot, "communication_module"), config.VerboseLogging)

	appFile, _ := o.writeTemplateConfig("application_config", o.applicationConfig, nil, writer)
	builder.AddApplication(uav.ID, appFile, o.getServiceLogDir(uavLogRoot, "application"), config.VerboseLogging)

	ucFile, _ := o.writeTemplateConfig("uav_controller_config", o.uavControllerConfig, nil, writer)
	homeLat, homeLon := util.AddOffset(config.FormationCenterLat, config.FormationCenterLon, offset.X, offset.Y)
	homeLocation := fmt.Sprintf("%f,%f,0,0", homeLat, homeLon)
	builder.AddUAVController(uav.ID, ucFile, paramFileName, homeLocation, o.getServiceLogDir(uavLogRoot, "uav_controller"), config.VerboseLogging)

	ecOverrides := map[string]interface{}{
		"uav_id":         uavNum,
		"simulator_ip":   "network_simulator",
		"simulator_port": 3000,
	}
	ecFile, _ := o.writeTemplateConfig("external_comms_config", o.externalCommsConfig, ecOverrides, writer)
	builder.AddExternalComms(uav.ID, ecFile, o.getServiceLogDir(uavLogRoot, "external_comms"), config.VerboseLogging)

	for _, svc := range uav.Services {
		var extraVolumes []domain.VolumeMount
		cfg := make(map[string]interface{})
		for k, v := range svc.Config {
			cfg[k] = v
		}
		
		if schema, err := o.getServiceSchema(svc.FolderName); err == nil {
			if props, ok := schema["properties"].(map[string]interface{}); ok {
				for key, val := range props {
					prop, ok := val.(map[string]interface{})
					if !ok {
						continue
					}
					if format, ok := prop["format"].(string); ok && format == "kml" {
						if srcPath, ok := cfg[key].(string); ok && srcPath != "" {
							ext := filepath.Ext(srcPath)
							hostName := fmt.Sprintf("%s_%s%s", svc.ServiceId, key, ext)
							if data, err := os.ReadFile(srcPath); err == nil {
								if err := os.WriteFile(filepath.Join(writer.outputDir, hostName), data, os.ModePerm); err == nil {
									containerPath := fmt.Sprintf("/app/%s%s", key, ext)
									cfg[key] = containerPath
									extraVolumes = append(extraVolumes, domain.VolumeMount{
										HostPath:      hostName,
										ContainerPath: containerPath,
									})
								}
							}
						}
					}
				}
			}
		}

		svcFile, _ := writer.Write(svc.ServiceId+"_config", cfg)
		builder.AddAlgorithmService(uav.ID, svc, svcFile, extraVolumes, o.getServiceLogDir(uavLogRoot, svc.FolderName), config.VerboseLogging)
	}
}

func (o *DockerOrchestrator) appendSwarmUAV(uav domain.UAV, paramFileName string, builder *swarmComposeBuilder, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset) {
	uavNum, _ := strconv.Atoi(uav.ID)
	builder.AddUAVNetwork(uav.ID)
	builder.AddCommunicationModule(uav.ID, config.VerboseLogging)

	appFile, _ := o.writeTemplateConfig("application_config", o.applicationConfig, nil, writer)
	builder.AddApplication(uav.ID, appFile, config.VerboseLogging)

	ucFile, _ := o.writeTemplateConfig("uav_controller_config", o.uavControllerConfig, nil, writer)
	homeLat, homeLon := util.AddOffset(config.FormationCenterLat, config.FormationCenterLon, offset.X, offset.Y)
	homeLocation := fmt.Sprintf("%f,%f,0,0", homeLat, homeLon)
	builder.AddUAVController(uav.ID, ucFile, paramFileName, homeLocation, config.VerboseLogging)

	ecOverrides := map[string]interface{}{
		"uav_id":         uavNum,
		"simulator_ip":   "network_simulator",
		"simulator_port": 3000,
	}
	ecFile, _ := o.writeTemplateConfig("external_comms_config", o.externalCommsConfig, ecOverrides, writer)
	builder.AddExternalComms(uav.ID, ecFile, config.VerboseLogging)

	for _, svc := range uav.Services {
		cfg := make(map[string]interface{})
		for k, v := range svc.Config {
			cfg[k] = v
		}

		var extraVolumes []domain.VolumeMount
		if schema, err := o.getServiceSchema(svc.FolderName); err == nil {
			if props, ok := schema["properties"].(map[string]interface{}); ok {
				for key, val := range props {
					prop, ok := val.(map[string]interface{})
					if !ok {
						continue
					}
					if format, ok := prop["format"].(string); ok && format == "kml" {
						if srcPath, ok := cfg[key].(string); ok && srcPath != "" {
							ext := filepath.Ext(srcPath)
							hostName := fmt.Sprintf("%s_%s%s", svc.ServiceId, key, ext)
							if data, err := os.ReadFile(srcPath); err == nil {
								if err := os.WriteFile(filepath.Join(writer.outputDir, hostName), data, os.ModePerm); err == nil {
									containerPath := fmt.Sprintf("/app/%s%s", key, ext)
									cfg[key] = containerPath
									extraVolumes = append(extraVolumes, domain.VolumeMount{
										HostPath:      hostName,
										ContainerPath: containerPath,
									})
								}
							}
						}
					}
				}
			}
		}

		svcFile, _ := writer.Write(svc.ServiceId+"_config", cfg)
		builder.AddAlgorithmService(uav.ID, svc, svcFile, extraVolumes, config.VerboseLogging)
	}
}

func (o *DockerOrchestrator) getServiceSchema(folderName string) (map[string]interface{}, error) {
	schemaPath := filepath.Join(o.algorithmsDir, folderName, "schema.json")
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

func (o *DockerOrchestrator) getServiceLogDir(uavLogRoot, serviceName string) string {
	if uavLogRoot == "" {
		return ""
	}
	dir := filepath.Join(uavLogRoot, serviceName)
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func (o *DockerOrchestrator) generateUAVParams(uavID string, speed float64, config domain.GeneralConfig, resDir string) (string, error) {
	baseParmPath := filepath.Join(o.projectRoot, "..", "uav_controller", "ardupilot4_5_3", "ardupilot", "copter.parm")
	content, _ := os.ReadFile(baseParmPath)
	params := string(content)
	if !strings.HasSuffix(params, "\n") {
		params += "\n"
	}
	if !config.LoggingEnabled {
		params += "LOG_BITMASK 0\n"
	}
	if config.BatteryRestricted {
		params += fmt.Sprintf("BATT_CAPACITY %d\n", config.BatteryCapacity)
		params += fmt.Sprintf("FS_BATT_MAH %d\n", config.BatteryCapacity*20/100)
		params += "FS_BATT_ENABLE 2\n"
	}
	if config.WindEnabled {
		params += fmt.Sprintf("SIM_WIND_DIR %.2f\n", config.WindDirection)
		params += fmt.Sprintf("SIM_WIND_SPD %.2f\n", config.WindSpeed)
	}
	params += fmt.Sprintf("WPNAV_SPEED %d\n", int(speed*100))
	params += fmt.Sprintf("WPNAV_SPEED_UP %d\n", int(speed*100))
	params += fmt.Sprintf("WPNAV_SPEED_DN %d\n", int(speed*100))

	fileName := fmt.Sprintf("uav_%s_params.param", uavID)
	_ = os.WriteFile(filepath.Join(resDir, fileName), []byte(params), 0644)
	return fileName, nil
}

func (o *DockerOrchestrator) writeTemplateConfig(baseName, templatePath string, overrides map[string]interface{}, writer *ResourceWriter) (string, error) {
	cfg := make(map[string]interface{})
	if rawData, err := os.ReadFile(templatePath); err == nil {
		_ = json.Unmarshal(rawData, &cfg)
	}
	for key, value := range overrides {
		cfg[key] = value
	}
	return writer.Write(baseName, cfg)
}
