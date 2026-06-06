package docker

import (
	"fmt"
	"strings"
	"ui/internal/domain"
)

// swarmComposeBuilder assembles a Docker Stack (Swarm-compatible) YAML document.
//
// Key differences from composeBuilder:
//   - Networks use the "overlay" driver for cross-node communication.
//   - No container_name (Swarm manages service replicas).
//   - No depends_on (not supported in Swarm mode).
//   - Config files are declared as Docker Swarm "configs" and mounted read-only
//     into each container instead of using local bind-mounts.
//   - Local log directories (bind-mounts of host paths) are omitted.
//   - Images are referenced by name only — build: blocks are excluded because
//     images must already be built and available on the Swarm nodes.
type swarmComposeBuilder struct {
	services strings.Builder
	networks strings.Builder
	configs  strings.Builder
	// configNames tracks which configs have already been declared to avoid duplicates.
	configNames map[string]bool
}

func newSwarmComposeBuilder() *swarmComposeBuilder {
	b := &swarmComposeBuilder{
		configNames: make(map[string]bool),
	}
	b.services.WriteString("services:\n")
	b.networks.WriteString("networks:\n")
	b.configs.WriteString("configs:\n")
	return b
}

// AddNetsimGateway appends the shared netsim_gateway service block.
func (b *swarmComposeBuilder) AddNetsimGateway(configFileName string, netsimAddrs []string, limits ResourceLimits, verbose bool, telemetryPort, msgPort, subPort int) {
	env := b.buildEnvBlock("    ", verbose)
	if len(netsimAddrs) > 0 {
		addrsStr := strings.Join(netsimAddrs, ",")
		if env == "" {
			env = fmt.Sprintf("    environment:\n      - ADDRS=%s\n", addrsStr)
		} else {
			env += fmt.Sprintf("      - ADDRS=%s\n", addrsStr)
		}
	}
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, "")

	fmt.Fprintf(&b.services, `  netsim_gateway:
    image: netsim_gateway
    ports:
      - target: %d
        published: %d
        protocol: udp
        mode: ingress
      - target: %d
        published: %d
        protocol: udp
        mode: ingress
      - target: %d
        published: %d
        protocol: udp
        mode: ingress
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      - air

`, telemetryPort, telemetryPort, msgPort, msgPort, subPort, subPort, env, configName, lims)
	b.addNetwork("air")
}

// AddNetsim appends a netsim worker service block.
func (b *swarmComposeBuilder) AddNetsim(instanceID int, configFileName string, limits ResourceLimits, verbose bool) {
	var env string
	if verbose {
		env = fmt.Sprintf("    environment:\n      - DEBUG=true\n      - NODE_ID=netsim_%d\n", instanceID)
	} else {
		env = fmt.Sprintf("    environment:\n      - NODE_ID=netsim_%d\n", instanceID)
	}
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, "")

	fmt.Fprintf(&b.services, `  netsim_%d:
    image: netsim
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      - air

`, instanceID, env, configName, lims)
}

// AddUAVNetwork registers the per-UAV overlay network.
func (b *swarmComposeBuilder) AddUAVNetwork(swarmID, uavID string) {
	b.addNetwork(uavNetworkName(swarmID, uavID))
}

// AddCommunicationModule appends the communication_module service for a UAV.
func (b *swarmComposeBuilder) AddCommunicationModule(swarmID, uavID string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)
	env := b.buildUAVEnvBlock("    ", swarmID, uavID, verbose)
	lims := b.buildSwarmDeployBlock(limits, "")

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_communication_module:
    image: communication_module
%s%s    networks:
      %s:
        aliases:
          - communication_module

`, swarmID, uavID, env, lims, uavNet)
}

// AddMixer appends the mixer service for a UAV in swarm mode.
func (b *swarmComposeBuilder) AddMixer(swarmID, uavID, image, configFileName string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)
	env := b.buildUAVEnvBlock("    ", swarmID, uavID, verbose)
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, `      restart_policy:
        condition: on-failure
        delay: "5s"
        max_attempts: 20
        window: "120s"
`)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_mixer:
    image: 127.0.0.1:5000/%s
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      %s:
        aliases:
          - mixer

`, swarmID, uavID, image, env, configName, lims, uavNet)
}

func (b *swarmComposeBuilder) AddUAVController(swarmID, uavID, controllerFolderName, configFileName, paramFileName, homeLocation, arduPilotInstanceFile string, limits ResourceLimits, verbose bool, loggingEnabled bool) {
	uavNet := uavNetworkName(swarmID, uavID)
	configName := b.declareConfig(configFileName)
	paramName := b.declareConfig(paramFileName)

	var arduPilotConfig string
	if arduPilotInstanceFile != "" {
		arduPilotConfig = b.declareConfig(arduPilotInstanceFile)
	}

	envLines := fmt.Sprintf("    environment:\n      - UAV_HOME_LOCATION=%s\n      - UAV_ID=%s\n      - SWARM_ID=%s\n", homeLocation, uavID, swarmID)
	if verbose {
		envLines += "      - DEBUG=true\n"
	}
	if arduPilotInstanceFile != "" {
		envLines += fmt.Sprintf("      - ARDUPILOT_INSTANCE=/app/%s\n", arduPilotInstanceFile)
	}
	lims := b.buildSwarmDeployBlock(limits, "")

	var extraConfigs string
	if arduPilotInstanceFile != "" {
		extraConfigs = fmt.Sprintf("      - source: %s\n        target: /app/%s\n", arduPilotConfig, arduPilotInstanceFile)
	}

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_controller:
    image: 127.0.0.1:5000/%s
%s    configs:
      - source: %s
        target: /app/config.json
      - source: %s
        target: /app/copter.parm
%s%s    networks:
      %s:
        aliases:
          - uav_controller

`, swarmID, uavID, controllerFolderName, envLines, configName, paramName, extraConfigs, lims, uavNet)
}

// AddExternalComms appends the external_comms service for a UAV.
// This service bridges the per-UAV network and the shared air network.
func (b *swarmComposeBuilder) AddExternalComms(swarmID, uavID, configFileName string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)
	env := b.buildUAVEnvBlock("    ", swarmID, uavID, verbose)
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, `      restart_policy:
        condition: on-failure
        delay: "5s"
        max_attempts: 20
        window: "120s"
`)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_external_comms:
    image: external_comms
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      %s:
        aliases:
          - external_comms
      air:

`, swarmID, uavID, env, configName, lims, uavNet)
}

// AddAlgorithmService appends a user-deployed algorithm service for a UAV.
// Extra volumes (e.g. KML files) are also promoted to Docker configs.
func (b *swarmComposeBuilder) AddAlgorithmService(swarmID, uavID string, svc domain.DeployedService, configFileName string, extraVolumes []domain.VolumeMount, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)
	env := b.buildUAVEnvBlock("    ", swarmID, uavID, verbose)
	configName := b.declareConfig(configFileName)

	var configMounts strings.Builder
	fmt.Fprintf(&configMounts, "      - source: %s\n        target: /app/config.json\n", configName)
	for _, v := range extraVolumes {
		extraName := b.declareConfig(v.HostPath)
		fmt.Fprintf(&configMounts, "      - source: %s\n        target: %s\n", extraName, v.ContainerPath)
	}
	
	lims := b.buildSwarmDeployBlock(limits, `      restart_policy:
        condition: on-failure
        delay: "5s"
        max_attempts: 20
        window: "120s"
`)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_%s:
    image: %s
%s    configs:
%s    networks:
      %s:
        aliases:
          - %s
%s
`, swarmID, uavID, svc.ServiceId,
		svc.ServiceId,
		env,
		configMounts.String(),
		uavNet,
		svc.ServiceId,
		strings.TrimSuffix(lims, "\n")) // trim trailing newline because template handles spacing
}

// Build returns the final YAML document as a string.
func (b *swarmComposeBuilder) Build() string {
	return b.services.String() + "\n" + b.networks.String() + "\n" + b.configs.String()
}

// AddLogger appends the central logger service.
func (b *swarmComposeBuilder) AddLogger(configFileName string, limits ResourceLimits) {
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, "")

	fmt.Fprintf(&b.services, `  logger:
    image: logger
%s    ports:
      - target: 5000
        published: 5000
        protocol: udp
        mode: host
      - target: 8080
        published: 8080
        protocol: tcp
        mode: host
    configs:
      - source: %s
        target: /app/config.json
    networks:
      - air

`, lims, configName)
}

// declareConfig registers a file as a Docker Swarm config (idempotent) and
// returns the config name to use in service definitions.
// Config names are derived from the file name with dots replaced by underscores
// to satisfy Docker naming constraints.
func (b *swarmComposeBuilder) declareConfig(fileName string) string {
	name := strings.ReplaceAll(fileName, ".", "_")
	if !b.configNames[name] {
		b.configNames[name] = true
		// "file:" path is relative to the compose file location (simDir/resources/).
		fmt.Fprintf(&b.configs, "  %s:\n    file: ./resources/%s\n", name, fileName)
	}
	return name
}

// addNetwork appends an overlay network definition.
func (b *swarmComposeBuilder) addNetwork(name string) {
	fmt.Fprintf(&b.networks, "  %s:\n    driver: overlay\n    attachable: true\n", name)
}

// buildEnvBlock returns a YAML environment block indented with the given prefix,
// or an empty string when no environment variables are needed.
func (b *swarmComposeBuilder) buildEnvBlock(indent string, verbose bool) string {
	if !verbose {
		return ""
	}
	return indent + "environment:\n" + indent + "  - DEBUG=true\n"
}

func (b *swarmComposeBuilder) buildUAVEnvBlock(indent string, swarmID, uavID string, verbose bool) string {
	env := indent + "environment:\n" + indent + "  - UAV_ID=" + uavID + "\n" + indent + "  - SWARM_ID=" + swarmID + "\n"
	if verbose {
		env += indent + "  - DEBUG=true\n"
	}
	return env
}

func (b *swarmComposeBuilder) buildSwarmDeployBlock(limits ResourceLimits, existingRules string) string {
	if !limits.HasLimits() && existingRules == "" {
		return ""
	}

	var bldr strings.Builder
	bldr.WriteString("    deploy:\n")
	if existingRules != "" {
		bldr.WriteString(existingRules)
	}

	if limits.HasLimits() {
		bldr.WriteString("      resources:\n        limits:\n")
		if limits.RAM != "" {
			fmt.Fprintf(&bldr, "          memory: %s\n", limits.RAM)
		}
		if limits.CPU > 0 {
			fmt.Fprintf(&bldr, "          cpus: '%.2f'\n", limits.CPU)
		}
	}
	return bldr.String()
}
