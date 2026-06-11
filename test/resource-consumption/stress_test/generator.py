"""
generator.py — Dynamically generates docker-compose YAML for each new swarm.

Each call to `generate_swarm_compose(swarm_id)` returns the YAML content
(as a string) for the 11 containers that make up one 2-UAV swarm:
  UAV 1 (master): communication_module, mixer, controller, external_comms,
                   followme (master), mission
  UAV 2 (slave):  communication_module, mixer, controller, external_comms,
                   followme (slave)

Resources (JSON configs, KML, param file) are the same for every swarm and
are mounted read-only from base_resources/.

Subnets are allocated from 10.10.0.0/16, one /28 per UAV:
  UAV global index 1 → 10.10.0.0/28
  UAV global index 2 → 10.10.0.16/28
  ...
  UAV global index N → 10.10.0.{(N-1)*16}/28
"""

import os
import yaml

from thresholds import (
    AIR_NETWORK_NAME,
    BASE_RESOURCES,
    UAV_SUBNET_BASE,
    UAV_SUBNET_PREFIX,
    STRESS_TEST_DIR,
)

GENERATED_DIR = os.path.join(STRESS_TEST_DIR, "generated")
UAV_LOGS_DIR  = os.path.join(STRESS_TEST_DIR, "uav_logs")

HOME_LAT = "39.481646"
HOME_LON = "-0.349228"
HOME_ALT = "0"
HOME_HDG = "0"
UAV_HOME_LOCATION = f"{HOME_LAT},{HOME_LON},{HOME_ALT},{HOME_HDG}"


def _uav_subnet(uav_global_index: int) -> str:
    """Return the /28 subnet string for the given global UAV index (1-based)."""
    block = (uav_global_index - 1) * 16
    third_octet  = block // 256
    fourth_octet = block % 256
    return f"{UAV_SUBNET_BASE}.{third_octet}.{fourth_octet}/{UAV_SUBNET_PREFIX}"


def _net_name(swarm_id: int, uav_id: int) -> str:
    return f"swarm_net_{swarm_id}_uav_{uav_id}"


def _svc(swarm_id: int, uav_id: int, service: str) -> str:
    return f"swarm_{swarm_id}_uav_{uav_id}_{service}"


def _uav_services(swarm_id: int, uav_id: int, is_master: bool) -> dict:
    """Build the docker-compose service definitions for one UAV."""
    net = _net_name(swarm_id, uav_id)
    svc = lambda s: _svc(swarm_id, uav_id, s)

    base = BASE_RESOURCES
    uav_logs = os.path.join(UAV_LOGS_DIR, f"swarm_{swarm_id}_uav_{uav_id}")
    os.makedirs(uav_logs, exist_ok=True)

    env = [
        f"UAV_ID={uav_id}",
        f"SWARM_ID={swarm_id}",
    ]

    services: dict = {}

    # ── communication_module ──────────────────────────────────────────────
    services[svc("communication_module")] = {
        "image": "communication_module",
        "container_name": svc("communication_module"),
        "environment": env,
        "networks": {net: {"aliases": ["communication_module"]}},
    }

    # ── controller ────────────────────────────────────────────────────────
    services[svc("controller")] = {
        "image": "uav_controller_arducopter4_5_3",
        "container_name": svc("controller"),
        "depends_on": [svc("communication_module")],
        "environment": env + [f"UAV_HOME_LOCATION={UAV_HOME_LOCATION}"],
        "volumes": [
            f"{os.path.join(base, 'uav_controller.json')}:/app/config.json:ro",
            f"{os.path.join(base, 'uav.param')}:/app/copter.parm:ro",
            f"{uav_logs}/:/app/logs/",
        ],
        "networks": {net: {"aliases": ["uav_controller"]}},
    }

    # ── mixer ─────────────────────────────────────────────────────────────
    services[svc("mixer")] = {
        "image": "mixer",
        "container_name": svc("mixer"),
        "depends_on": [svc("communication_module"), svc("controller")],
        "environment": env,
        "volumes": [
            f"{os.path.join(base, 'mixer.json')}:/app/config.json:ro",
        ],
        "networks": {net: {"aliases": ["mixer"]}},
    }

    # ── external_comms ────────────────────────────────────────────────────
    services[svc("external_comms")] = {
        "image": "external_comms",
        "container_name": svc("external_comms"),
        "depends_on": [svc("communication_module")],
        "environment": env,
        "volumes": [
            f"{os.path.join(base, 'external_comms.json')}:/app/config.json:ro",
        ],
        "networks": {
            AIR_NETWORK_NAME: {},
            net: {"aliases": ["external_comms"]},
        },
    }

    # ── followme ──────────────────────────────────────────────────────────
    followme_cfg = "followme_master.json" if is_master else "followme_slave.json"
    services[svc("followme")] = {
        "image": "followme",
        "container_name": svc("followme"),
        "depends_on": [svc("communication_module")],
        "environment": env,
        "volumes": [
            f"{os.path.join(base, followme_cfg)}:/app/config.json:ro",
        ],
        "networks": {net: {"aliases": ["followme"]}},
    }

    # ── mission (master only) ─────────────────────────────────────────────
    if is_master:
        services[svc("mission")] = {
            "image": "mission",
            "container_name": svc("mission"),
            "depends_on": [svc("communication_module")],
            "environment": env,
            "volumes": [
                f"{os.path.join(base, 'mission.json')}:/app/config.json:ro",
                f"{os.path.join(base, 'mission.kml')}:/app/mission_file.kml:ro",
            ],
            "networks": {net: {"aliases": ["mission"]}},
        }

    return services


def generate_swarm_compose(swarm_id: int, uav_global_offset: int) -> str:
    """
    Generate docker-compose YAML content for swarm `swarm_id`.

    `uav_global_offset` is the 1-based index of the first UAV in this swarm
    across the entire test run.  Used to allocate unique /28 subnets.

    Returns the YAML string ready to write to a file.
    """
    os.makedirs(GENERATED_DIR, exist_ok=True)

    uav1_global = uav_global_offset       # master
    uav2_global = uav_global_offset + 1   # slave

    services = {}
    services.update(_uav_services(swarm_id, 1, is_master=True))
    services.update(_uav_services(swarm_id, 2, is_master=False))

    net1 = _net_name(swarm_id, 1)
    net2 = _net_name(swarm_id, 2)

    networks = {
        AIR_NETWORK_NAME: {"external": True, "name": AIR_NETWORK_NAME},
        net1: {
            "driver": "bridge",
            "ipam": {"config": [{"subnet": _uav_subnet(uav1_global)}]},
        },
        net2: {
            "driver": "bridge",
            "ipam": {"config": [{"subnet": _uav_subnet(uav2_global)}]},
        },
    }

    compose = {
        "services": services,
        "networks": networks,
    }

    return yaml.dump(compose, default_flow_style=False, sort_keys=False)


def write_swarm_compose(swarm_id: int, uav_global_offset: int) -> str:
    """Write the docker-compose file for the swarm and return its path."""
    content = generate_swarm_compose(swarm_id, uav_global_offset)
    path = os.path.join(GENERATED_DIR, f"docker-compose-swarm-{swarm_id}.yaml")
    with open(path, "w") as f:
        f.write(content)
    return path


def generate_infra_compose() -> str:
    """
    Generate docker-compose for the shared infrastructure:
    netsim_gateway, netsim_1, logger.

    Returns the path to the written file.
    """
    os.makedirs(GENERATED_DIR, exist_ok=True)

    base = BASE_RESOURCES

    compose = {
        "services": {
            "netsim_gateway": {
                "image": "netsim_gateway",
                "container_name": "netsim_gateway",
                "extra_hosts": ["host.docker.internal:host-gateway"],
                "ports": [
                    "3000:3000/udp",
                    "3001:3001/udp",
                    "3002:3002/udp",
                ],
                "environment": ["ADDRS=netsim_1:3000"],
                "volumes": [
                    f"{os.path.join(base, 'netsim_gateway.json')}:/app/config.json:ro",
                ],
                "networks": {AIR_NETWORK_NAME: {}},
            },
            "netsim_1": {
                "image": "netsim",
                "container_name": "netsim_1",
                "depends_on": ["netsim_gateway"],
                "environment": ["NODE_ID=netsim_1"],
                "volumes": [
                    f"{os.path.join(base, 'netsim.json')}:/app/config.json:ro",
                ],
                "networks": {AIR_NETWORK_NAME: {}},
            },
            "logger": {
                "image": "logger",
                "container_name": "logger",
                "ports": ["5000:5000/udp", "8080:8080/tcp"],
                "volumes": [
                    f"{os.path.join(base, 'logger.json')}:/app/config.json:ro",
                ],
                "networks": {AIR_NETWORK_NAME: {}},
            },
        },
        "networks": {
            AIR_NETWORK_NAME: {
                "driver": "bridge",
                "ipam": {"config": [{"subnet": "10.9.0.0/24"}]},
            }
        },
    }

    path = os.path.join(GENERATED_DIR, "docker-compose-infra.yaml")
    with open(path, "w") as f:
        f.write(yaml.dump(compose, default_flow_style=False, sort_keys=False))
    return path
