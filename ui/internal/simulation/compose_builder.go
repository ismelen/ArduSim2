package simulation

import (
	"fmt"
	"strings"
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

// AddNetworkSimulator appends the shared network_simulator service block.
func (b *composeBuilder) AddNetworkSimulator(logDir string, verbose bool) {
	var env string
	if verbose {
		env = "    environment:\n      - DEBUG=true\n"
	}

	var vols string
	if logDir != "" {
		vols = fmt.Sprintf("    volumes:\n      - %s:/app/logs\n", logDir)
	}

	fmt.Fprintf(&b.services, `  network_simulator:
    image: network_simulator
    build:
      context: ../../network_simulator
      dockerfile: Dockerfile
    container_name: network_simulator
    extra_hosts:
      - "host.docker.internal:host-gateway"
    ports:
      - 3000:3000/udp
%s%s    networks:
      - air

`, env, vols)
	b.addNetwork("air")
}

// AddUAVNetwork registers the per-UAV bridge network.
func (b *composeBuilder) AddUAVNetwork(uavID string) {
	b.addNetwork(uavNetworkName(uavID))
}

// AddCommunicationModule appends the communication_module service for a UAV.
func (b *composeBuilder) AddCommunicationModule(uavID, logDir string, verbose bool) {
	uavNet := uavNetworkName(uavID)
	
	var env string
	if verbose {
		env = "    environment:\n      - DEBUG=true\n"
	}

	var vols string
	if logDir != "" {
		vols = fmt.Sprintf("    volumes:\n      - %s:/app/logs\n", logDir)
	}

	fmt.Fprintf(&b.services, `  communication_module_%s:
    image: communication_module
    build:
      context: ../../communication_module
      dockerfile: Dockerfile
    container_name: communication_module_%s
%s%s    networks:
      %s:
        aliases:
          - communication_module

`, uavID, uavID, env, vols, uavNet)
}

// AddApplication appends the application service for a UAV.
func (b *composeBuilder) AddApplication(uavID, configFileName, logDir string, verbose bool) {
	uavNet := uavNetworkName(uavID)

	var env string
	if verbose {
		env = "    environment:\n      - DEBUG=true\n"
	}

	vols := fmt.Sprintf("      - ./resources/%s:/app/config.json\n", configFileName)
	if logDir != "" {
		vols += fmt.Sprintf("      - %s:/app/logs\n", logDir)
	}

	fmt.Fprintf(&b.services, `  application_%s:
    image: application
    build:
      context: ../../application
      dockerfile: Dockerfile
    container_name: application_%s
    depends_on:
      - communication_module_%s
      - uav_controller_%s
%s    volumes:
%s    networks:
      %s:
        aliases:
          - application

`, uavID, uavID, uavID, uavID, env, vols, uavNet)
}

// AddUAVController appends the uav_controller (SITL) service for a UAV.
func (b *composeBuilder) AddUAVController(uavID, configFileName, paramFileName, homeLocation, logDir string, verbose bool) {
	uavNet := uavNetworkName(uavID)

	var env string
	if verbose {
		env = "      - DEBUG=true\n"
	}

	vols := fmt.Sprintf("      - ./resources/%s:/app/config.json\n", configFileName)
	vols += fmt.Sprintf("      - ./resources/%s:/app/copter.parm\n", paramFileName)
	if logDir != "" {
		vols += fmt.Sprintf("      - %s:/app/logs\n", logDir)
	}

	fmt.Fprintf(&b.services, `  uav_controller_%s:
    image: copter453
    build:
      context: ../../uav_controller/ardupilot4_5_3
      dockerfile: SITL
    container_name: uav_controller_%s
    depends_on:
      - communication_module_%s
    environment:
      - UAV_HOME_LOCATION=%s
%s    volumes:
%s    networks:
      %s:
        aliases:
          - uav_controller

`, uavID, uavID, uavID, homeLocation, env, vols, uavNet)
}

// AddExternalComms appends the external_comms service for a UAV.
// This service bridges the per-UAV network and the shared air network.
func (b *composeBuilder) AddExternalComms(uavID, configFileName, logDir string, verbose bool) {
	uavNet := uavNetworkName(uavID)

	var env string
	if verbose {
		env = "    environment:\n      - DEBUG=true\n"
	}

	vols := fmt.Sprintf("      - ./resources/%s:/app/config.json\n", configFileName)
	if logDir != "" {
		vols += fmt.Sprintf("      - %s:/app/logs\n", logDir)
	}

	fmt.Fprintf(&b.services, `  external_comms_%s:
    image: external_comms
    build:
      context: ../../auxiliaries/external_comms
      dockerfile: Dockerfile
    container_name: external_comms_%s
    depends_on:
      - communication_module_%s
      - network_simulator
%s    volumes:
%s    networks:
      %s:
        aliases:
          - external_comms
      air:

`, uavID, uavID, uavID, env, vols, uavNet)
}

// AddAlgorithmService appends a user-deployed algorithm service for a UAV.
func (b *composeBuilder) AddAlgorithmService(uavID string, svc DeployedService, configFileName string, extraVolumes []VolumeMount, logDir string, verbose bool) {
	uavNet := uavNetworkName(uavID)

	var env string
	if verbose {
		env = "    environment:\n      - DEBUG=true\n"
	}

	var vols strings.Builder
	fmt.Fprintf(&vols, "      - ./resources/%s:/app/config.json\n", configFileName)
	for _, v := range extraVolumes {
		fmt.Fprintf(&vols, "      - ./resources/%s:%s\n", v.HostPath, v.ContainerPath)
	}
	if logDir != "" {
		fmt.Fprintf(&vols, "      - %s:/app/logs\n", logDir)
	}

	fmt.Fprintf(&b.services, `  %s_%s:
    image: %s
    build:
      context: ../../algorithms/%s
      dockerfile: Dockerfile
    container_name: %s_%s
    depends_on:
      - communication_module_%s
%s    volumes:
%s    networks:
      %s:
        aliases:
          - %s

`, svc.ServiceId, uavID,
		svc.ServiceId,
		svc.ServiceId,
		svc.ServiceId, uavID,
		uavID,
		env,
		vols.String(),
		uavNet,
		svc.ServiceId)
}

// Build returns the final YAML document as a string.
func (b *composeBuilder) Build() string {
	return b.services.String() + "\n" + b.networks.String()
}

// addNetwork appends a bridge network definition if not already present.
// Multiple calls with the same name are intentionally idempotent — the YAML
// key would duplicate, so callers are responsible for calling once per name.
func (b *composeBuilder) addNetwork(name string) {
	fmt.Fprintf(&b.networks, "  %s:\n    driver: bridge\n", name)
}

func uavNetworkName(uavID string) string {
	return fmt.Sprintf("uav_net_%s", uavID)
}
