#!/usr/bin/env python3
"""
run_stress_test.py — ArduSim2 local UAV stress test orchestrator.

Usage:
    python run_stress_test.py [--max-swarms N] [--dry-run]

Options:
    --max-swarms N    Stop after N swarms regardless of resource usage (default: unlimited)
    --dry-run         Run with max-swarms=2 to validate the pipeline without stress

Dependencies (install with pip on the system Python):
    pip install docker requests matplotlib psutil pyyaml

The script:
  1. Creates the Docker network 'air' if it does not exist.
  2. Starts the shared infrastructure (netsim_gateway, netsim_1, logger).
  3. Starts the monitoring stack (cAdvisor, Prometheus, Grafana).
  4. Loops, adding one swarm (2 UAVs) at a time:
       a. Generates docker-compose for the new swarm.
       b. Brings up the swarm containers.
       c. Waits 90s for stabilisation while sampling metrics.
       d. Sends the 'start' command to all UAVs in the new swarm via the
          netsim_gateway messages port (3001) — same mechanism as the GUI.
       e. Evaluates CPU/RAM thresholds.
       f. If limit reached: observes for 2 min, then stops and generates report.
  5. Tears down everything and opens the report.
"""

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import time
import webbrowser

from thresholds import (
    AIR_NETWORK_NAME,
    AIR_NETWORK_DRIVER,
    AIR_NETWORK_SUBNET,
    CPU_THRESHOLD_PERCENT,
    RAM_THRESHOLD_PERCENT,
    STABILIZATION_WAIT_S,
    DEGRADATION_OBSERVE_S,
    METRICS_SAMPLE_INTERVAL_S,
    START_TARGET_HOST,
    START_TARGET_PORT,
    STRESS_TEST_DIR,
    MONITOR_COMPOSE_PATH,
)
from generator import generate_infra_compose, write_swarm_compose
from monitor import PrometheusMonitor
from report import generate_report


# ── docker helpers ────────────────────────────────────────────────────────────

def _run(cmd: list, check: bool = True, capture: bool = False) -> subprocess.CompletedProcess:
    """Run a shell command, printing it first."""
    print(f"[cmd] {' '.join(str(c) for c in cmd)}")
    return subprocess.run(
        cmd,
        check=check,
        capture_output=capture,
        text=True,
    )


def docker_network_exists(name: str) -> bool:
    result = _run(
        ["docker", "network", "ls", "--filter", f"name=^{name}$", "--format", "{{{{.Name}}}}"],
        capture=True,
    )
    return name in result.stdout.strip().split("\n")


def create_air_network() -> None:
    if docker_network_exists(AIR_NETWORK_NAME):
        print(f"[net] Network '{AIR_NETWORK_NAME}' already exists.")
        return
    _run([
        "docker", "network", "create",
        "--driver", AIR_NETWORK_DRIVER,
        "--subnet", AIR_NETWORK_SUBNET,
        AIR_NETWORK_NAME,
    ])
    print(f"[net] Created network '{AIR_NETWORK_NAME}' ({AIR_NETWORK_SUBNET}).")


def compose_up(compose_path: str, project_name: str = None) -> None:
    cmd = ["docker", "compose", "-f", compose_path, "up", "-d", "--remove-orphans"]
    if project_name:
        cmd += ["--project-name", project_name]
    _run(cmd)


def compose_down(compose_path: str, project_name: str = None) -> None:
    cmd = ["docker", "compose", "-f", compose_path, "down", "--remove-orphans"]
    if project_name:
        cmd += ["--project-name", project_name]
    _run(cmd, check=False)


def all_containers_running(swarm_id: int) -> bool:
    """Return True if all 11 containers for swarm_id show status=running."""
    result = _run(
        ["docker", "ps", "--filter", f"name=swarm_{swarm_id}_",
         "--format", "{{.Status}}"],
        capture=True,
    )
    statuses = [l.strip() for l in result.stdout.strip().splitlines() if l.strip()]
    return all("Up" in s for s in statuses) and len(statuses) >= 11


# ── START command sender ──────────────────────────────────────────────────────

def send_start_to_swarm(swarm_id: int) -> None:
    """
    Send the 'start' command to all UAVs in the swarm via the netsim_gateway
    messages port (3001).  Mirrors the GUI's SendGlobalBroadcast call.

    Packet format (from subscriber.go / simulation_interactor.go):
      {
        "uav_id": "",          // empty = broadcast to all UAVs
        "payload": {
          "topic":   "algo/followme",
          "payload": {"command": "start"}
        }
      }
    """
    payload = {
        "uav_id": "",
        "payload": {
            "topic": "algo/followme",
            "payload": {"command": "start"},
        },
    }
    data = json.dumps(payload).encode("utf-8")

    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.sendto(data, (START_TARGET_HOST, START_TARGET_PORT))
        sock.close()
        print(f"[start] Sent 'start' for swarm {swarm_id} → "
              f"{START_TARGET_HOST}:{START_TARGET_PORT}")
    except Exception as e:
        print(f"[start] WARNING: could not send start command: {e}")


# ── main loop ─────────────────────────────────────────────────────────────────

def run(max_swarms: int = 0) -> None:
    monitor = PrometheusMonitor()
    swarm_compose_paths: list = []

    # ── pre-flight checks ─────────────────────────────────────────────────
    print("\n=== ArduSim2 Stress Test ===\n")

    if not shutil.which("docker"):
        sys.exit("[error] Docker is not installed or not in PATH.")

    try:
        _run(["docker", "info"], capture=True)
    except subprocess.CalledProcessError:
        sys.exit("[error] Docker daemon is not running.")

    # ── Docker network ────────────────────────────────────────────────────
    create_air_network()

    # ── Infrastructure stack (netsim_gateway + netsim_1 + logger) ────────
    infra_path = generate_infra_compose()
    print(f"\n[infra] Starting infrastructure stack from {infra_path}")
    compose_up(infra_path, project_name="ardusim_infra")
    time.sleep(5)  # give containers a moment to bind ports

    # ── Monitoring stack (cAdvisor + Prometheus + Grafana) ────────────────
    if os.path.exists(MONITOR_COMPOSE_PATH):
        print(f"\n[monitor] Starting monitoring stack from {MONITOR_COMPOSE_PATH}")
        compose_up(MONITOR_COMPOSE_PATH, project_name="ardusim_monitor")

        # Wait for Prometheus to be ready
        print("[monitor] Waiting for Prometheus to be ready...")
        for _ in range(30):
            if monitor.check_prometheus_alive():
                print("[monitor] Prometheus is up.")
                break
            time.sleep(2)
        else:
            print("[monitor] WARNING: Prometheus did not respond. Metrics may be missing.")
    else:
        print(f"[monitor] WARNING: monitoring compose not found at {MONITOR_COMPOSE_PATH}")

    # ── Main swarm loop ───────────────────────────────────────────────────
    swarm_id       = 0
    max_uavs_seen  = 0
    limit_reached  = False
    degradation_samples = []

    try:
        while True:
            swarm_id += 1
            if max_swarms and swarm_id > max_swarms:
                print(f"\n[loop] Reached max-swarms limit ({max_swarms}). Stopping.")
                break

            num_uavs = swarm_id * 2
            # Each swarm uses 2 UAV global slots
            uav_global_offset = (swarm_id - 1) * 2 + 1

            print(f"\n{'='*60}")
            print(f"  Adding swarm {swarm_id}  →  total UAVs: {num_uavs}")
            print(f"{'='*60}")

            # Generate and launch compose for this swarm
            compose_path = write_swarm_compose(swarm_id, uav_global_offset)
            swarm_compose_paths.append(compose_path)
            compose_up(compose_path, project_name=f"ardusim_swarm_{swarm_id}")

            # Wait for stabilisation, sampling metrics along the way
            print(f"\n[loop] Waiting {STABILIZATION_WAIT_S}s for stabilisation...")
            pt = monitor.wait_and_sample(
                STABILIZATION_WAIT_S,
                num_uavs,
                event_on_start=f"swarm_{swarm_id}_added",
            )

            # Send START command
            send_start_to_swarm(swarm_id)

            # Take one more sample after start command
            time.sleep(3)
            pt = monitor.sample(num_uavs)

            if pt is None:
                print("[loop] Could not get metrics — skipping threshold check.")
                continue

            max_uavs_seen = max(max_uavs_seen, num_uavs)

            # Threshold check
            if pt.cpu_percent >= CPU_THRESHOLD_PERCENT or \
               pt.ram_percent >= RAM_THRESHOLD_PERCENT:
                print(
                    f"\n[limit] RESOURCE LIMIT REACHED at {num_uavs} UAVs "
                    f"(CPU={pt.cpu_percent:.1f}%, RAM={pt.ram_percent:.1f}%)"
                )
                limit_reached = True
                break

        # ── Degradation observation ────────────────────────────────────────
        if limit_reached:
            print(f"\n[degradation] Observing system for {DEGRADATION_OBSERVE_S}s …")
            end = time.time() + DEGRADATION_OBSERVE_S
            while time.time() < end:
                dp = monitor.sample(max_uavs_seen, event=None)
                if dp:
                    degradation_samples.append(dp)
                time.sleep(METRICS_SAMPLE_INTERVAL_S)

    except KeyboardInterrupt:
        print("\n[!] Test interrupted by user.")

    finally:
        # ── Teardown ──────────────────────────────────────────────────────
        print("\n[teardown] Stopping all swarm containers …")
        for path in reversed(swarm_compose_paths):
            swarm_n = os.path.basename(path).replace(
                "docker-compose-swarm-", "").replace(".yaml", "")
            compose_down(path, project_name=f"ardusim_swarm_{swarm_n}")

        print("[teardown] Stopping infrastructure …")
        infra_path = os.path.join(
            STRESS_TEST_DIR, "generated", "docker-compose-infra.yaml"
        )
        if os.path.exists(infra_path):
            compose_down(infra_path, project_name="ardusim_infra")

        # Keep monitoring stack running so user can inspect Grafana afterward.
        print("[teardown] Monitoring stack left running (Grafana: http://localhost:3001).")

        # ── Report ────────────────────────────────────────────────────────
        if monitor.samples:
            print("\n[report] Generating report …")
            html_path = generate_report(
                monitor.samples,
                degradation_samples,
                max_uavs=max_uavs_seen,
            )
            try:
                webbrowser.open(f"file://{html_path}")
            except Exception:
                pass
            print(f"\n✓ Max UAVs supported: {max_uavs_seen}")
            print(f"✓ Report: {html_path}")
        else:
            print("[report] No samples collected — skipping report.")


# ── CLI ───────────────────────────────────────────────────────────────────────

def main() -> None:
    parser = argparse.ArgumentParser(
        description="ArduSim2 local UAV stress test",
    )
    parser.add_argument(
        "--max-swarms", type=int, default=0,
        help="Stop after this many swarms (0 = unlimited, until resource limit).",
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Run with max-swarms=2 (quick pipeline validation).",
    )
    args = parser.parse_args()

    max_swarms = 2 if args.dry_run else args.max_swarms
    run(max_swarms=max_swarms)


if __name__ == "__main__":
    main()
