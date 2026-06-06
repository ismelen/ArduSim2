package docker

import (
	"fmt"
	"strings"
	"ui/internal/domain"
)

// composeBuilder assembles a Docker Compose YAML document incrementally.
// Each method appends a well-scoped block, keeping the building logic DRY
// and each service block easy to reason about in isolation.
type composeBuilder struct {
	services strings.Builder
	networks strings.Builder
}

func newComposeBuilder() *composeBuilder {
	b := &composeBuilder{}
	b.services.WriteString("services:\n")
	b.networks.WriteString("networks:\n")
	return b
}

// AddNetsimGateway appends the shared netsim_gateway service block.
func (b *composeBuilder) AddNetsimGateway(configFileName string, netsimAddrs []string, limits ResourceLimits, verbose bool, telemetryPort, msgPort, subPort int) {
	var env string
	if verbose {
		env = "    environment:\n      - DEBUG=true\n"
	}
	
	if len(netsimAddrs) > 0 {
		addrsStr := strings.Join(netsimAddrs, ",")
		if env == "" {
			env = fmt.Sprintf("    environment:\n      - ADDRS=%s\n", addrsStr)
		} else {
			env += fmt.Sprintf("      - ADDRS=%s\n", addrsStr)
		}
	}

	vols := fmt.Sprintf("    volumes:\n      - ./resources/%s:/app/config.json\n", configFileName)
	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  netsim_gateway:
    image: netsim_gateway
    container_name: netsim_gateway
    extra_hosts:
      - "host.docker.internal:host-gateway"
    ports:
      - %d:%d/udp
      - %d:%d/udp
      - %d:%d/udp
%s%s%s    networks:
      - air

`, telemetryPort, telemetryPort, msgPort, msgPort, subPort, subPort, env, vols, lims)
	b.addNetwork("air", "10.9.0.0/24")
}

// AddNetsim appends a netsim worker service block.
func (b *composeBuilder) AddNetsim(instanceID int, configFileName string, limits ResourceLimits, verbose bool) {
	var env string
	if verbose {
		env = fmt.Sprintf("    environment:\n      - DEBUG=true\n      - NODE_ID=netsim_%d\n", instanceID)
	} else {
		env = fmt.Sprintf("    environment:\n      - NODE_ID=netsim_%d\n", instanceID)
	}

	vols := fmt.Sprintf("    volumes:\n      - ./resources/%s:/app/config.json\n", configFileName)
	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  netsim_%d:
    image: netsim
    container_name: netsim_%d
    depends_on:
      - netsim_gateway
%s%s%s    networks:
      - air

`, instanceID, instanceID, env, vols, lims)
}

// AddUAVNetwork registers the per-UAV bridge network.
func (b *composeBuilder) AddUAVNetwork(swarmID, uavID string, subnet string) {
	b.addNetwork(uavNetworkName(swarmID, uavID), subnet)
}

// AddCommunicationModule appends the communication_module service for a UAV.
func (b *composeBuilder) AddCommunicationModule(swarmID, uavID string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)
	
	env := fmt.Sprintf("    environment:\n      - UAV_ID=%s\n      - SWARM_ID=%s\n", uavID, swarmID)
	if verbose {
		env += "      - DEBUG=true\n"
	}
	
	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_communication_module:
    image: communication_module
    container_name: swarm_%s_uav_%s_communication_module
%s%s    networks:
      %s:
        aliases:
          - communication_module

`, swarmID, uavID, swarmID, uavID, env, lims, uavNet)
}

// AddMixer appends the mixer service for a UAV.
func (b *composeBuilder) AddMixer(swarmID, uavID, image, configFileName string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)

	env := fmt.Sprintf("    environment:\n      - UAV_ID=%s\n      - SWARM_ID=%s\n", uavID, swarmID)
	if verbose {
		env += "      - DEBUG=true\n"
	}

	vols := fmt.Sprintf("      - ./resources/%s:/app/config.json\n", configFileName)
	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_mixer:
    image: %s
    container_name: swarm_%s_uav_%s_mixer
    depends_on:
      - swarm_%s_uav_%s_communication_module
      - swarm_%s_uav_%s_controller
%s    volumes:
%s%s    networks:
      %s:
        aliases:
          - mixer

`, swarmID, uavID, image, swarmID, uavID, swarmID, uavID, swarmID, uavID, env, vols, lims, uavNet)
}

// AddUAVController appends the uav_controller (SITL) service for a UAV.
func (b *composeBuilder) AddUAVController(swarmID, uavID, controllerFolderName, configFileName, paramFileName, homeLocation, arduPilotInstanceFile string, limits ResourceLimits, verbose bool, loggingEnabled bool) {
	uavNet := uavNetworkName(swarmID, uavID)

	var env string
	if verbose {
		env = "      - DEBUG=true\n"
	}
	if arduPilotInstanceFile != "" {
		env += fmt.Sprintf("      - ARDUPILOT_INSTANCE=/app/%s\n", arduPilotInstanceFile)
	}

	vols := fmt.Sprintf("      - ./resources/%s:/app/config.json\n", configFileName)
	vols += fmt.Sprintf("      - ./resources/%s:/app/copter.parm\n", paramFileName)
	if arduPilotInstanceFile != "" {
		vols += fmt.Sprintf("      - ./resources/%s:/app/%s\n", arduPilotInstanceFile, arduPilotInstanceFile)
	}
	if loggingEnabled {
		vols += fmt.Sprintf("      - ./uav_logs/%s/:/app/logs/\n", uavID)
	}
	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_controller:
    image: %s
    container_name: swarm_%s_uav_%s_controller
    depends_on:
      - swarm_%s_uav_%s_communication_module
    environment:
      - UAV_HOME_LOCATION=%s
      - UAV_ID=%s
      - SWARM_ID=%s
%s    volumes:
%s%s    networks:
      %s:
        aliases:
          - uav_controller

`, swarmID, uavID, controllerFolderName, swarmID, uavID, swarmID, uavID, homeLocation, uavID, swarmID, env, vols, lims, uavNet)
}

// AddExternalComms appends the external_comms service for a UAV.
// This service bridges the per-UAV network and the shared air network.
func (b *composeBuilder) AddExternalComms(swarmID, uavID, configFileName string, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)

	env := fmt.Sprintf("    environment:\n      - UAV_ID=%s\n      - SWARM_ID=%s\n", uavID, swarmID)
	if verbose {
		env += "      - DEBUG=true\n"
	}

	vols := fmt.Sprintf("      - ./resources/%s:/app/config.json\n", configFileName)
	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_external_comms:
    image: external_comms
    container_name: swarm_%s_uav_%s_external_comms
    depends_on:
      - swarm_%s_uav_%s_communication_module
      - netsim_gateway
%s    volumes:
%s%s    networks:
      %s:
        aliases:
          - external_comms
      air:

`, swarmID, uavID, swarmID, uavID, swarmID, uavID, env, vols, lims, uavNet)
}

// AddAlgorithmService appends a user-deployed algorithm service for a UAV.
func (b *composeBuilder) AddAlgorithmService(swarmID, uavID string, svc domain.DeployedService, configFileName string, extraVolumes []domain.VolumeMount, limits ResourceLimits, verbose bool) {
	uavNet := uavNetworkName(swarmID, uavID)

	env := fmt.Sprintf("    environment:\n      - UAV_ID=%s\n      - SWARM_ID=%s\n", uavID, swarmID)
	if verbose {
		env += "      - DEBUG=true\n"
	}

	var vols strings.Builder
	fmt.Fprintf(&vols, "      - ./resources/%s:/app/config.json\n", configFileName)
	for _, v := range extraVolumes {
		fmt.Fprintf(&vols, "      - ./resources/%s:%s\n", v.HostPath, v.ContainerPath)
	}

	lims := b.buildLocalLimits(limits)

	fmt.Fprintf(&b.services, `  swarm_%s_uav_%s_%s:
    image: %s
    container_name: swarm_%s_uav_%s_%s
    depends_on:
      - swarm_%s_uav_%s_communication_module
%s    volumes:
%s%s    networks:
      %s:
        aliases:
          - %s

`, swarmID, uavID, svc.ServiceId,
		svc.ServiceId,
		swarmID, uavID, svc.ServiceId,
		swarmID, uavID,
		env,
		vols.String(),
		lims,
		uavNet,
		svc.ServiceId)
}

// Build returns the final YAML document as a string.
func (b *composeBuilder) Build() string {
	return b.services.String() + "\n" + b.networks.String()
}

// AddLogger appends the central logger service.
func (b *composeBuilder) AddLogger(configFileName string, limits ResourceLimits) {
	lims := b.buildLocalLimits(limits)
	vols := fmt.Sprintf("    volumes:\n      - ./resources/%s:/app/config.json\n", configFileName)

	fmt.Fprintf(&b.services, `  logger:
    image: logger
    container_name: logger
    ports:
      - 5000:5000/udp
      - 8080:8080/tcp
%s%s    networks:
      - air

`, vols, lims)
}

// addNetwork appends a bridge network definition if not already present.
// Multiple calls with the same name are intentionally idempotent — the YAML
// key would duplicate, so callers are responsible for calling once per name.
func (b *composeBuilder) addNetwork(name string, subnet string) {
	if subnet != "" {
		fmt.Fprintf(&b.networks, "  %s:\n    driver: bridge\n    ipam:\n      config:\n        - subnet: %s\n", name, subnet)
	} else {
		fmt.Fprintf(&b.networks, "  %s:\n    driver: bridge\n", name)
	}
}

func uavNetworkName(swarmID, uavID string) string {
	return fmt.Sprintf("swarm_net_%s_uav_%s", swarmID, uavID)
}

func (b *composeBuilder) buildLocalLimits(limits ResourceLimits) string {
	if !limits.HasLimits() {
		return ""
	}
	var bldr strings.Builder
	if limits.RAM != "" {
		fmt.Fprintf(&bldr, "    mem_limit: %s\n", limits.RAM)
	}
	if limits.CPU > 0 {
		fmt.Fprintf(&bldr, "    cpus: \"%.2f\"\n", limits.CPU)
	}
	return bldr.String()
}
