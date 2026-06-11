"""
Stress test configuration and thresholds.
Edit these values to tune the behaviour of the test.
"""

# ---------------------------------------------------------------------------
# Resource limits — set to 100 to find the true hardware ceiling.
# ---------------------------------------------------------------------------
CPU_THRESHOLD_PERCENT = 100.0   # % of total host CPU (across all cores)
RAM_THRESHOLD_PERCENT = 100.0   # % of total host RAM

# ---------------------------------------------------------------------------
# Timing
# ---------------------------------------------------------------------------
STABILIZATION_WAIT_S      = 90   # seconds to wait after docker compose up
DEGRADATION_OBSERVE_S     = 120  # extra observation window once limit is hit
METRICS_SAMPLE_INTERVAL_S = 5    # how often Prometheus is queried (seconds)

# ---------------------------------------------------------------------------
# Monitoring stack
# ---------------------------------------------------------------------------
PROMETHEUS_URL = "http://localhost:9090"

# ---------------------------------------------------------------------------
# Docker networking
# ---------------------------------------------------------------------------
# The 'air' network is defined by the ZIP compose file (driver=bridge, subnet 10.9.0.0/24).
# All netsim_gateway / netsim / logger / external_comms containers share it.
AIR_NETWORK_NAME   = "air"
AIR_NETWORK_SUBNET = "10.9.0.0/24"
AIR_NETWORK_DRIVER = "bridge"

# Per-UAV isolated networks live in this space, one /28 per UAV.
# /28 = 16 IPs; supports up to 4096 UAVs in 10.10.0.0/16.
UAV_SUBNET_BASE   = "10.10"   # first two octets
UAV_SUBNET_PREFIX = 28

# ---------------------------------------------------------------------------
# netsim_gateway ports (from netsim_gateway_config_f78e7a71.json)
# ---------------------------------------------------------------------------
NETSIM_GATEWAY_TELEMETRY_PORT   = 3000   # inbound telemetry from UAVs
NETSIM_GATEWAY_MESSAGES_PORT    = 3001   # inbound broadcast messages from UAVs
NETSIM_GATEWAY_SUBSCRIBERS_PORT = 3002   # UI subscriber registration
NETSIM_GATEWAY_NETSIM_PORT      = 3003   # communication with netsim nodes

# The START command is sent to the messages port (3001) mimicking the GUI.
START_TARGET_HOST = "127.0.0.1"
START_TARGET_PORT = NETSIM_GATEWAY_MESSAGES_PORT

# ---------------------------------------------------------------------------
# File / directory structure
# ---------------------------------------------------------------------------
import os

STRESS_TEST_DIR  = os.path.dirname(os.path.abspath(__file__))
BASE_RESOURCES   = os.path.join(STRESS_TEST_DIR, "base_resources")
RESULTS_BASE_DIR = os.path.join(STRESS_TEST_DIR, "results")

# Infra docker-compose (netsim_gateway + netsim + logger) — generated at runtime
INFRA_COMPOSE_PATH = os.path.join(STRESS_TEST_DIR, "generated", "docker-compose-infra.yaml")

# Monitoring stack
MONITOR_COMPOSE_PATH = os.path.join(
    STRESS_TEST_DIR, "..", "local", "docker-compose.yaml"
)
