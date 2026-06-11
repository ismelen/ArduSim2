"""
monitor.py — Prometheus client for CPU and RAM metrics.

Queries cAdvisor metrics via Prometheus to obtain:
  - CPU usage (% of total host cores)
  - RAM usage (% and absolute bytes)

Also collects per-container stats for debugging / fine-grained analysis.
"""

import time
import requests
from dataclasses import dataclass, field
from typing import List, Optional

from thresholds import PROMETHEUS_URL, METRICS_SAMPLE_INTERVAL_S


@dataclass
class MetricPoint:
    timestamp: float          # Unix timestamp
    num_uavs: int             # number of UAVs running at this moment
    cpu_percent: float        # host CPU % (across all cores)
    ram_percent: float        # host RAM %
    ram_bytes: int            # host RAM used in bytes
    ram_total_bytes: int      # host RAM total in bytes
    cpu_cores: int            # number of logical CPU cores on the host
    event: Optional[str] = None  # e.g. "swarm_1_added", "limit_reached"


class PrometheusMonitor:
    """
    Thin wrapper around the Prometheus HTTP API.
    Keeps an in-memory log of MetricPoint samples.
    """

    def __init__(self, url: str = PROMETHEUS_URL):
        self.url = url.rstrip("/")
        self.samples: List[MetricPoint] = []

    # ------------------------------------------------------------------
    # Low-level helpers
    # ------------------------------------------------------------------

    def _query(self, promql: str) -> Optional[float]:
        """Run an instant query and return the first scalar value, or None."""
        try:
            r = requests.get(
                f"{self.url}/api/v1/query",
                params={"query": promql},
                timeout=10,
            )
            r.raise_for_status()
            data = r.json()
            results = data.get("data", {}).get("result", [])
            if results:
                return float(results[0]["value"][1])
        except Exception as e:
            print(f"[monitor] Prometheus query error: {e}")
        return None

    # ------------------------------------------------------------------
    # Host-level metrics
    # ------------------------------------------------------------------

    def cpu_total_cores(self) -> Optional[int]:
        """Number of logical CPUs reported by cAdvisor."""
        val = self._query("machine_cpu_cores")
        return int(val) if val is not None else None

    def ram_total_bytes(self) -> Optional[int]:
        """Total host RAM in bytes reported by cAdvisor."""
        val = self._query("machine_memory_bytes")
        return int(val) if val is not None else None

    def cpu_used_cores(self) -> Optional[float]:
        """
        Aggregate CPU usage in cores across all running containers.
        Uses a 30s rate window for stability.
        """
        val = self._query(
            'sum(rate(container_cpu_usage_seconds_total{image!=""}[30s]))'
        )
        return val  # fractional cores

    def ram_used_bytes(self) -> Optional[int]:
        """Aggregate RAM usage in bytes across all running containers."""
        val = self._query(
            'sum(container_memory_usage_bytes{image!=""})'
        )
        return int(val) if val is not None else None

    # ------------------------------------------------------------------
    # Combined snapshot
    # ------------------------------------------------------------------

    def sample(self, num_uavs: int, event: str = None) -> Optional[MetricPoint]:
        """
        Take a single metric sample and append it to self.samples.
        Returns the MetricPoint, or None if Prometheus is unreachable.
        """
        cores      = self.cpu_total_cores()
        ram_total  = self.ram_total_bytes()
        cpu_cores  = self.cpu_used_cores()
        ram_used   = self.ram_used_bytes()

        if None in (cores, ram_total, cpu_cores, ram_used):
            print("[monitor] Incomplete metrics — skipping sample.")
            return None

        cpu_pct = min((cpu_cores / cores) * 100.0, 100.0)
        ram_pct = min((ram_used / ram_total) * 100.0, 100.0)

        pt = MetricPoint(
            timestamp=time.time(),
            num_uavs=num_uavs,
            cpu_percent=cpu_pct,
            ram_percent=ram_pct,
            ram_bytes=ram_used,
            ram_total_bytes=ram_total,
            cpu_cores=cores,
            event=event,
        )
        self.samples.append(pt)
        print(
            f"[monitor] UAVs={num_uavs:3d}  CPU={cpu_pct:5.1f}%  "
            f"RAM={ram_pct:5.1f}% ({ram_used / 1e9:.2f} GB / "
            f"{ram_total / 1e9:.2f} GB)"
        )
        return pt

    # ------------------------------------------------------------------
    # Convenience: sample for a period
    # ------------------------------------------------------------------

    def observe_for(self, duration_s: int, num_uavs: int, event_prefix: str = "") -> None:
        """
        Continuously sample at METRICS_SAMPLE_INTERVAL_S for `duration_s` seconds.
        """
        end = time.time() + duration_s
        first = True
        while time.time() < end:
            evt = (event_prefix or None) if first else None
            self.sample(num_uavs, event=evt)
            first = False
            time.sleep(METRICS_SAMPLE_INTERVAL_S)

    def wait_and_sample(
        self,
        wait_s: int,
        num_uavs: int,
        event_on_start: str = None,
    ) -> MetricPoint:
        """
        Wait `wait_s` seconds while sampling every METRICS_SAMPLE_INTERVAL_S,
        then take and return one final decisive sample.
        """
        elapsed = 0
        first = True
        while elapsed < wait_s:
            sleep_time = min(METRICS_SAMPLE_INTERVAL_S, wait_s - elapsed)
            time.sleep(sleep_time)
            elapsed += sleep_time
            evt = event_on_start if first else None
            self.sample(num_uavs, event=evt)
            first = False

        return self.samples[-1] if self.samples else None

    def check_prometheus_alive(self) -> bool:
        """Return True if the Prometheus instance is reachable."""
        try:
            r = requests.get(f"{self.url}/-/ready", timeout=5)
            return r.status_code == 200
        except Exception:
            return False
