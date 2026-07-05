#!/usr/bin/env python3
"""
Plot Ricart-Agrawala critical section acquisition delays.

Expects a CSV with columns:
release_node, release_ts, enter_node, enter_ts, sync_delay_ms, recv_delay_ms

Usage:
    python plot_ra_delays.py data.csv
"""

import csv
import sys
from datetime import datetime

import matplotlib.dates as mdates
import matplotlib.pyplot as plt


def parse_ts(ts):
    # Handle variable fractional-second precision and trailing Z
    ts = ts.replace("Z", "")
    if "." in ts:
        base, frac = ts.split(".")
        frac = (frac + "000000")[:6]  # pad/truncate to microseconds
        ts = f"{base}.{frac}"
    return datetime.fromisoformat(ts)


def load_data(path):
    rows = []
    with open(path, newline="") as f:
        reader = csv.DictReader(f)
        for r in reader:
            rows.append(
                {
                    "release_node": r["release_node"],
                    "release_ts": parse_ts(r["release_ts"]),
                    "enter_node": r["enter_node"],
                    "enter_ts": parse_ts(r["enter_ts"]),
                    "sync_delay_ms": float(r["sync_delay_ms"]),
                    "recv_delay_ms": float(r["recv_delay_ms"]),
                }
            )
    return rows


def main():
    path = sys.argv[1] if len(sys.argv) > 1 else "data.csv"
    rows = load_data(path)

    times = [r["release_ts"] for r in rows]
    sync_delays = [r["sync_delay_ms"] for r in rows]
    recv_delays = [r["recv_delay_ms"] for r in rows]
    labels = [f"{r['release_node']}->{r['enter_node']}" for r in rows]

    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(12, 9), sharex=True)

    # --- Plot 1: delays over time as lines ---
    ax1.plot(times, sync_delays, marker="o", label="sync_delay_ms")
    ax1.plot(times, recv_delays, marker="s", label="recv_delay_ms")
    ax1.set_ylabel("Delay (ms)")
    ax1.set_title("Critical Section Release -> Enter Delay Over Time")
    ax1.legend()
    ax1.grid(True, alpha=0.3)

    # --- Plot 2: bar chart per transition, annotated with node pair ---
    x = range(len(rows))
    width = 0.4
    ax2.bar([i - width / 2 for i in x], sync_delays, width=width, label="sync_delay_ms")
    ax2.bar([i + width / 2 for i in x], recv_delays, width=width, label="recv_delay_ms")
    ax2.set_xticks(list(x))
    ax2.set_xticklabels(labels, rotation=45, ha="right")
    ax2.set_ylabel("Delay (ms)")
    ax2.set_xlabel("Release node -> Enter node (per event)")
    ax2.set_title("Per-Transition CS Acquisition Delay")
    ax2.legend()
    ax2.grid(True, alpha=0.3, axis="y")

    ax1.xaxis.set_major_formatter(mdates.DateFormatter("%H:%M:%S"))
    fig.autofmt_xdate()

    plt.tight_layout()
    out_file = "ra_delays.png"
    plt.savefig(out_file, dpi=150)
    print(f"Saved plot to {out_file}")
    plt.show()


if __name__ == "__main__":
    main()
