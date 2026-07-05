#!/usr/bin/env python3
"""
Benchmark Comparison Chart Generator
Reads benchmark_results.json and produces a bar chart comparing
throughput (msgs/sec) across transport methods or MQTT fan-out scenarios.

Usage:
    python plot_benchmarks.py
    python plot_benchmarks.py --input path/to/results.json --output chart.png
    python plot_benchmarks.py --run 2   # plot a specific run_id
"""

import argparse
import json
import re
import sys
from pathlib import Path

import matplotlib

matplotlib.use("Agg")  # headless — swap to "TkAgg" if you want an interactive window
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import numpy as np

# ── colour palette (one per transport) ────────────────────────────────────────
PALETTE = {
    "stream_grpc": "#4C9BE8",
    "unary_grpc": "#E87C4C",
    "http": "#6DBE6D",
    "mqtt": "#B46DE8",
}

# Palette for grouped/faceted plots (e.g. QoS levels)
GROUP_PALETTE = {
    "qos0": "#4C9BE8",
    "qos1": "#E87C4C",
}

FALLBACK_COLOURS = plt.cm.tab10.colors


def load_results(path: str) -> dict:
    with open(path) as f:
        return json.load(f)


def pick_run(data: dict, run_id: int | None) -> dict:
    runs = data["benchmark_runs"]
    if not runs:
        sys.exit("No benchmark runs found in the file.")
    if run_id is None:
        return runs[-1]  # default: latest run
    for r in runs:
        if r["run_id"] == run_id:
            return r
    sys.exit(f"run_id {run_id} not found.")


def detect_mqtt_fanout(results: dict) -> bool:
    """Return True if all keys follow the mqtt_fanout_<N>_subs_qos<M> pattern."""
    pattern = re.compile(r"^mqtt_fanout_(\d+)_subs_qos(\d+)$")
    return all(pattern.match(k) for k in results)


def parse_mqtt_fanout(results: dict) -> tuple[list[int], dict[str, dict]]:
    """
    Parse MQTT fan-out keys into:
      - sorted list of subscriber counts
      - dict keyed by qos label -> {sub_count: result_dict}
    """
    pattern = re.compile(r"^mqtt_fanout_(\d+)_subs_qos(\d+)$")
    by_qos: dict[str, dict[int, dict]] = {}

    for key, val in results.items():
        m = pattern.match(key)
        if not m:
            continue
        n_subs = int(m.group(1))
        qos = f"qos{m.group(2)}"
        by_qos.setdefault(qos, {})[n_subs] = val

    all_counts = sorted({n for qos_data in by_qos.values() for n in qos_data})
    return all_counts, by_qos


def plot_mqtt_fanout(run: dict, output_path: str) -> None:
    """Grouped bar chart: X = subscriber count, groups = QoS level."""
    results = run["results"]
    run_id = run["run_id"]
    timestamp = run.get("timestamp", "")

    sub_counts, by_qos = parse_mqtt_fanout(results)
    qos_labels = sorted(by_qos.keys())  # e.g. ["qos0", "qos1"]
    n_groups = len(sub_counts)
    n_bars = len(qos_labels)

    bar_width = 0.35
    group_spacing = 0.1
    total_width = n_bars * bar_width + group_spacing
    x = np.arange(n_groups) * total_width

    fig, ax = plt.subplots(figsize=(13, 6))
    fig.patch.set_facecolor("#1A1A2E")
    ax.set_facecolor("#16213E")

    for i, qos in enumerate(qos_labels):
        qos_data = by_qos[qos]
        speeds = [qos_data.get(n, {}).get("msgs_per_sec", 0) for n in sub_counts]
        elapsed = [qos_data.get(n, {}).get("elapsed_seconds", 0) for n in sub_counts]
        offset = (i - (n_bars - 1) / 2) * bar_width
        colour = GROUP_PALETTE.get(qos, FALLBACK_COLOURS[i % len(FALLBACK_COLOURS)])

        bars = ax.bar(
            x + offset,
            speeds,
            width=bar_width,
            color=colour,
            edgecolor="white",
            linewidth=0.6,
            zorder=3,
            label=qos.upper().replace("QOS", "QoS "),
        )

        # value labels above each bar
        all_speeds = [v["msgs_per_sec"] for qd in by_qos.values() for v in qd.values()]
        max_speed = max(all_speeds) if all_speeds else 1

        for bar, spd, ela in zip(bars, speeds, elapsed):
            if spd == 0:
                continue
            label = (
                f"{spd:,.0f}\n({ela:.2f}s)"
                if spd < 10_000
                else f"{spd / 1000:.1f}k\n({ela:.2f}s)"
            )
            ax.text(
                bar.get_x() + bar.get_width() / 2,
                bar.get_height() + max_speed * 0.012,
                label,
                ha="center",
                va="bottom",
                fontsize=7.5,
                color="white",
                fontweight="bold",
                linespacing=1.4,
            )

    # grid & spines
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda v, _: f"{v:,.0f}"))
    ax.grid(axis="y", color="white", alpha=0.12, linestyle="--", zorder=0)
    for spine in ax.spines.values():
        spine.set_edgecolor("#444466")

    x_tick_labels = [f"{n:,}" for n in sub_counts]
    ax.set_xticks(x)
    ax.set_xticklabels(x_tick_labels, color="white", fontsize=11)
    ax.tick_params(axis="y", colors="white", labelsize=10)

    all_speeds = [v["msgs_per_sec"] for qd in by_qos.values() for v in qd.values()]
    ax.set_ylim(0, max(all_speeds) * 1.30)

    ax.set_title(
        "MQTT Fan-out Throughput  –  QoS 0 vs QoS 1 by Subscriber Count",
        color="white",
        fontsize=15,
        fontweight="bold",
        pad=16,
    )
    ax.set_ylabel("Messages / second", color="white", fontsize=11, labelpad=10)
    ax.set_xlabel("Subscriber Count", color="white", fontsize=11, labelpad=10)

    legend = ax.legend(
        fontsize=11,
        framealpha=0.25,
        facecolor="#1A1A2E",
        edgecolor="#444466",
        labelcolor="white",
    )

    num_messages = next(iter(results.values()))["num_messages"]
    fig.text(
        0.99,
        0.01,
        f"{num_messages:,} messages per test",
        ha="right",
        va="bottom",
        color="#AAAACC",
        fontsize=8,
    )
    if timestamp:
        fig.text(
            0.01,
            0.01,
            f"Recorded: {timestamp}",
            ha="left",
            va="bottom",
            color="#AAAACC",
            fontsize=8,
        )

    plt.tight_layout()
    plt.savefig(
        output_path, dpi=150, bbox_inches="tight", facecolor=fig.get_facecolor()
    )
    print(f"Chart saved → {output_path}")


def plot_generic(run: dict, output_path: str) -> None:
    """Original single-bar chart for mixed transport results."""
    results = run["results"]
    timestamp = run.get("timestamp", "")

    keys = list(results.keys())
    labels = [results[k]["label"] for k in keys]
    speeds = [results[k]["msgs_per_sec"] for k in keys]
    counts = [results[k]["num_messages"] for k in keys]
    secs = [results[k]["elapsed_seconds"] for k in keys]
    colours = [
        PALETTE.get(k, FALLBACK_COLOURS[i % len(FALLBACK_COLOURS)])
        for i, k in enumerate(keys)
    ]

    x = np.arange(len(keys))
    bar_width = 0.55

    fig, ax = plt.subplots(figsize=(10, 6))
    fig.patch.set_facecolor("#1A1A2E")
    ax.set_facecolor("#16213E")

    bars = ax.bar(
        x,
        speeds,
        width=bar_width,
        color=colours,
        edgecolor="white",
        linewidth=0.6,
        zorder=3,
    )

    for bar, speed, elapsed, n in zip(bars, speeds, secs, counts):
        ax.text(
            bar.get_x() + bar.get_width() / 2,
            bar.get_height() + max(speeds) * 0.015,
            f"{speed:,.0f}\nmsgs/s\n({elapsed:.2f}s)",
            ha="center",
            va="bottom",
            fontsize=9,
            color="white",
            fontweight="bold",
            linespacing=1.4,
        )

    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda v, _: f"{v:,.0f}"))
    ax.grid(axis="y", color="white", alpha=0.12, linestyle="--", zorder=0)
    for spine in ax.spines.values():
        spine.set_edgecolor("#444466")

    ax.set_xticks(x)
    ax.set_xticklabels(labels, color="white", fontsize=12)
    ax.tick_params(axis="y", colors="white", labelsize=10)
    ax.set_ylim(0, max(speeds) * 1.30)

    ax.set_title(
        "Transport Throughput Comparison  –  Run with ~20ms RTT latency",
        color="white",
        fontsize=15,
        fontweight="bold",
        pad=16,
    )
    ax.set_ylabel("Messages / second", color="white", fontsize=11, labelpad=10)
    ax.set_xlabel("Transport Method", color="white", fontsize=11, labelpad=10)

    msg_count = counts[0] if len(set(counts)) == 1 else "variable"
    note = (
        f"{msg_count:,} messages per test"
        if isinstance(msg_count, int)
        else "message counts vary per test"
    )
    fig.text(0.99, 0.01, note, ha="right", va="bottom", color="#AAAACC", fontsize=8)
    if timestamp:
        fig.text(
            0.01,
            0.01,
            f"Recorded: {timestamp}",
            ha="left",
            va="bottom",
            color="#AAAACC",
            fontsize=8,
        )

    plt.tight_layout()
    plt.savefig(
        output_path, dpi=150, bbox_inches="tight", facecolor=fig.get_facecolor()
    )
    print(f"Chart saved → {output_path}")


def plot(run: dict, output_path: str) -> None:
    if detect_mqtt_fanout(run["results"]):
        plot_mqtt_fanout(run, output_path)
    else:
        plot_generic(run, output_path)


def main():
    parser = argparse.ArgumentParser(description="Plot transport benchmark results.")
    parser.add_argument(
        "--input",
        default="benchmark_results.json",
        help="Path to the JSON results file (default: benchmark_results.json)",
    )
    parser.add_argument(
        "--output",
        default="benchmark_chart.png",
        help="Output image path (default: benchmark_chart.png)",
    )
    parser.add_argument(
        "--run",
        type=int,
        default=None,
        help="run_id to plot (default: latest)",
    )
    args = parser.parse_args()

    data = load_results(args.input)
    run = pick_run(data, args.run)
    plot(run, args.output)


if __name__ == "__main__":
    main()
