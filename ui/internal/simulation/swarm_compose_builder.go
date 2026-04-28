package simulation

import (
	"fmt"
	"strings"
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

// AddNetworkSimulator appends the shared network_simulator service block.
func (b *swarmComposeBuilder) AddNetworkSimulator(verbose bool) {
	env := b.buildEnvBlock("    ", verbose)

	fmt.Fprintf(&b.services, `  network_simulator:
    image: network_simulator
    ports:
      - target: 3000
        published: 3000
        protocol: udp
        mode: ingress
%s    networks:
      - air

`, env)
	b.addNetwork("air")
}

// AddUAVNetwork registers the per-UAV overlay network.
func (b *swarmComposeBuilder) AddUAVNetwork(uavID string) {
	b.addNetwork(uavNetworkName(uavID))
}

// AddCommunicationModule appends the communication_module service for a UAV.
func (b *swarmComposeBuilder) AddCommunicationModule(uavID string, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildEnvBlock("    ", verbose)

	fmt.Fprintf(&b.services, `  communication_module_%s:
    image: communication_module
%s    networks:
      %s:
        aliases:
          - communication_module

`, uavID, env, uavNet)
}

// AddApplication appends the application service for a UAV.
func (b *swarmComposeBuilder) AddApplication(uavID, configFileName string, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildEnvBlock("    ", verbose)
	configName := b.declareConfig(configFileName)

	fmt.Fprintf(&b.services, `  application_%s:
    image: application
%s    configs:
      - source: %s
        target: /app/config.json
    deploy:
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s
    networks:
      %s:
        aliases:
          - application

`, uavID, env, configName, uavNet)
}

// AddUAVController appends the uav_controller (SITL) service for a UAV.
func (b *swarmComposeBuilder) AddUAVController(uavID, configFileName, paramFileName, homeLocation string, verbose bool) {
	uavNet := uavNetworkName(uavID)
	configName := b.declareConfig(configFileName)
	paramName := b.declareConfig(paramFileName)

	envLines := fmt.Sprintf("    environment:\n      - UAV_HOME_LOCATION=%s\n", homeLocation)
	if verbose {
		envLines += "      - DEBUG=true\n"
	}

	fmt.Fprintf(&b.services, `  uav_controller_%s:
    image: copter453
%s    configs:
      - source: %s
        target: /app/config.json
      - source: %s
        target: /app/copter.parm
    networks:
      %s:
        aliases:
          - uav_controller

`, uavID, envLines, configName, paramName, uavNet)
}

// AddExternalComms appends the external_comms service for a UAV.
// This service bridges the per-UAV network and the shared air network.
func (b *swarmComposeBuilder) AddExternalComms(uavID, configFileName string, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildEnvBlock("    ", verbose)
	configName := b.declareConfig(configFileName)

	fmt.Fprintf(&b.services, `  external_comms_%s:
    image: external_comms
%s    configs:
      - source: %s
        target: /app/config.json
    deploy:
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s
    networks:
      %s:
        aliases:
          - external_comms
      air:

`, uavID, env, configName, uavNet)
}

// AddAlgorithmService appends a user-deployed algorithm service for a UAV.
// Extra volumes (e.g. KML files) are also promoted to Docker configs.
func (b *swarmComposeBuilder) AddAlgorithmService(uavID string, svc DeployedService, configFileName string, extraVolumes []VolumeMount, verbose bool) {
	uavNet := uavNetworkName(uavID)
	env := b.buildEnvBlock("    ", verbose)
	configName := b.declareConfig(configFileName)

	var configMounts strings.Builder
	fmt.Fprintf(&configMounts, "      - source: %s\n        target: /app/config.json\n", configName)
	for _, v := range extraVolumes {
		extraName := b.declareConfig(v.HostPath)
		fmt.Fprintf(&configMounts, "      - source: %s\n        target: %s\n", extraName, v.ContainerPath)
	}

	fmt.Fprintf(&b.services, `  %s_%s:
    image: %s
%s    configs:
%s    networks:
      %s:
        aliases:
          - %s
    deploy:
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s

`, svc.ServiceId, uavID,
		svc.ServiceId,
		env,
		configMounts.String(),
		uavNet,
		svc.ServiceId)
}

// Build returns the final YAML document as a string.
func (b *swarmComposeBuilder) Build() string {
	return b.services.String() + "\n" + b.networks.String() + "\n" + b.configs.String()
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
