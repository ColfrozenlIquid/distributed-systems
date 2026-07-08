import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

# ---------------------------------------------------------------------------
# 1. Load data
# ---------------------------------------------------------------------------
# Labels encode both the scenario type and the cluster size used for that
# run ("<Scenario> <N_NODES>"), since latency scenario and node count are
# independent axes in the dataset (node count is tracked for context/plot
# labeling even though it no longer feeds a bound).

FILES = {
    "Uniform 10": "./metrics/sync_delay_uniform_10.csv",
    "Uniform 20": "./metrics/sync_delay_uniform_20.csv",
    "Uniform 64": "./metrics/sync_delay_uniform_64.csv",
    "Fast 10": "./metrics/sync_delay_fast_10.csv",
    "Fast 20": "./metrics/sync_delay_fast_20.csv",
    "Fast 64": "./metrics/sync_delay_fast_64.csv",
    "Chaos 10": "./metrics/sync_delay_chaos_10.csv",
    "Chaos 20": "./metrics/sync_delay_chaos_20.csv",
    "Chaos 64": "./metrics/sync_delay_chaos_64.csv",
}

COLORS_OBSERVED = {"Chaos": "#e0554f", "Fast": "#3fa796", "Uniform": "#5b7fd6"}
COLOR_BOUND = "#a3a3a3"

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
# 2. Diagnostics: lower bound + how observed data relates to it
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
    bounds[label] = lower

    obs = df["sync_delay_ms"]
    n_negative = int((obs < 0).sum())
    obs_valid = obs[obs >= 0]  # exclude clock-skew artifacts from the % breakdown

    pct_below = (obs_valid < lower).mean() * 100
    pct_at_or_near = ((obs_valid >= lower) & (obs_valid <= lower * 1.5)).mean() * 100
    pct_above = (obs_valid > lower * 1.5).mean() * 100

    report_rows.append(
        {
            "scenario": label,
            "n_nodes": n_nodes,
            "n": len(df),
            "n_negative_clock_skew": n_negative,
            "mean_ms": obs.mean(),
            "median_ms": obs.median(),
            "lower_bound_ms": lower,
            "pct_below_lower": pct_below,
            "pct_at_or_near_lower": pct_at_or_near,
            "pct_well_above_lower": pct_above,
        }
    )

report = pd.DataFrame(report_rows).set_index("scenario")
pd.set_option("display.float_format", lambda v: f"{v:,.1f}")
print("=" * 78)
print("DIAGNOSTIC REPORT: observed sync_delay_ms vs. concurrent-dispatch lower bound")
print("=" * 78)
print(report.to_string())
print()
for label in scenarios:
    lower = bounds[label]
    n_neg = report.loc[label, "n_negative_clock_skew"]
    n_nodes = node_counts[label]
    print(
        f"{label:12s} (N={n_nodes:2d}): lower bound {lower:.0f} ms  |  "
        f"{report.loc[label, 'pct_below_lower']:.0f}% faster than theoretical minimum, "
        f"{report.loc[label, 'pct_at_or_near_lower']:.0f}% at/near minimum, "
        f"{report.loc[label, 'pct_well_above_lower']:.0f}% well above minimum (likely retry/timeout plateau)"
        + (
            f"  ({n_neg} negative/clock-skew rows excluded from %s)" % "%"
            if n_neg
            else ""
        )
    )
print()

# ---------------------------------------------------------------------------
# 3. Bar plot: observed (mean, median) vs. theoretical lower bound
# ---------------------------------------------------------------------------

x = np.arange(len(scenarios))
width = 0.25

fig, ax = plt.subplots(figsize=(11, 6))

obs_mean = [report.loc[s, "mean_ms"] for s in scenarios]
obs_median = [report.loc[s, "median_ms"] for s in scenarios]
lower_vals = [bounds[s] for s in scenarios]
bar_colors = [COLORS_OBSERVED[parse_label(s)[0]] for s in scenarios]

bars_mean = ax.bar(
    x - width,
    obs_mean,
    width,
    label="Observed (mean)",
    color=bar_colors,
)
bars_median = ax.bar(
    x,
    obs_median,
    width,
    label="Observed (median)",
    color=bar_colors,
    alpha=0.55,
)
bars_lower = ax.bar(
    x + width,
    lower_vals,
    width,
    label="Theoretical lower bound\n(2\u00d7base, concurrent dispatch)",
    color=COLOR_BOUND,
    hatch="//",
)

for bars in (bars_mean, bars_median, bars_lower):
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

# x-tick labels still include node count per bar group for context, even
# though the bound itself no longer depends on it.
xtick_labels = [f"{s}\n(N={node_counts[s]})" for s in scenarios]
ax.set_xticks(x)
ax.set_xticklabels(xtick_labels)
ax.set_ylabel("Handoff time (ms)")
ax.set_title(
    "Ricart\u2013Agrawala Handoff Latency: Observed vs. Theoretical Lower Bound\n"
    "(lower bound = concurrent dispatch, one round trip; serial-dispatch\n"
    "upper bound omitted -- contention count doesn't correlate with delay)",
    fontsize=13,
)
ax.legend(frameon=False, fontsize=8, loc="upper left")
ax.grid(alpha=0.3, axis="y")

plt.tight_layout()
plt.savefig("sync_delay_vs_theoretical.png", dpi=180, bbox_inches="tight")
print("Saved sync_delay_vs_theoretical.png")
