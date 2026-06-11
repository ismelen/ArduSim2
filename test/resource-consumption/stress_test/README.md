# ArduSim2 — Local UAV Stress Test

This directory contains the scripts to determine the maximum number of UAVs
that can be simulated simultaneously on a local machine.

## What it does

The test progressively adds **swarms of 2 UAVs** (1 master + 1 slave running
the FollowMe + Mission scenario) and monitors CPU and RAM until the hardware
ceiling is reached.  At the end it produces an HTML report with:

- Maximum UAVs reached
- CPU and RAM time-series charts
- Resource consumption vs number of UAVs chart
- 2-minute degradation observation chart
- Machine specifications
- UAV/CPU and UAV/RAM ratios

## Prerequisites

### Python dependencies
```bash
pip install docker requests matplotlib psutil pyyaml
```

### Docker images
The following images must already be built / pulled before running the test:

| Image | Source |
|---|---|
| `communication_module` | `src/communication_module` |
| `mixer` | `src/mixers/mixer` |
| `uav_controller_arducopter4_5_3` | `src/controllers/uav_controller` |
| `external_comms` | `src/external_comms` |
| `followme` | `src/algorithms/follow_me` |
| `mission` | `src/algorithms/mission` |
| `netsim_gateway` | `src/netsim_gateway` |
| `netsim` | `src/netsim` |
| `logger` | `src/logger` |

## Running the test

```bash
cd test/resource-consumption/stress_test

# Full stress test (runs until hardware limit)
python run_stress_test.py

# Dry-run (stops after 2 swarms = 4 UAVs, for pipeline validation)
python run_stress_test.py --dry-run

# Limit to a specific number of swarms
python run_stress_test.py --max-swarms 5
```

## File structure

```
stress_test/
├── run_stress_test.py   # Main orchestrator
├── generator.py         # Dynamic docker-compose generator
├── monitor.py           # Prometheus metrics client
├── report.py            # HTML report + chart generator
├── thresholds.py        # Configurable limits and timing
├── base_resources/      # Shared config files (same for all swarms)
│   ├── followme_master.json
│   ├── followme_slave.json
│   ├── external_comms.json
│   ├── mixer.json
│   ├── mission.json
│   ├── mission.kml
│   ├── uav_controller.json
│   ├── uav.param
│   ├── netsim.json
│   ├── netsim_gateway.json
│   └── logger.json
├── generated/           # Auto-generated docker-compose files (per run)
├── uav_logs/            # ArduPilot logs per UAV
└── results/             # HTML reports
    └── report_<timestamp>/
        ├── index.html
        ├── cpu_over_time.png
        ├── ram_over_time.png
        ├── consumption_vs_uavs.png
        ├── degradation_period.png
        └── data.csv
```

## Configuration

Edit `thresholds.py` to change:

| Parameter | Default | Description |
|---|---|---|
| `CPU_THRESHOLD_PERCENT` | 100.0 | CPU % to trigger limit detection |
| `RAM_THRESHOLD_PERCENT` | 100.0 | RAM % to trigger limit detection |
| `STABILIZATION_WAIT_S` | 90 | Seconds to wait after each new swarm |
| `DEGRADATION_OBSERVE_S` | 120 | Duration of post-limit observation window |
| `METRICS_SAMPLE_INTERVAL_S` | 5 | Prometheus sampling interval |

## Monitoring dashboards

Once the test is running, you can observe live metrics at:

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3001
- **cAdvisor**: http://localhost:8081

> The monitoring stack is left running after the test finishes so you can
> inspect Grafana dashboards. Stop it manually with:
> ```bash
> docker compose -f ../local/docker-compose.yaml down
> ```

## Network architecture

```
Docker network 'air'  (10.9.0.0/24)
  └── netsim_gateway, netsim_1, logger
  └── external_comms of every UAV

Per-UAV networks (10.10.0.0/16, one /28 per UAV)
  └── communication_module, mixer, controller,
      external_comms, followme, [mission]
```
