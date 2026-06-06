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
	return o.buildKubernetesManifests(swarms, config, resDir, simDir)
}

func (o *DockerOrchestrator) PrepareExport(swarms []domain.Swarm, config domain.GeneralConfig, simDir string) error {
	resDir := filepath.Join(simDir, "resources")
	if err := os.MkdirAll(resDir, 0755); err != nil {
		return fmt.Errorf("create simulation dirs: %w", err)
	}

	if _, err := o.buildLocalCompose(swarms, config, resDir, simDir); err != nil {
		return err
	}
	if _, err := o.buildKubernetesManifests(swarms, config, resDir, simDir); err != nil {
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

	telPort, msgPort, subPort := 3000, 3001, 3002
	if p, ok := gwCfg["telemetry_port"].(float64); ok {
		telPort = int(p)
	}
	if p, ok := gwCfg["messages_port"].(float64); ok {
		msgPort = int(p)
	}
	if p, ok := gwCfg["subscribers_port"].(float64); ok {
		subPort = int(p)
	}

	gwEnv := make(map[string]string)
	if config.VerboseLogging {
		gwEnv["DEBUG"] = "true"
	}
	if len(netsimAddrs) > 0 {
		gwEnv["ADDRS"] = strings.Join(netsimAddrs, ",")
	}
	
	builder.addNetwork("air", "10.9.0.0/24")
	builder.AddService(ComposeService{
		Name:          "netsim_gateway",
		Image:         "netsim_gateway",
		ContainerName: "netsim_gateway",
		ExtraHosts:    []string{"host.docker.internal:host-gateway"},
		Ports: []string{
			fmt.Sprintf("%d:%d/udp", telPort, telPort),
			fmt.Sprintf("%d:%d/udp", msgPort, msgPort),
			fmt.Sprintf("%d:%d/udp", subPort, subPort),
		},
		Environment: gwEnv,
		Volumes:     []string{fmt.Sprintf("./resources/%s:/app/config.json", gwFile)},
		Networks:    map[string][]string{"air": nil},
		Limits:      gwLimits,
	})

	nsCfg := LoadRawConfig(o.netsimConfig)
	nsLimits := ParseResourceLimits(nsCfg)
	nsOverrides := buildNetsimOverrides(config)
	nsFile, _ := o.writeTemplateConfig("netsim_config", o.netsimConfig, nsOverrides, writer)

	for i := 1; i <= netsimInstances; i++ {
		nsEnv := map[string]string{"NODE_ID": fmt.Sprintf("netsim_%d", i)}
		if config.VerboseLogging {
			nsEnv["DEBUG"] = "true"
		}
		builder.AddService(ComposeService{
			Name:          fmt.Sprintf("netsim_%d", i),
			Image:         "netsim",
			ContainerName: fmt.Sprintf("netsim_%d", i),
			DependsOn:     []string{"netsim_gateway"},
			Environment:   nsEnv,
			Volumes:       []string{fmt.Sprintf("./resources/%s:/app/config.json", nsFile)},
			Networks:      map[string][]string{"air": nil},
			Limits:        nsLimits,
		})
	}

	loggerCfg := LoadRawConfig(o.loggerConfig)
	loggerLimits := ParseResourceLimits(loggerCfg)
	loggerFile, _ := o.writeTemplateConfig("logger_config", o.loggerConfig, nil, writer)

	builder.AddService(ComposeService{
		Name:          "logger",
		Image:         "logger",
		ContainerName: "logger",
		Ports:         []string{"5000:5000/udp", "8080:8080/tcp"},
		Volumes:       []string{fmt.Sprintf("./resources/%s:/app/config.json", loggerFile)},
		Networks:      map[string][]string{"air": nil},
		Limits:        loggerLimits,
	})

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
			
			uavServices := o.buildComposeUAV(swarm.ID, uav, paramFile, arduPilotInstanceFile, writer, config, offsets[i], swarm, pool, builder)
			for _, svc := range uavServices {
				builder.AddService(svc)
			}
		}
	}

	composePath := filepath.Join(simDir, "docker-compose.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

func (o *DockerOrchestrator) buildKubernetesManifests(swarms []domain.Swarm, config domain.GeneralConfig, resDir, simDir string) (string, error) {
	writer := NewResourceWriter(resDir)
	builder := newKubernetesBuilder()

	netsimInstances := normalizeNetsimInstances(config.NetsimInstances)
	netsimAddrs := make([]string, netsimInstances)
	for i := 1; i <= netsimInstances; i++ {
		netsimAddrs[i-1] = fmt.Sprintf("netsim-%d:3000", i)
	}

	gwCfg := LoadRawConfig(o.netsimGatewayConfig)
	gwFile, _ := o.writeTemplateConfig("netsim_gateway_config", o.netsimGatewayConfig, nil, writer)
	telPort, msgPort, subPort := 3000, 3001, 3002
	if p, ok := gwCfg["telemetry_port"].(float64); ok {
		telPort = int(p)
	}
	if p, ok := gwCfg["messages_port"].(float64); ok {
		msgPort = int(p)
	}
	if p, ok := gwCfg["subscribers_port"].(float64); ok {
		subPort = int(p)
	}

	nsOverrides := buildNetsimOverrides(config)
	nsFile, _ := o.writeTemplateConfig("netsim_config", o.netsimConfig, nsOverrides, writer)

	loggerFile, _ := o.writeTemplateConfig("logger_config", o.loggerConfig, nil, writer)

	// We'll collect UAV deployments here
	type uavDep struct {
		name       string
		containers []KubeContainer
	}
	var uavDeployments []uavDep

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
			containers := o.buildKubernetesUAV(swarm.ID, uav, paramFile, arduPilotInstanceFile, writer, config, offsets[i], swarm)
			uavDeployments = append(uavDeployments, uavDep{
				name:       fmt.Sprintf("swarm-%s-uav-%s", swarm.ID, uav.ID),
				containers: containers,
			})
		}
	}

	// Read all generated files from resDir to put into ConfigMap
	configMapData := make(map[string]string)
	if entries, err := os.ReadDir(resDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				b, _ := os.ReadFile(filepath.Join(resDir, e.Name()))
				configMapData[e.Name()] = string(b)
			}
		}
	}
	builder.AddConfigMap("recursos-simulacion", configMapData)

	// Add Gateway
	builder.AddDeployment("netsim-gateway", []KubeContainer{
		{
			Name:  "gateway",
			Image: o.getImageName("netsim_gateway", config.DockerHubUser),
			Ports: []KubePort{
				{ContainerPort: telPort, Protocol: "UDP"},
				{ContainerPort: msgPort, Protocol: "UDP"},
				{ContainerPort: subPort, Protocol: "UDP"},
			},
			Env:          map[string]string{"ADDRS": strings.Join(netsimAddrs, ",")},
			VolumeMounts: []KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: gwFile}},
		},
	}, []string{"recursos-simulacion"})

	// Add Netsims
	for i := 1; i <= netsimInstances; i++ {
		builder.AddDeployment(fmt.Sprintf("netsim-%d", i), []KubeContainer{
			{
				Name:         "netsim",
				Image:        o.getImageName("netsim", config.DockerHubUser),
				Env:          map[string]string{"NODE_ID": fmt.Sprintf("netsim_%d", i)},
				VolumeMounts: []KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: nsFile}},
			},
		}, []string{"recursos-simulacion"})
	}

	// Add Logger
	builder.AddDeployment("logger", []KubeContainer{
		{
			Name:  "logger",
			Image: o.getImageName("logger", config.DockerHubUser),
			Ports: []KubePort{
				{ContainerPort: 5000, Protocol: "UDP"},
				{ContainerPort: 8080, Protocol: "TCP"},
			},
			VolumeMounts: []KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: loggerFile}},
		},
	}, []string{"recursos-simulacion"})

	// Add UAVs
	for _, dep := range uavDeployments {
		builder.AddDeployment(dep.name, dep.containers, []string{"recursos-simulacion"})
	}

	composePath := filepath.Join(simDir, "kubernetes.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

func (o *DockerOrchestrator) getImageName(image string, user string) string {
	if user != "" {
		return fmt.Sprintf("%s/%s", user, image)
	}
	return image
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

func (o *DockerOrchestrator) StartKubernetes(manifestPath, dockerHubUser string) error {
	o.ui.EmitEvent("simulation:log", "Kubernetes deployment logic not yet implemented. The manifest has been generated at: "+manifestPath)
	return nil
}

func (o *DockerOrchestrator) StopKubernetes(simName string) error {
	o.ui.EmitEvent("simulation:log", "Kubernetes stop logic not yet implemented.")
	return nil
}

func (o *DockerOrchestrator) CollectKubernetesLogs(simName, destDir string) error {
	o.ui.EmitEvent("simulation:log", "Kubernetes log collection not yet implemented.")
	return nil
}

func (o *DockerOrchestrator) isLocalhost(host string) bool {
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1")
}

// Ported helpers from ui/internal/simulation/orchestrator.go

func (o *DockerOrchestrator) buildComposeUAV(swarmID string, uav domain.UAV, paramFileName, arduPilotInstanceFile string, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset, swarm domain.Swarm, pool *subnetPool, builder *composeBuilder) []ComposeService {
	nContainers := 4 + len(uav.Services)
	subnet := pool.Next(nContainers)
	uavNet := fmt.Sprintf("swarm_net_%s_uav_%s", swarmID, uav.ID)
	builder.addNetwork(uavNet, subnet)

	var services []ComposeService
	uavEnv := map[string]string{
		"UAV_ID":   uav.ID,
		"SWARM_ID": swarmID,
	}
	if config.VerboseLogging {
		uavEnv["DEBUG"] = "true"
	}

	commName := fmt.Sprintf("swarm_%s_uav_%s_communication_module", swarmID, uav.ID)
	services = append(services, ComposeService{
		Name:          commName,
		Image:         "communication_module",
		ContainerName: commName,
		Environment:   uavEnv,
		Networks:      map[string][]string{uavNet: {"communication_module"}},
	})

	mixer := o.resolveMixer(uav, config)
	appFile, _, appLimits := o.buildServiceResources(mixer, writer)
	mixerName := fmt.Sprintf("swarm_%s_uav_%s_mixer", swarmID, uav.ID)
	services = append(services, ComposeService{
		Name:          mixerName,
		Image:         mixer.FolderName,
		ContainerName: mixerName,
		DependsOn:     []string{commName, fmt.Sprintf("swarm_%s_uav_%s_controller", swarmID, uav.ID)},
		Environment:   uavEnv,
		Volumes:       []string{fmt.Sprintf("./resources/%s:/app/config.json", appFile)},
		Networks:      map[string][]string{uavNet: {"mixer"}},
		Limits:        appLimits,
	})

	controller := o.resolveController(uav, config)
	ucFile, _, ucLimits := o.buildServiceResources(controller, writer)
	var homeLat, homeLon float64
	if uav.HomeOverride != nil {
		homeLat, homeLon = uav.HomeOverride.Lat, uav.HomeOverride.Lon
	} else {
		homeLat, homeLon = util.AddOffset(swarm.FormationCenterLat, swarm.FormationCenterLon, offset.X, offset.Y)
	}
	
	ctrlEnv := map[string]string{
		"UAV_HOME_LOCATION": fmt.Sprintf("%f,%f,0,0", homeLat, homeLon),
		"UAV_ID":            uav.ID,
		"SWARM_ID":          swarmID,
	}
	if config.VerboseLogging {
		ctrlEnv["DEBUG"] = "true"
	}
	if arduPilotInstanceFile != "" {
		ctrlEnv["ARDUPILOT_INSTANCE"] = "/app/" + arduPilotInstanceFile
	}
	
	ctrlMounts := []string{
		fmt.Sprintf("./resources/%s:/app/config.json", ucFile),
		fmt.Sprintf("./resources/%s:/app/copter.parm", paramFileName),
	}
	if arduPilotInstanceFile != "" {
		ctrlMounts = append(ctrlMounts, fmt.Sprintf("./resources/%s:/app/%s", arduPilotInstanceFile, arduPilotInstanceFile))
	}
	if config.LoggingEnabled {
		ctrlMounts = append(ctrlMounts, fmt.Sprintf("./uav_logs/%s/:/app/logs/", uav.ID))
	}
	
	ctrlName := fmt.Sprintf("swarm_%s_uav_%s_controller", swarmID, uav.ID)
	services = append(services, ComposeService{
		Name:          ctrlName,
		Image:         controller.FolderName,
		ContainerName: ctrlName,
		DependsOn:     []string{commName},
		Environment:   ctrlEnv,
		Volumes:       ctrlMounts,
		Networks:      map[string][]string{uavNet: {"uav_controller"}},
		Limits:        ucLimits,
	})

	ecCfg := LoadRawConfig(o.externalCommsConfig)
	ecLimits := ParseResourceLimits(ecCfg)
	ecFile, _ := o.writeTemplateConfig("external_comms_config", o.externalCommsConfig, nil, writer)
	
	ecName := fmt.Sprintf("swarm_%s_uav_%s_external_comms", swarmID, uav.ID)
	services = append(services, ComposeService{
		Name:          ecName,
		Image:         "external_comms",
		ContainerName: ecName,
		DependsOn:     []string{commName, "netsim_gateway"},
		Environment:   uavEnv,
		Volumes:       []string{fmt.Sprintf("./resources/%s:/app/config.json", ecFile)},
		Networks:      map[string][]string{uavNet: {"external_comms"}, "air": nil},
		Limits:        ecLimits,
	})

	for _, svc := range uav.Services {
		svcFile, extraVolumes, algoLimits := o.buildServiceResources(svc, writer)
		
		svcMounts := []string{fmt.Sprintf("./resources/%s:/app/config.json", svcFile)}
		for _, v := range extraVolumes {
			svcMounts = append(svcMounts, fmt.Sprintf("./resources/%s:%s", v.HostPath, v.ContainerPath))
		}
		
		svcName := fmt.Sprintf("swarm_%s_uav_%s_%s", swarmID, uav.ID, svc.ServiceId)
		services = append(services, ComposeService{
			Name:          svcName,
			Image:         svc.ServiceId,
			ContainerName: svcName,
			DependsOn:     []string{commName},
			Environment:   uavEnv,
			Volumes:       svcMounts,
			Networks:      map[string][]string{uavNet: {svc.ServiceId}},
			Limits:        algoLimits,
		})
	}

	return services
}

func (o *DockerOrchestrator) buildKubernetesUAV(swarmID string, uav domain.UAV, paramFileName, arduPilotInstanceFile string, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset, swarm domain.Swarm) []KubeContainer {
	var containers []KubeContainer
	uavEnv := map[string]string{
		"UAV_ID":   uav.ID,
		"SWARM_ID": swarmID,
	}

	containers = append(containers, KubeContainer{
		Name:  "communication-module",
		Image: o.getImageName("communication_module", config.DockerHubUser),
		Env:   uavEnv,
	})

	mixer := o.resolveMixer(uav, config)
	appFile, _, _ := o.buildServiceResources(mixer, writer)
	containers = append(containers, KubeContainer{
		Name:         "mixer",
		Image:        o.getImageName(mixer.FolderName, config.DockerHubUser),
		Env:          uavEnv,
		VolumeMounts: []KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: appFile}},
	})

	controller := o.resolveController(uav, config)
	ucFile, _, _ := o.buildServiceResources(controller, writer)
	var homeLat, homeLon float64
	if uav.HomeOverride != nil {
		homeLat, homeLon = uav.HomeOverride.Lat, uav.HomeOverride.Lon
	} else {
		homeLat, homeLon = util.AddOffset(swarm.FormationCenterLat, swarm.FormationCenterLon, offset.X, offset.Y)
	}
	homeLocation := fmt.Sprintf("%f,%f,0,0", homeLat, homeLon)
	
	ctrlEnv := map[string]string{
		"UAV_HOME_LOCATION": homeLocation,
		"UAV_ID":            uav.ID,
		"SWARM_ID":          swarmID,
	}
	if arduPilotInstanceFile != "" {
		ctrlEnv["ARDUPILOT_INSTANCE"] = "/app/" + arduPilotInstanceFile
	}
	
	ctrlMounts := []KubeVolumeMount{
		{Name: "recursos", MountPath: "/app/config.json", SubPath: ucFile},
		{Name: "recursos", MountPath: "/app/copter.parm", SubPath: paramFileName},
	}
	if arduPilotInstanceFile != "" {
		ctrlMounts = append(ctrlMounts, KubeVolumeMount{Name: "recursos", MountPath: "/app/" + arduPilotInstanceFile, SubPath: arduPilotInstanceFile})
	}
	containers = append(containers, KubeContainer{
		Name:         "uav-controller",
		Image:        o.getImageName(controller.FolderName, config.DockerHubUser),
		Env:          ctrlEnv,
		VolumeMounts: ctrlMounts,
	})

	ecFile, _ := o.writeTemplateConfig("external_comms_config", o.externalCommsConfig, nil, writer)
	containers = append(containers, KubeContainer{
		Name:         "external-comms",
		Image:        o.getImageName("external_comms", config.DockerHubUser),
		Env:          uavEnv,
		VolumeMounts: []KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: ecFile}},
	})

	for _, svc := range uav.Services {
		svcFile, extraVolumes, _ := o.buildServiceResources(svc, writer)
		
		svcMounts := []KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: svcFile}}
		for _, v := range extraVolumes {
			svcMounts = append(svcMounts, KubeVolumeMount{Name: "recursos", MountPath: v.ContainerPath, SubPath: v.HostPath})
		}
		
		containers = append(containers, KubeContainer{
			Name:         svc.ServiceId,
			Image:        o.getImageName(svc.ServiceId, config.DockerHubUser),
			Env:          uavEnv,
			VolumeMounts: svcMounts,
		})
	}
	return containers
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
