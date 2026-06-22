#!/usr/bin/env python3
"""
Benchmark Comparison Chart Generator
Reads benchmark_results.json and produces a bar chart comparing
throughput (msgs/sec) across transport methods.

Usage:
    python plot_benchmarks.py
    python plot_benchmarks.py --input path/to/results.json --output chart.png
    python plot_benchmarks.py --run 2   # plot a specific run_id
"""

import argparse
import json
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


def plot(run: dict, output_path: str) -> None:
    results = run["results"]
    run_id = run["run_id"]
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

    # value labels above each bar
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

    # grid & spines
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda v, _: f"{v:,.0f}"))
    ax.grid(axis="y", color="white", alpha=0.12, linestyle="--", zorder=0)
    for spine in ax.spines.values():
        spine.set_edgecolor("#444466")

    ax.set_xticks(x)
    ax.set_xticklabels(labels, color="white", fontsize=12)
    ax.tick_params(axis="y", colors="white", labelsize=10)

    # headroom so labels don't clip
    ax.set_ylim(0, max(speeds) * 1.30)

    # titles
    ax.set_title(
        f"Transport Throughput Comparison  –  Run with ~20ms RTT latency",
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
        "--run", type=int, default=None, help="run_id to plot (default: latest)"
    )
    args = parser.parse_args()

    data = load_results(args.input)
    run = pick_run(data, args.run)
    plot(run, args.output)


if __name__ == "__main__":
    main()
