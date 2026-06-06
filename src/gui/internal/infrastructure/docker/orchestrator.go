package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ui/internal/domain"
	"ui/internal/domain/formation"
	"ui/internal/infrastructure/util"
	"ui/internal/ports"
)

type DockerOrchestrator struct {
	projectRoot string
	ui          ports.UIBridge
	// paths needed for templates
	externalCommsConfig string
	netsimGatewayConfig string
	netsimConfig        string
	loggerConfig        string
	algorithmsDir       string
	mixersDir           string
	controllersDir      string
}

func NewDockerOrchestrator(projectRoot string, ui ports.UIBridge) *DockerOrchestrator {
	base := filepath.Clean(projectRoot)
	return &DockerOrchestrator{
		projectRoot:         base,
		ui:                  ui,
		externalCommsConfig: filepath.Join(base, "..", "external_comms", "config.json"),
		netsimGatewayConfig: filepath.Join(base, "..", "netsim_gateway", "config.json"),
		netsimConfig:        filepath.Join(base, "..", "netsim", "config.json"),
		loggerConfig:        filepath.Join(base, "..", "logger", "config.json"),
		algorithmsDir:       filepath.Join(base, "..", "algorithms"),
		mixersDir:           filepath.Join(base, "..", "mixers"),
		controllersDir:      filepath.Join(base, "..", "controllers"),
	}
}

func (o *DockerOrchestrator) Run(swarms []domain.Swarm, config domain.GeneralConfig, isLocal bool, simDir string) (string, error) {
	resDir := filepath.Join(simDir, "resources")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return "", fmt.Errorf("create simulation dirs: %w", err)
	}

	if isLocal {
		return o.buildLocalCompose(swarms, config, resDir, simDir)
	}
	return o.buildSwarmCompose(swarms, config, resDir, simDir)
}

func (o *DockerOrchestrator) PrepareExport(swarms []domain.Swarm, config domain.GeneralConfig, simDir string) error {
	resDir := filepath.Join(simDir, "resources")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return fmt.Errorf("create simulation dirs: %w", err)
	}

	if _, err := o.buildLocalCompose(swarms, config, resDir, simDir); err != nil {
		return err
	}
	if _, err := o.buildSwarmCompose(swarms, config, resDir, simDir); err != nil {
		return err
	}
	return nil
}

func normalizeNetsimInstances(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func (o *DockerOrchestrator) buildLocalCompose(swarms []domain.Swarm, config domain.GeneralConfig, resDir, simDir string) (string, error) {
	writer := NewResourceWriter(resDir)
	builder := newComposeBuilder()
	pool := newSubnetPool()

	netsimInstances := normalizeNetsimInstances(config.NetsimInstances)
	netsimAddrs := make([]string, netsimInstances)
	for i := 1; i <= netsimInstances; i++ {
		netsimAddrs[i-1] = fmt.Sprintf("netsim_%d:3000", i)
	}

	gwCfg := LoadRawConfig(o.netsimGatewayConfig)
	gwLimits := ParseResourceLimits(gwCfg)
	gwFile, _ := o.writeTemplateConfig("netsim_gateway_config", o.netsimGatewayConfig, nil, writer)
	builder.AddNetsimGateway(gwFile, netsimAddrs, gwLimits, config.VerboseLogging)

	nsCfg := LoadRawConfig(o.netsimConfig)
	nsLimits := ParseResourceLimits(nsCfg)
	nsOverrides := buildNetsimOverrides(config)
	nsFile, _ := o.writeTemplateConfig("netsim_config", o.netsimConfig, nsOverrides, writer)
	for i := 1; i <= netsimInstances; i++ {
		builder.AddNetsim(i, nsFile, nsLimits, config.VerboseLogging)
	}

	loggerCfg := LoadRawConfig(o.loggerConfig)
	loggerLimits := ParseResourceLimits(loggerCfg)
	loggerFile, _ := o.writeTemplateConfig("logger_config", o.loggerConfig, nil, writer)
	builder.AddLogger(loggerFile, loggerLimits)

	if config.LoggingEnabled {
		for _, swarm := range swarms {
			for _, uav := range swarm.UAVs {
				logDir := filepath.Join(simDir, "uav_logs", fmt.Sprintf("swarm_%s_uav_%s", swarm.ID, uav.ID))
				os.MkdirAll(logDir, 0755)
			}
		}
	}

	for _, swarm := range swarms {
		f := formation.GetFormation(swarm.GroundFormation)
		offsets := f.CalculateOffsets(len(swarm.UAVs), swarm.FormationSpacing)

		for i, uav := range swarm.UAVs {
			arduPilotInstance := config.DefaultArduPilotInstance
			if uav.ArduPilotInstance != nil {
				arduPilotInstance = *uav.ArduPilotInstance
			}
			var arduPilotInstanceFile string
			if arduPilotInstance != "" {
				arduPilotInstanceFile = filepath.Base(arduPilotInstance)
				destPath := filepath.Join(resDir, arduPilotInstanceFile)
				_ = copyFile(arduPilotInstance, destPath)
			}

			controller := o.resolveController(uav, config)
			paramFile, _ := o.generateUAVParams(swarm.ID, uav, config, controller.FolderName, resDir)
			o.appendUAV(swarm.ID, uav, paramFile, arduPilotInstanceFile, builder, writer, config, offsets[i], swarm, pool)
		}
	}

	composePath := filepath.Join(simDir, "docker-compose.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

func (o *DockerOrchestrator) buildSwarmCompose(swarms []domain.Swarm, config domain.GeneralConfig, resDir, simDir string) (string, error) {
	writer := NewResourceWriter(resDir)
	builder := newSwarmComposeBuilder()

	netsimInstances := normalizeNetsimInstances(config.NetsimInstances)
	netsimAddrs := make([]string, netsimInstances)
	for i := 1; i <= netsimInstances; i++ {
		netsimAddrs[i-1] = fmt.Sprintf("netsim_%d:3000", i)
	}

	gwCfg := LoadRawConfig(o.netsimGatewayConfig)
	gwLimits := ParseResourceLimits(gwCfg)
	gwFile, _ := o.writeTemplateConfig("netsim_gateway_config", o.netsimGatewayConfig, nil, writer)
	builder.AddNetsimGateway(gwFile, netsimAddrs, gwLimits, config.VerboseLogging)

	nsCfg := LoadRawConfig(o.netsimConfig)
	nsLimits := ParseResourceLimits(nsCfg)
	nsOverrides := buildNetsimOverrides(config)
	nsFile, _ := o.writeTemplateConfig("netsim_config", o.netsimConfig, nsOverrides, writer)
	for i := 1; i <= netsimInstances; i++ {
		builder.AddNetsim(i, nsFile, nsLimits, config.VerboseLogging)
	}

	loggerCfg := LoadRawConfig(o.loggerConfig)
	loggerLimits := ParseResourceLimits(loggerCfg)
	loggerFile, _ := o.writeTemplateConfig("logger_config", o.loggerConfig, nil, writer)
	builder.AddLogger(loggerFile, loggerLimits)

	for _, swarm := range swarms {
		f := formation.GetFormation(swarm.GroundFormation)
		offsets := f.CalculateOffsets(len(swarm.UAVs), swarm.FormationSpacing)

		for i, uav := range swarm.UAVs {
			arduPilotInstance := config.DefaultArduPilotInstance
			if uav.ArduPilotInstance != nil {
				arduPilotInstance = *uav.ArduPilotInstance
			}
			var arduPilotInstanceFile string
			if arduPilotInstance != "" {
				arduPilotInstanceFile = filepath.Base(arduPilotInstance)
				destPath := filepath.Join(resDir, arduPilotInstanceFile)
				_ = copyFile(arduPilotInstance, destPath)
			}

			controller := o.resolveController(uav, config)
			paramFile, _ := o.generateUAVParams(swarm.ID, uav, config, controller.FolderName, resDir)
			o.appendSwarmUAV(swarm.ID, uav, paramFile, arduPilotInstanceFile, builder, writer, config, offsets[i], swarm)
		}
	}

	composePath := filepath.Join(simDir, "docker-compose.swarm.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

// buildNetsimOverrides constructs a flat override map for the netsim config.json.
// It sets loss_mode always (when NetsimMode is set) and max_range_m only when
// the mode is "fixed_range" and the user has explicitly provided a value.
func buildNetsimOverrides(config domain.GeneralConfig) map[string]interface{} {
	if config.NetsimMode == "" {
		return nil
	}
	overrides := map[string]interface{}{
		"loss_mode": config.NetsimMode,
	}
	if config.NetsimMode == "fixed_range" && config.NetsimMaxRangeM != nil {
		overrides["max_range_m"] = *config.NetsimMaxRangeM
	}
	return overrides
}

func (o *DockerOrchestrator) StartCompose(composePath string) error {
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING")
	}

	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "up", "-d")
	cmd.Dir = filepath.Dir(composePath)

	var stderrBuf bytes.Buffer
	stdout, _ := cmd.StdoutPipe()
	// Tee stderr: emit to UI log AND capture in buffer for error reporting.
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := NewLogScanner(stdout, io.TeeReader(stderrPipe, &stderrBuf))
		for scanner.Scan() {
			o.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	waitErr := cmd.Wait()
	<-done // ensure all output has been flushed before reading the buffer

	if waitErr != nil {
		details := strings.TrimSpace(stderrBuf.String())
		if details != "" {
			return fmt.Errorf("docker compose: %w\n%s", waitErr, details)
		}
		return fmt.Errorf("docker compose: %w", waitErr)
	}
	return nil
}

func (o *DockerOrchestrator) BuildCompose(composePath string) error {
	checkCmd := exec.Command("docker", "info")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("DOCKER_NOT_RUNNING")
	}

	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "build")
	cmd.Dir = filepath.Dir(composePath)

	var stderrBuf bytes.Buffer
	stdout, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := NewLogScanner(stdout, io.TeeReader(stderrPipe, &stderrBuf))
		for scanner.Scan() {
			o.ui.EmitEvent("simulation:log", scanner.Text())
		}
	}()

	waitErr := cmd.Wait()
	<-done // ensure all output has been flushed before reading the buffer

	if waitErr != nil {
		details := strings.TrimSpace(stderrBuf.String())
		if details != "" {
			return fmt.Errorf("docker compose build: %w\n%s", waitErr, details)
		}
		return fmt.Errorf("docker compose build: %w", waitErr)
	}
	o.ui.EmitEvent("simulation:log", "[Build] Images successfully built.")
	return nil
}

func (o *DockerOrchestrator) BuildAllImages(simDir string) error {
	composeStr := `services:
  netsim_gateway:
    image: netsim_gateway
    build:
      context: ../../src/netsim_gateway
      dockerfile: Dockerfile
  netsim:
    image: netsim
    build:
      context: ../../src/netsim
      dockerfile: Dockerfile
  logger:
    image: logger
    build:
      context: ../../src/logger
      dockerfile: Dockerfile
  communication_module:
    image: communication_module
    build:
      context: ../../src/communication_module
      dockerfile: Dockerfile
  external_comms:
    image: external_comms
    build:
      context: ../../src/external_comms
      dockerfile: Dockerfile
`
	appendServices := func(dir, category string, dockerfile string) {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					name := entry.Name()
					composeStr += fmt.Sprintf(`  %s:
    image: %s
    build:
      context: ../../src/%s/%s
      dockerfile: %s
`, name, name, category, name, dockerfile)
				}
			}
		}
	}

	appendServices(o.algorithmsDir, "algorithms", "Dockerfile")
	appendServices(o.mixersDir, "mixers", "Dockerfile")
	appendServices(o.controllersDir, "controllers", "SITL")

	resDir := filepath.Join(simDir, "resources")
	os.MkdirAll(resDir, 0755)

	composePath := filepath.Join(simDir, "docker-compose.build.yaml")
	if err := os.WriteFile(composePath, []byte(composeStr), 0644); err != nil {
		return fmt.Errorf("failed to write build compose file: %w", err)
	}

	return o.BuildCompose(composePath)
}

func (o *DockerOrchestrator) StopCompose(composePath string) error {
	cmd := exec.Command("docker", "compose", "-f", filepath.Base(composePath), "down", "--remove-orphans", "--volumes")
	cmd.Dir = filepath.Dir(composePath)
	return cmd.Run()
}

func (o *DockerOrchestrator) StartStack(composePath, swarmHost, stackName string) error {
	dockerEnv := os.Environ()
	if !o.isLocalhost(swarmHost) && swarmHost != "" {
		dockerEnv = append(dockerEnv, "DOCKER_HOST="+swarmHost)
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
	if !o.isLocalhost(swarmHost) && swarmHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+swarmHost)
	}
	return cmd.Run()
}

func (o *DockerOrchestrator) CollectSwarmLogs(stackName, swarmHost, simName, destDir string) error {
	// TODO: implement
	return nil
}

func (o *DockerOrchestrator) isLocalhost(host string) bool {
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1")
}

// Ported helpers from ui/internal/simulation/orchestrator.go

func (o *DockerOrchestrator) appendUAV(swarmID string, uav domain.UAV, paramFileName, arduPilotInstanceFile string, builder *composeBuilder, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset, swarm domain.Swarm, pool *subnetPool) {
	nContainers := 4 + len(uav.Services)
	subnet := pool.Next(nContainers)
	builder.AddUAVNetwork(swarmID, uav.ID, subnet)
	builder.AddCommunicationModule(swarmID, uav.ID, ResourceLimits{}, config.VerboseLogging)

	mixer := o.resolveMixer(uav, config)
	appFile, _, appLimits := o.buildServiceResources(mixer, writer)
	builder.AddMixer(swarmID, uav.ID, mixer.FolderName, appFile, appLimits, config.VerboseLogging)

	controller := o.resolveController(uav, config)
	ucFile, _, ucLimits := o.buildServiceResources(controller, writer)
	var homeLat, homeLon float64
	if uav.HomeOverride != nil {
		homeLat, homeLon = uav.HomeOverride.Lat, uav.HomeOverride.Lon
	} else {
		homeLat, homeLon = util.AddOffset(swarm.FormationCenterLat, swarm.FormationCenterLon, offset.X, offset.Y)
	}
	homeLocation := fmt.Sprintf("%f,%f,0,0", homeLat, homeLon)
	builder.AddUAVController(swarmID, uav.ID, controller.FolderName, ucFile, paramFileName, homeLocation, arduPilotInstanceFile, ucLimits, config.VerboseLogging, config.LoggingEnabled)

	ecCfg := LoadRawConfig(o.externalCommsConfig)
	ecLimits := ParseResourceLimits(ecCfg)
	ecFile, _ := o.writeTemplateConfig("external_comms_config", o.externalCommsConfig, nil, writer)
	builder.AddExternalComms(swarmID, uav.ID, ecFile, ecLimits, config.VerboseLogging)

	for _, svc := range uav.Services {
		svcFile, extraVolumes, algoLimits := o.buildServiceResources(svc, writer)
		builder.AddAlgorithmService(swarmID, uav.ID, svc, svcFile, extraVolumes, algoLimits, config.VerboseLogging)
	}
}

func (o *DockerOrchestrator) appendSwarmUAV(swarmID string, uav domain.UAV, paramFileName, arduPilotInstanceFile string, builder *swarmComposeBuilder, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset, swarm domain.Swarm) {
	builder.AddUAVNetwork(swarmID, uav.ID)
	builder.AddCommunicationModule(swarmID, uav.ID, ResourceLimits{}, config.VerboseLogging)

	mixer := o.resolveMixer(uav, config)
	appFile, _, appLimits := o.buildServiceResources(mixer, writer)
	builder.AddMixer(swarmID, uav.ID, mixer.FolderName, appFile, appLimits, config.VerboseLogging)

	controller := o.resolveController(uav, config)
	ucFile, _, ucLimits := o.buildServiceResources(controller, writer)
	var homeLat, homeLon float64
	if uav.HomeOverride != nil {
		homeLat, homeLon = uav.HomeOverride.Lat, uav.HomeOverride.Lon
	} else {
		homeLat, homeLon = util.AddOffset(swarm.FormationCenterLat, swarm.FormationCenterLon, offset.X, offset.Y)
	}
	homeLocation := fmt.Sprintf("%f,%f,0,0", homeLat, homeLon)
	builder.AddUAVController(swarmID, uav.ID, controller.FolderName, ucFile, paramFileName, homeLocation, arduPilotInstanceFile, ucLimits, config.VerboseLogging, config.LoggingEnabled)

	ecCfg := LoadRawConfig(o.externalCommsConfig)
	ecLimits := ParseResourceLimits(ecCfg)
	ecFile, _ := o.writeTemplateConfig("external_comms_config", o.externalCommsConfig, nil, writer)
	builder.AddExternalComms(swarmID, uav.ID, ecFile, ecLimits, config.VerboseLogging)

	for _, svc := range uav.Services {
		svcFile, extraVolumes, algoLimits := o.buildServiceResources(svc, writer)
		builder.AddAlgorithmService(swarmID, uav.ID, svc, svcFile, extraVolumes, algoLimits, config.VerboseLogging)
	}
}

func (o *DockerOrchestrator) resolveMixer(uav domain.UAV, config domain.GeneralConfig) domain.DeployedService {
	if uav.Mixer != nil {
		return *uav.Mixer
	}
	return config.DefaultMixer
}

func (o *DockerOrchestrator) resolveController(uav domain.UAV, config domain.GeneralConfig) domain.DeployedService {
	if uav.Controller != nil {
		return *uav.Controller
	}
	return config.DefaultController
}

func (o *DockerOrchestrator) buildServiceResources(svc domain.DeployedService, writer *ResourceWriter) (string, []domain.VolumeMount, ResourceLimits) {
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

	var algoLimits ResourceLimits
	if schema, err := o.getServiceSchema(svc.FolderName); err == nil {
		algoLimits = ParseResourceLimits(schema)
	}

	return svcFile, extraVolumes, algoLimits
}

func (o *DockerOrchestrator) getServiceSchema(folderName string) (map[string]interface{}, error) {
	searchDirs := []string{o.algorithmsDir, o.mixersDir, o.controllersDir}
	for _, dir := range searchDirs {
		schemaPath := filepath.Join(dir, folderName, "schema.json")
		if rawData, err := os.ReadFile(schemaPath); err == nil {
			var schema map[string]interface{}
			if err := json.Unmarshal(rawData, &schema); err == nil {
				return schema, nil
			}
		}
	}
	return nil, fmt.Errorf("schema not found for %s", folderName)
}

func (o *DockerOrchestrator) generateUAVParams(swarmID string, uav domain.UAV, config domain.GeneralConfig, controllerFolderName string, resDir string) (string, error) {
	baseParmPath := filepath.Join(o.projectRoot, "..", "controllers", controllerFolderName, "ardupilot", "copter.parm")
	content, _ := os.ReadFile(baseParmPath)
	params := string(content)
	if !strings.HasSuffix(params, "\n") {
		params += "\n"
	}
	if !config.LoggingEnabled {
		params += "LOG_BITMASK 0\n"
	}
	uavBattery := config.BatteryCapacity
	if uavBattery <= 0 {
		uavBattery = 5000
	}
	if uav.BatteryCapacity != nil {
		uavBattery = *uav.BatteryCapacity
	}
	params += fmt.Sprintf("BATT_CAPACITY %d\n", uavBattery)
	params += fmt.Sprintf("FS_BATT_MAH %d\n", uavBattery*20/100)
	params += "FS_BATT_ENABLE 2\n"
	params += "BATT_MONITOR 4\n"
	if config.WindEnabled {
		params += fmt.Sprintf("SIM_WIND_DIR %.2f\n", config.WindDirection)
		params += fmt.Sprintf("SIM_WIND_SPD %.2f\n", config.WindSpeed)
	}
	uavSpeed := config.DefaultUAVSpeed
	if uavSpeed <= 0 {
		uavSpeed = 10.0
	}
	if uav.Speed != nil {
		uavSpeed = *uav.Speed
	}
	params += fmt.Sprintf("WPNAV_SPEED %d\n", int(uavSpeed*100))
	params += fmt.Sprintf("WPNAV_SPEED_UP %d\n", int(uavSpeed*100))
	params += fmt.Sprintf("WPNAV_SPEED_DN %d\n", int(uavSpeed*100))

	fileName := fmt.Sprintf("swarm_%s_uav_%s_params.param", swarmID, uav.ID)
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
