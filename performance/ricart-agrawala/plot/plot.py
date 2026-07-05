"""
Compare OBSERVED handoff latency (sync_delay_ms) against the THEORETICAL
latency bounds for Ricart-Agrawala, for each network scenario / cluster size.

--------------------------------------------------------------------------
WHY THE ORIGINAL MODEL DIDN'T FIT
--------------------------------------------------------------------------
The original script computed a single number:

    THEORETICAL_MS = 2 * (N-1) * BASE_LATENCY_MS

This is the *message-complexity* bound (2*(N-1) messages), which is only
equal to wall-clock time if every REQUEST/REPLY pair is sent and awaited
strictly one-after-another (serial dispatch).

The data doesn't support that assumption:

  - In the "chaos" file, the number of nodes actually contended with (the
    `k` column) ranges from 0 to 19, but correlation(k, sync_delay_ms) is
    ~0.09 -- essentially zero. If dispatch were serial, more contenders
    would mean proportionally more delay. It doesn't, which means REQUESTs
    are sent concurrently and the node waits on the slowest single round
    trip, not N-1 sequential ones.

  - In "uniform" and "fast", k is constant (~19) for almost every row, yet
    sync_delay_ms is clearly BIMODAL: a large fraction of entries finish in
    single-digit ms (fast-path grant, no real waiting), and the rest sit on
    a fixed plateau (~300-700ms) that does not scale with the configured
    2ms/20ms base latency. That plateau looks like a fixed timeout/retry
    constant in the implementation, not a network-latency effect at all.

Given that, a single fitted theoretical point is the wrong tool: it invites
comparing a blended mean of two different regimes to a number that
describes neither. Instead this version:

  1. Computes a BOUNDING ENVELOPE per scenario:
       lower = 2 * BASE_LATENCY_MS                  (best case: fully
                                                       concurrent dispatch,
                                                       one round trip)
       upper = 2 * (N_NODES - 1) * BASE_LATENCY_MS  (worst case: fully
                                                       serial dispatch)
     N_NODES is now taken PER FILE, not from a single global constant,
     since the dataset includes both 10-node and 20-node cluster runs.
  2. Plots the full observed DISTRIBUTION (box + strip of points) against
     that envelope, instead of collapsing everything to a mean.
  3. Reports what fraction of observed entries fall below / inside / above
     the envelope, so you can see *how* it fails to fit, not just that it
     does.
  4. Flags negative sync_delay_ms values (clock-skew artifacts between
     simulated node clocks) rather than silently averaging them in.

--------------------------------------------------------------------------
PARAMETERS (from the Go simulator's scenario config):
--------------------------------------------------------------------------
    scenarios := map[string]Scenario{
        "uniform": {Latency: 20 * time.Millisecond, Jitter: 5 * time.Millisecond},
        "fast":    {Latency: 2  * time.Millisecond, Jitter: 1 * time.Millisecond},
        "chaos":   {Latency: 100 * time.Millisecond, Jitter: 50 * time.Millisecond},
    }
--------------------------------------------------------------------------
"""

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

# ---------------------------------------------------------------------------
# 1. Load data
# ---------------------------------------------------------------------------
# Labels encode both the scenario type and the cluster size used for that
# run ("<Scenario> <N_NODES>"), since latency scenario and node count are
# now independent axes in the dataset.

FILES = {
    "Uniform 10": "./metrics/sync_delay_uniform_10.csv",
    "Uniform 20": "./metrics/sync_delay_uniform_20.csv",
    "Fast 10": "./metrics/sync_delay_fast_10.csv",
    "Fast 20": "./metrics/sync_delay_fast_20.csv",
    "Chaos 10": "./metrics/sync_delay_chaos_10.csv",
    "Chaos 20": "./metrics/sync_delay_chaos_20.csv",
}

COLORS_OBSERVED = {"Chaos": "#e0554f", "Fast": "#3fa796", "Uniform": "#5b7fd6"}
COLOR_ENVELOPE = "#a3a3a3"

BASE_LATENCY_MS = {
    "Uniform": 20,
    "Fast": 2,
    "Chaos": 100,
}


def parse_label(label):
    """'Uniform 20' -> ('Uniform', 20). Keeps scenario-type lookups
    (color, base latency) separate from the per-file node count, so this
    keeps working no matter how many different N_NODES variants exist."""
    scenario_type, n_nodes_str = label.rsplit(" ", 1)
    return scenario_type, int(n_nodes_str)


frames = {label: pd.read_csv(path) for label, path in FILES.items()}

# ---------------------------------------------------------------------------
# 2. Diagnostics: envelope bounds + how observed data relates to them
# ---------------------------------------------------------------------------

scenarios = list(FILES.keys())
bounds = {}
node_counts = {}
report_rows = []

for label in scenarios:
    scenario_type, n_nodes = parse_label(label)
    node_counts[label] = n_nodes

    df = frames[label]
    base = BASE_LATENCY_MS[scenario_type]
    lower = 2 * base
    upper = 2 * (n_nodes - 1) * base
    bounds[label] = (lower, upper)

    obs = df["sync_delay_ms"]
    n_negative = int((obs < 0).sum())
    obs_valid = obs[obs >= 0]  # exclude clock-skew artifacts from the % breakdown

    pct_below = (obs_valid < lower).mean() * 100
    pct_within = ((obs_valid >= lower) & (obs_valid <= upper)).mean() * 100
    pct_above = (obs_valid > upper).mean() * 100

    report_rows.append(
        {
            "scenario": label,
            "n_nodes": n_nodes,
            "n": len(df),
            "n_negative_clock_skew": n_negative,
            "mean_ms": obs.mean(),
            "median_ms": obs.median(),
            "lower_bound_ms": lower,
            "upper_bound_ms": upper,
            "pct_below_lower": pct_below,
            "pct_within_envelope": pct_within,
            "pct_above_upper": pct_above,
        }
    )

report = pd.DataFrame(report_rows).set_index("scenario")
pd.set_option("display.float_format", lambda v: f"{v:,.1f}")
print("=" * 78)
print("DIAGNOSTIC REPORT: observed sync_delay_ms vs. concurrent/serial envelope")
print("=" * 78)
print(report.to_string())
print()
for label in scenarios:
    lower, upper = bounds[label]
    n_neg = report.loc[label, "n_negative_clock_skew"]
    n_nodes = node_counts[label]
    print(
        f"{label:12s} (N={n_nodes:2d}): envelope [{lower:.0f}, {upper:.0f}] ms  |  "
        f"{report.loc[label, 'pct_below_lower']:.0f}% faster than best case, "
        f"{report.loc[label, 'pct_within_envelope']:.0f}% inside envelope, "
        f"{report.loc[label, 'pct_above_upper']:.0f}% slower than worst case"
        + (
            f"  ({n_neg} negative/clock-skew rows excluded from %s)" % "%"
            if n_neg
            else ""
        )
    )
print()

# ---------------------------------------------------------------------------
# 3. Bar plot: observed (mean, median) vs. theoretical lower/upper bound
# ---------------------------------------------------------------------------

x = np.arange(len(scenarios))
width = 0.2

fig, ax = plt.subplots(figsize=(11, 6))

obs_mean = [report.loc[s, "mean_ms"] for s in scenarios]
obs_median = [report.loc[s, "median_ms"] for s in scenarios]
lower_vals = [bounds[s][0] for s in scenarios]
upper_vals = [bounds[s][1] for s in scenarios]
bar_colors = [COLORS_OBSERVED[parse_label(s)[0]] for s in scenarios]

bars_mean = ax.bar(
    x - 1.5 * width,
    obs_mean,
    width,
    label="Observed (mean)",
    color=bar_colors,
)
bars_median = ax.bar(
    x - 0.5 * width,
    obs_median,
    width,
    label="Observed (median)",
    color=bar_colors,
    alpha=0.55,
)
bars_lower = ax.bar(
    x + 0.5 * width,
    lower_vals,
    width,
    label="Theoretical lower bound\n(2\u00d7base, concurrent dispatch)",
    color=COLOR_ENVELOPE,
    hatch="//",
)
bars_upper = ax.bar(
    x + 1.5 * width,
    upper_vals,
    width,
    label="Theoretical upper bound\n(2\u00d7(N-1)\u00d7base, serial dispatch)",
    color=COLOR_ENVELOPE,
    hatch="xx",
    alpha=0.6,
)

for bars in (bars_mean, bars_median, bars_lower, bars_upper):
    for bar in bars:
        h = bar.get_height()
        ax.annotate(
            f"{h:,.0f}",
            xy=(bar.get_x() + bar.get_width() / 2, h),
            xytext=(0, 4),
            textcoords="offset points",
            ha="center",
            fontsize=8,
        )

# x-tick labels include node count per bar group since N now varies per
# scenario, and the upper-bound formula depends on it.
xtick_labels = [f"{s}\n(N={node_counts[s]})" for s in scenarios]
ax.set_xticks(x)
ax.set_xticklabels(xtick_labels)
ax.set_ylabel("Handoff time (ms)")
ax.set_title(
    "Ricart\u2013Agrawala Handoff Latency: Observed vs. Theoretical Bounds\n"
    "(lower = concurrent dispatch, upper = serial dispatch; N varies by run)",
    fontsize=13,
)
ax.legend(frameon=False, fontsize=8, loc="upper left")
ax.grid(alpha=0.3, axis="y")

plt.tight_layout()
plt.savefig("sync_delay_vs_theoretical.png", dpi=180, bbox_inches="tight")
print("Saved sync_delay_vs_theoretical.png")
