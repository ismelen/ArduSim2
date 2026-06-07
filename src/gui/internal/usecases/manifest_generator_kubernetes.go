package usecases

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ui/internal/domain"
	"ui/internal/domain/formation"
	"ui/internal/infrastructure/util"
	"ui/internal/ports"
)

func (g *ManifestGenerator) buildKubernetesManifests(swarms []domain.Swarm, config domain.GeneralConfig, resDir, simDir string) (string, error) {
	writer := NewResourceWriter(resDir)
	builder := g.newKubernetesBuilder()

	netsimInstances := normalizeNetsimInstances(config.NetsimInstances)
	netsimAddrs := make([]string, netsimInstances)
	for i := 1; i <= netsimInstances; i++ {
		netsimAddrs[i-1] = fmt.Sprintf("netsim-%d:3000", i)
	}

	gwCfg := g.LoadRawConfig(g.netsimGatewayConfig)
	gwFile, _ := g.writeTemplateConfig("netsim_gateway_config", g.netsimGatewayConfig, nil, writer)
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

	nsOverrides := g.buildNetsimOverrides(config)
	nsFile, _ := g.writeTemplateConfig("netsim_config", g.netsimConfig, nsOverrides, writer)

	loggerFile, _ := g.writeTemplateConfig("logger_config", g.loggerConfig, nil, writer)

	type uavDep struct {
		name       string
		containers []ports.KubeContainer
		volumes    []ports.KubeVolume
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
			var arduPilotHostPath string
			if arduPilotInstance != "" {
				absPath, err := filepath.Abs(arduPilotInstance)
				if err == nil {
					arduPilotHostPath = absPath
				} else {
					arduPilotHostPath = arduPilotInstance
				}
			}

			paramFile, _ := g.generateUAVParams(swarm.ID, uav, config, resDir)
			containers, volumes := g.buildKubernetesUAV(swarm.ID, uav, paramFile, arduPilotHostPath, writer, config, offsets[i], swarm)
			uavDeployments = append(uavDeployments, uavDep{
				name:       fmt.Sprintf("swarm-%s-uav-%s", swarm.ID, uav.ID),
				containers: containers,
				volumes:    volumes,
			})
		}
	}

	// We no longer generate a ConfigMap to keep the YAML clean.
	// Instead, we will mount the entire 'resources' directory using a hostPath volume.

	builder.AddDeployment("netsim-gateway", []ports.KubeContainer{
		{
			Name:  "gateway",
			Image: g.getImageName("netsim_gateway", config.DockerHubRepository),
			Ports: []ports.KubePort{
				{ContainerPort: telPort, Protocol: "UDP"},
				{ContainerPort: msgPort, Protocol: "UDP"},
				{ContainerPort: subPort, Protocol: "UDP"},
			},
			Env:          map[string]string{"ADDRS": strings.Join(netsimAddrs, ",")},
			VolumeMounts: []ports.KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: gwFile}},
		},
	}, []ports.KubeVolume{{Name: "recursos", HostPath: resDir, Type: "Directory"}})

	for i := 1; i <= netsimInstances; i++ {
		builder.AddDeployment(fmt.Sprintf("netsim-%d", i), []ports.KubeContainer{
			{
				Name:         "netsim",
				Image:        g.getImageName("netsim", config.DockerHubRepository),
				Env:          map[string]string{"NODE_ID": fmt.Sprintf("netsim_%d", i)},
				VolumeMounts: []ports.KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: nsFile}},
			},
		}, []ports.KubeVolume{{Name: "recursos", HostPath: resDir, Type: "Directory"}})
	}

	builder.AddDeployment("logger", []ports.KubeContainer{
		{
			Name:  "logger",
			Image: g.getImageName("logger", config.DockerHubRepository),
			Ports: []ports.KubePort{
				{ContainerPort: 5000, Protocol: "UDP"},
				{ContainerPort: 8080, Protocol: "TCP"},
			},
			VolumeMounts: []ports.KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: loggerFile}},
		},
	}, []ports.KubeVolume{{Name: "recursos", HostPath: resDir, Type: "Directory"}})

	for _, dep := range uavDeployments {
		builder.AddDeployment(dep.name, dep.containers, dep.volumes)
	}

	composePath := filepath.Join(simDir, "kubernetes.yaml")
	_ = os.WriteFile(composePath, []byte(builder.Build()), 0644)
	return composePath, nil
}

func (g *ManifestGenerator) buildKubernetesUAV(swarmID string, uav domain.UAV, paramFileName, arduPilotHostPath string, writer *ResourceWriter, config domain.GeneralConfig, offset formation.Offset, swarm domain.Swarm) ([]ports.KubeContainer, []ports.KubeVolume) {
	var containers []ports.KubeContainer
	uavEnv := map[string]string{
		"UAV_ID":   uav.ID,
		"SWARM_ID": swarmID,
	}

	containers = append(containers, ports.KubeContainer{
		Name:  "communication-module",
		Image: g.getImageName("communication_module", config.DockerHubRepository),
		Env:   uavEnv,
	})

	mixer := g.resolveMixer(uav, config)
	appFile, _, _ := g.buildServiceResources(mixer, writer)
	containers = append(containers, ports.KubeContainer{
		Name:         "mixer",
		Image:        g.getImageName(mixer.FolderName, config.DockerHubRepository),
		Env:          uavEnv,
		VolumeMounts: []ports.KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: appFile}},
	})

	ucCfg := g.LoadRawConfig(filepath.Join(g.projectRoot, "..", "uav_controller", "ardupilot4_5_3", "config.sitl.json"))
	ucFile, _ := writer.Write(uav.ID+"_uav_controller_config", ucCfg)
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
	if arduPilotHostPath != "" {
		ctrlEnv["ARDUPILOT_INSTANCE"] = "/app/custom_arducopter"
	}
	
	ctrlMounts := []ports.KubeVolumeMount{
		{Name: "recursos", MountPath: "/app/config.json", SubPath: ucFile},
		{Name: "recursos", MountPath: "/app/copter.parm", SubPath: paramFileName},
	}
	if arduPilotHostPath != "" {
		ctrlMounts = append(ctrlMounts, ports.KubeVolumeMount{Name: "custom-binary", MountPath: "/app/custom_arducopter"})
	}
	containers = append(containers, ports.KubeContainer{
		Name:         "uav-controller",
		Image:        g.getImageName("uav_controller", config.DockerHubRepository),
		Env:          ctrlEnv,
		VolumeMounts: ctrlMounts,
	})

	ecFile, _ := g.writeTemplateConfig("external_comms_config", g.externalCommsConfig, nil, writer)
	containers = append(containers, ports.KubeContainer{
		Name:         "external-comms",
		Image:        g.getImageName("external_comms", config.DockerHubRepository),
		Env:          uavEnv,
		VolumeMounts: []ports.KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: ecFile}},
	})

	for _, svc := range uav.Services {
		svcFile, extraVolumes, _ := g.buildServiceResources(svc, writer)
		
		svcMounts := []ports.KubeVolumeMount{{Name: "recursos", MountPath: "/app/config.json", SubPath: svcFile}}
		for _, v := range extraVolumes {
			svcMounts = append(svcMounts, ports.KubeVolumeMount{Name: "recursos", MountPath: v.ContainerPath, SubPath: v.HostPath})
		}
		
		containers = append(containers, ports.KubeContainer{
			Name:         svc.ServiceId,
			Image:        g.getImageName(svc.ServiceId, config.DockerHubRepository),
			Env:          uavEnv,
			VolumeMounts: svcMounts,
		})
	}
	
	volumes := []ports.KubeVolume{
		{Name: "recursos", HostPath: writer.outputDir, Type: "Directory"},
	}
	if arduPilotHostPath != "" {
		volumes = append(volumes, ports.KubeVolume{
			Name:     "custom-binary",
			HostPath: arduPilotHostPath,
			Type:     "File",
		})
	}

	return containers, volumes
}
