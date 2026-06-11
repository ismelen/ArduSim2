"""
report.py — Generates the final HTML stress test report.

Produces:
  results/report_<timestamp>/
      index.html          — self-contained HTML with embedded PNG charts
      cpu_over_time.png
      ram_over_time.png
      consumption_vs_uavs.png
      degradation_period.png
      data.csv            — raw data table
"""

import os
import csv
import platform
import datetime
import math
from typing import List, Optional

import psutil
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker

from monitor import MetricPoint
from thresholds import RESULTS_BASE_DIR


# ── colour palette ────────────────────────────────────────────────────────────
CPU_COLOR  = "#4A90E2"
RAM_COLOR  = "#E2844A"
LIMIT_COLOR = "#E24A4A"
EVENT_COLOR = "#7ED321"
BG_COLOR   = "#1A1A2E"
TEXT_COLOR  = "#E0E0E0"
GRID_COLOR  = "#2A2A4E"

plt.rcParams.update({
    "figure.facecolor":  BG_COLOR,
    "axes.facecolor":    BG_COLOR,
    "axes.edgecolor":    GRID_COLOR,
    "axes.labelcolor":   TEXT_COLOR,
    "xtick.color":       TEXT_COLOR,
    "ytick.color":       TEXT_COLOR,
    "text.color":        TEXT_COLOR,
    "grid.color":        GRID_COLOR,
    "legend.facecolor":  "#16213E",
    "legend.edgecolor":  GRID_COLOR,
})


# ── helpers ───────────────────────────────────────────────────────────────────

def _machine_specs() -> dict:
    mem = psutil.virtual_memory()
    cpu_freq = psutil.cpu_freq()
    return {
        "os":          f"{platform.system()} {platform.release()} ({platform.version()})",
        "machine":     platform.machine(),
        "processor":   platform.processor() or "N/A",
        "cpu_physical": psutil.cpu_count(logical=False),
        "cpu_logical":  psutil.cpu_count(logical=True),
        "cpu_freq_mhz": f"{cpu_freq.max:.0f}" if cpu_freq else "N/A",
        "ram_total_gb": f"{mem.total / 1e9:.2f}",
        "python":       platform.python_version(),
    }


def _swarm_events(samples: List[MetricPoint]) -> List[tuple]:
    """Return (timestamp, label) for samples that have an event."""
    events = []
    for s in samples:
        if s.event:
            events.append((s.timestamp, s.event))
    return events


def _ts_to_min(samples: List[MetricPoint]) -> list:
    """Convert timestamps to minutes elapsed since first sample."""
    if not samples:
        return []
    t0 = samples[0].timestamp
    return [(s.timestamp - t0) / 60.0 for s in samples]


def _style_ax(ax, title: str, xlabel: str, ylabel: str, ylim=(0, 105)):
    ax.set_title(title, fontsize=13, fontweight="bold", pad=10)
    ax.set_xlabel(xlabel, fontsize=10)
    ax.set_ylabel(ylabel, fontsize=10)
    ax.set_ylim(*ylim)
    ax.grid(True, linestyle="--", alpha=0.4)
    ax.yaxis.set_major_formatter(mticker.FormatStrFormatter("%.0f"))
    ax.legend(loc="upper left", fontsize=9)


# ── chart generators ──────────────────────────────────────────────────────────

def _chart_cpu_over_time(samples: List[MetricPoint], out_path: str) -> None:
    fig, ax = plt.subplots(figsize=(10, 4))
    xs = _ts_to_min(samples)
    ax.plot(xs, [s.cpu_percent for s in samples],
            color=CPU_COLOR, linewidth=1.8, label="CPU %")
    ax.axhline(100, color=LIMIT_COLOR, linestyle="--", linewidth=1.2, label="Limit 100%")

    events = _swarm_events(samples)
    t0 = samples[0].timestamp if samples else 0
    for ts, label in events:
        xv = (ts - t0) / 60.0
        ax.axvline(xv, color=EVENT_COLOR, linestyle=":", linewidth=1, alpha=0.8)
        ax.text(xv + 0.05, 95, label, color=EVENT_COLOR, fontsize=7, rotation=90,
                va="top", alpha=0.85)

    _style_ax(ax, "CPU Usage Over Time", "Time (min)", "CPU (%)")
    fig.tight_layout()
    fig.savefig(out_path, dpi=150, bbox_inches="tight")
    plt.close(fig)


def _chart_ram_over_time(samples: List[MetricPoint], out_path: str) -> None:
    fig, ax = plt.subplots(figsize=(10, 4))
    xs = _ts_to_min(samples)
    ram_gb = [s.ram_bytes / 1e9 for s in samples]
    ram_total_gb = samples[0].ram_total_bytes / 1e9 if samples else 0

    ax2 = ax.twinx()
    ax.plot(xs, ram_gb, color=RAM_COLOR, linewidth=1.8, label="RAM (GB)")
    ax2.plot(xs, [s.ram_percent for s in samples],
             color=RAM_COLOR, linewidth=0, alpha=0)   # invisible, just for scale
    ax.axhline(ram_total_gb, color=LIMIT_COLOR, linestyle="--",
               linewidth=1.2, label=f"Total {ram_total_gb:.1f} GB")

    ax.set_title("RAM Usage Over Time", fontsize=13, fontweight="bold", pad=10)
    ax.set_xlabel("Time (min)", fontsize=10)
    ax.set_ylabel("RAM (GB)", fontsize=10)
    ax2.set_ylabel("RAM (%)", fontsize=10, color=TEXT_COLOR)
    ax2.set_ylim(0, 105)
    ax.set_ylim(0, ram_total_gb * 1.1)
    ax.grid(True, linestyle="--", alpha=0.4)
    ax.legend(loc="upper left", fontsize=9)

    events = _swarm_events(samples)
    t0 = samples[0].timestamp if samples else 0
    for ts, label in events:
        xv = (ts - t0) / 60.0
        ax.axvline(xv, color=EVENT_COLOR, linestyle=":", linewidth=1, alpha=0.8)

    fig.tight_layout()
    fig.savefig(out_path, dpi=150, bbox_inches="tight")
    plt.close(fig)


def _chart_consumption_vs_uavs(samples: List[MetricPoint], out_path: str) -> None:
    """One bar per unique UAV count — average CPU and RAM at that count."""
    from collections import defaultdict
    cpu_by_n: dict = defaultdict(list)
    ram_by_n: dict = defaultdict(list)
    for s in samples:
        cpu_by_n[s.num_uavs].append(s.cpu_percent)
        ram_by_n[s.num_uavs].append(s.ram_percent)

    ns = sorted(cpu_by_n.keys())
    cpu_avgs = [sum(cpu_by_n[n]) / len(cpu_by_n[n]) for n in ns]
    ram_avgs = [sum(ram_by_n[n]) / len(ram_by_n[n]) for n in ns]

    x = list(range(len(ns)))
    width = 0.38

    fig, ax = plt.subplots(figsize=(max(8, len(ns) * 0.9), 5))
    bars1 = ax.bar([i - width / 2 for i in x], cpu_avgs,
                   width, label="CPU %", color=CPU_COLOR, alpha=0.85)
    bars2 = ax.bar([i + width / 2 for i in x], ram_avgs,
                   width, label="RAM %", color=RAM_COLOR, alpha=0.85)
    ax.set_xticks(x)
    ax.set_xticklabels([str(n) for n in ns], rotation=45, ha="right", fontsize=9)
    _style_ax(ax, "Resource Consumption vs Number of UAVs",
              "Number of UAVs", "Usage (%)")
    fig.tight_layout()
    fig.savefig(out_path, dpi=150, bbox_inches="tight")
    plt.close(fig)


def _chart_degradation(
    degradation_samples: List[MetricPoint], out_path: str
) -> None:
    """Plot the 2-minute degradation observation window."""
    if not degradation_samples:
        return

    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(10, 6), sharex=True)
    xs = _ts_to_min(degradation_samples)

    ax1.plot(xs, [s.cpu_percent for s in degradation_samples],
             color=CPU_COLOR, linewidth=1.8, label="CPU %")
    ax1.axhline(100, color=LIMIT_COLOR, linestyle="--", linewidth=1.2)
    _style_ax(ax1, "Degradation Period — CPU", "Time (min)", "CPU (%)")

    ax2.plot(xs, [s.ram_percent for s in degradation_samples],
             color=RAM_COLOR, linewidth=1.8, label="RAM %")
    ax2.axhline(100, color=LIMIT_COLOR, linestyle="--", linewidth=1.2)
    _style_ax(ax2, "Degradation Period — RAM", "Time (min)", "RAM (%)")

    fig.suptitle("System Behaviour at Resource Limit", fontsize=14,
                 fontweight="bold", color=TEXT_COLOR)
    fig.tight_layout()
    fig.savefig(out_path, dpi=150, bbox_inches="tight")
    plt.close(fig)


# ── HTML report ───────────────────────────────────────────────────────────────

_HTML_TEMPLATE = """\
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ArduSim2 Stress Test Report — {timestamp}</title>
<style>
  :root {{
    --bg: #1A1A2E; --surface: #16213E; --accent: #4A90E2;
    --warn: #E2844A; --danger: #E24A4A; --ok: #7ED321;
    --text: #E0E0E0; --sub: #9090B0; --border: #2A2A4E;
  }}
  * {{ box-sizing: border-box; margin: 0; padding: 0; }}
  body {{ background: var(--bg); color: var(--text); font-family: 'Segoe UI', system-ui, sans-serif;
         padding: 2rem; line-height: 1.6; }}
  h1 {{ font-size: 2rem; color: var(--accent); margin-bottom: 0.25rem; }}
  h2 {{ font-size: 1.3rem; color: var(--accent); margin: 2rem 0 0.75rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }}
  .subtitle {{ color: var(--sub); margin-bottom: 2rem; }}
  .hero {{ background: var(--surface); border: 1px solid var(--border); border-radius: 12px;
           padding: 1.5rem 2rem; display: flex; gap: 3rem; flex-wrap: wrap; margin-bottom: 2rem; }}
  .hero-stat {{ text-align: center; }}
  .hero-val {{ font-size: 3rem; font-weight: 700; color: var(--ok); }}
  .hero-lbl {{ color: var(--sub); font-size: 0.85rem; }}
  .card {{ background: var(--surface); border: 1px solid var(--border); border-radius: 12px;
           padding: 1.5rem; margin-bottom: 1.5rem; }}
  .specs-grid {{ display: grid; grid-template-columns: max-content 1fr; gap: 0.4rem 1.5rem; }}
  .specs-grid dt {{ color: var(--sub); font-size: 0.9rem; }}
  .specs-grid dd {{ font-weight: 500; }}
  img {{ max-width: 100%; border-radius: 8px; border: 1px solid var(--border); margin-top: 0.5rem; }}
  table {{ width: 100%; border-collapse: collapse; font-size: 0.88rem; }}
  th {{ background: #0F3460; color: var(--accent); padding: 0.5rem 0.75rem; text-align: left; }}
  td {{ padding: 0.45rem 0.75rem; border-bottom: 1px solid var(--border); }}
  tr:hover td {{ background: rgba(74,144,226,0.07); }}
  .ratio-row {{ display: flex; gap: 2rem; flex-wrap: wrap; }}
  .ratio-card {{ background: #0F3460; border-radius: 8px; padding: 1rem 1.5rem; flex: 1; min-width: 180px; }}
  .ratio-val {{ font-size: 1.8rem; font-weight: 700; color: var(--accent); }}
  .ratio-lbl {{ color: var(--sub); font-size: 0.85rem; }}
</style>
</head>
<body>
<h1>ArduSim2 Stress Test</h1>
<p class="subtitle">Local simulation — maximum UAV capacity — {timestamp}</p>

<div class="hero">
  <div class="hero-stat"><div class="hero-val">{max_uavs}</div><div class="hero-lbl">Max UAVs reached</div></div>
  <div class="hero-stat"><div class="hero-val">{max_swarms}</div><div class="hero-lbl">Swarms deployed</div></div>
  <div class="hero-stat"><div class="hero-val" style="color:var(--warn)">{peak_cpu:.1f}%</div><div class="hero-lbl">Peak CPU</div></div>
  <div class="hero-stat"><div class="hero-val" style="color:var(--warn)">{peak_ram:.1f}%</div><div class="hero-lbl">Peak RAM</div></div>
</div>

<h2>Machine Specifications</h2>
<div class="card">
<dl class="specs-grid">
  <dt>Operating System</dt><dd>{os}</dd>
  <dt>Processor</dt><dd>{processor}</dd>
  <dt>Physical / Logical Cores</dt><dd>{cpu_physical} / {cpu_logical}</dd>
  <dt>CPU Frequency (max)</dt><dd>{cpu_freq_mhz} MHz</dd>
  <dt>Total RAM</dt><dd>{ram_total_gb} GB</dd>
  <dt>Architecture</dt><dd>{machine}</dd>
  <dt>Python version</dt><dd>{python}</dd>
</dl>
</div>

<h2>Resource Ratios</h2>
<div class="card">
<div class="ratio-row">
  <div class="ratio-card">
    <div class="ratio-val">{uavs_per_core:.2f}</div>
    <div class="ratio-lbl">UAVs per logical CPU core</div>
  </div>
  <div class="ratio-card">
    <div class="ratio-val">{uavs_per_gb:.2f}</div>
    <div class="ratio-lbl">UAVs per GB of RAM</div>
  </div>
  <div class="ratio-card">
    <div class="ratio-val">{uavs_per_cpu_pct:.2f}</div>
    <div class="ratio-lbl">UAVs per % of CPU used</div>
  </div>
</div>
</div>

<h2>CPU Usage Over Time</h2>
<div class="card"><img src="cpu_over_time.png" alt="CPU over time"></div>

<h2>RAM Usage Over Time</h2>
<div class="card"><img src="ram_over_time.png" alt="RAM over time"></div>

<h2>Consumption vs Number of UAVs</h2>
<div class="card"><img src="consumption_vs_uavs.png" alt="Consumption vs UAVs"></div>

{degradation_section}

<h2>Measurement Data</h2>
<div class="card">
<table>
  <thead><tr>
    <th>Time (min)</th><th>UAVs</th>
    <th>CPU %</th><th>RAM %</th><th>RAM (GB)</th><th>Event</th>
  </tr></thead>
  <tbody>
  {table_rows}
  </tbody>
</table>
</div>

<p style="color:var(--sub);font-size:0.8rem;margin-top:2rem;text-align:center;">
  ArduSim2 Stress Test · Generated {timestamp}
</p>
</body>
</html>
"""

_DEGRADATION_SECTION = """\
<h2>Degradation Period (2 min observation at limit)</h2>
<div class="card"><img src="degradation_period.png" alt="Degradation period"></div>
"""


def generate_report(
    samples: List[MetricPoint],
    degradation_samples: List[MetricPoint],
    max_uavs: int,
) -> str:
    """
    Generate the full report directory.
    Returns the path to index.html.
    """
    timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    report_dir = os.path.join(RESULTS_BASE_DIR, f"report_{timestamp}")
    os.makedirs(report_dir, exist_ok=True)

    # ── charts ────────────────────────────────────────────────────────────
    _chart_cpu_over_time(samples,
                         os.path.join(report_dir, "cpu_over_time.png"))
    _chart_ram_over_time(samples,
                         os.path.join(report_dir, "ram_over_time.png"))
    _chart_consumption_vs_uavs(samples,
                                os.path.join(report_dir, "consumption_vs_uavs.png"))
    if degradation_samples:
        _chart_degradation(degradation_samples,
                           os.path.join(report_dir, "degradation_period.png"))

    # ── CSV ──────────────────────────────────────────────────────────────
    csv_path = os.path.join(report_dir, "data.csv")
    t0 = samples[0].timestamp if samples else 0
    with open(csv_path, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["time_min", "num_uavs", "cpu_pct", "ram_pct",
                          "ram_gb", "ram_total_gb", "event"])
        for s in samples + degradation_samples:
            writer.writerow([
                f"{(s.timestamp - t0) / 60:.2f}",
                s.num_uavs,
                f"{s.cpu_percent:.2f}",
                f"{s.ram_percent:.2f}",
                f"{s.ram_bytes / 1e9:.3f}",
                f"{s.ram_total_bytes / 1e9:.3f}",
                s.event or "",
            ])

    # ── machine specs ─────────────────────────────────────────────────────
    specs = _machine_specs()
    all_samples = samples + degradation_samples

    peak_cpu = max((s.cpu_percent for s in all_samples), default=0)
    peak_ram = max((s.ram_percent for s in all_samples), default=0)
    max_swarms = max_uavs // 2

    # Ratios calculated at the last stable sample (before degradation)
    ref = samples[-1] if samples else None
    cpu_logical = specs["cpu_logical"]
    ram_total_gb_f = float(specs["ram_total_gb"])

    uavs_per_core    = max_uavs / cpu_logical if cpu_logical else 0
    uavs_per_gb      = max_uavs / ram_total_gb_f if ram_total_gb_f else 0
    uavs_per_cpu_pct = (max_uavs / ref.cpu_percent) if ref and ref.cpu_percent else 0

    # ── HTML table rows ───────────────────────────────────────────────────
    rows_html = []
    for s in all_samples:
        tmin = (s.timestamp - t0) / 60
        rows_html.append(
            f"<tr><td>{tmin:.2f}</td><td>{s.num_uavs}</td>"
            f"<td>{s.cpu_percent:.1f}</td><td>{s.ram_percent:.1f}</td>"
            f"<td>{s.ram_bytes / 1e9:.2f}</td>"
            f"<td>{s.event or ''}</td></tr>"
        )

    degradation_section = _DEGRADATION_SECTION if degradation_samples else ""

    html = _HTML_TEMPLATE.format(
        timestamp=datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        max_uavs=max_uavs,
        max_swarms=max_swarms,
        peak_cpu=peak_cpu,
        peak_ram=peak_ram,
        uavs_per_core=uavs_per_core,
        uavs_per_gb=uavs_per_gb,
        uavs_per_cpu_pct=uavs_per_cpu_pct,
        degradation_section=degradation_section,
        table_rows="\n  ".join(rows_html),
        **specs,
    )

    html_path = os.path.join(report_dir, "index.html")
    with open(html_path, "w", encoding="utf-8") as f:
        f.write(html)

    print(f"\n[report] Report written to: {html_path}")
    return html_path
