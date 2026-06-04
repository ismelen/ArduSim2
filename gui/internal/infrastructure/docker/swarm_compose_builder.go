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
func (b *swarmComposeBuilder) AddNetsimGateway(configFileName string, netsimAddrs []string, limits ResourceLimits, verbose bool) {
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
      - target: 3000
        published: 3000
        protocol: udp
        mode: ingress
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      - air

`, env, configName, lims)
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
func (b *swarmComposeBuilder) AddUAVNetwork(uavID string) {
	b.addNetwork(uavNetworkName(uavID))
}

// AddCommunicationModule appends the communication_module service for a UAV.
func (b *swarmComposeBuilder) AddCommunicationModule(uavID string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildUAVEnvBlock("    ", uavID, verbose)
	lims := b.buildSwarmDeployBlock(limits, "")

	fmt.Fprintf(&b.services, `  communication_module_%s:
    image: communication_module
%s%s    networks:
      %s:
        aliases:
          - communication_module

`, uavID, env, lims, uavNet)
}

// AddApplication appends the application service for a UAV.
func (b *swarmComposeBuilder) AddApplication(uavID, configFileName string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildUAVEnvBlock("    ", uavID, verbose)
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, `      restart_policy:
        condition: on-failure
        delay: "5s"
        max_attempts: 20
        window: "120s"
`)

	fmt.Fprintf(&b.services, `  application_%s:
    image: application
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      %s:
        aliases:
          - application

`, uavID, env, configName, lims, uavNet)
}

func (b *swarmComposeBuilder) AddUAVController(uavID, configFileName, paramFileName, homeLocation string, limits ResourceLimits, verbose bool, loggingEnabled bool) {
	uavNet := uavNetworkName(uavID)
	configName := b.declareConfig(configFileName)
	paramName := b.declareConfig(paramFileName)

	envLines := fmt.Sprintf("    environment:\n      - UAV_HOME_LOCATION=%s\n      - UAV_ID=%s\n", homeLocation, uavID)
	if verbose {
		envLines += "      - DEBUG=true\n"
	}
	lims := b.buildSwarmDeployBlock(limits, "")

	fmt.Fprintf(&b.services, `  uav_controller_%s:
    image: copter453
%s    configs:
      - source: %s
        target: /app/config.json
      - source: %s
        target: /app/copter.parm
%s    networks:
      %s:
        aliases:
          - uav_controller

`, uavID, envLines, configName, paramName, lims, uavNet)
}

// AddExternalComms appends the external_comms service for a UAV.
// This service bridges the per-UAV network and the shared air network.
func (b *swarmComposeBuilder) AddExternalComms(uavID, configFileName string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildUAVEnvBlock("    ", uavID, verbose)
	configName := b.declareConfig(configFileName)
	lims := b.buildSwarmDeployBlock(limits, `      restart_policy:
        condition: on-failure
        delay: "5s"
        max_attempts: 20
        window: "120s"
`)

	fmt.Fprintf(&b.services, `  external_comms_%s:
    image: external_comms
%s    configs:
      - source: %s
        target: /app/config.json
%s    networks:
      %s:
        aliases:
          - external_comms
      air:

`, uavID, env, configName, lims, uavNet)
}

// AddAlgorithmService appends a user-deployed algorithm service for a UAV.
// Extra volumes (e.g. KML files) are also promoted to Docker configs.
func (b *swarmComposeBuilder) AddAlgorithmService(uavID string, svc domain.DeployedService, configFileName string, extraVolumes []domain.VolumeMount, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildUAVEnvBlock("    ", uavID, verbose)
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

	fmt.Fprintf(&b.services, `  %s_%s:
    image: %s
%s    configs:
%s    networks:
      %s:
        aliases:
          - %s
%s
`, svc.ServiceId, uavID,
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
func (b *swarmComposeBuilder) AddLogger(limits ResourceLimits) {
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
    networks:
      - air

`, lims)
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

func (b *swarmComposeBuilder) buildUAVEnvBlock(indent string, uavID string, verbose bool) string {
	env := indent + "environment:\n" + indent + "  - UAV_ID=" + uavID + "\n"
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
