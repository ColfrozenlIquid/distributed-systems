import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

# ---------------------------------------------------------------------------
# 1. Load data
# ---------------------------------------------------------------------------

FILES = {
    "Uniform": "./metrics/sync_delay_uniform_20.csv",
    "Fast": "./metrics/sync_delay_fast_20.csv",
    "Chaos": "./metrics/sync_delay_chaos_20.csv",
}

COLORS_OBSERVED = {"Chaos": "#e0554f", "Fast": "#3fa796", "Uniform": "#5b7fd6"}
COLOR_ENVELOPE = "#a3a3a3"

N_NODES = 20  # <-- change if your cluster size differs

BASE_LATENCY_MS = {
    "Uniform": 20,
    "Fast": 2,
    "Chaos": 100,
}

frames = {label: pd.read_csv(path) for label, path in FILES.items()}

# ---------------------------------------------------------------------------
# 2. Diagnostics: envelope bounds + how observed data relates to them
# ---------------------------------------------------------------------------

scenarios = list(FILES.keys())
bounds = {}
report_rows = []

for label in scenarios:
    df = frames[label]
    lower = 2 * BASE_LATENCY_MS[label]
    upper = 2 * (N_NODES - 1) * BASE_LATENCY_MS[label]
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
    print(
        f"{label:8s}: envelope [{lower:.0f}, {upper:.0f}] ms  |  "
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
# 3. Plot: full distribution per scenario vs. envelope band
# ---------------------------------------------------------------------------

fig, axes = plt.subplots(1, len(scenarios), figsize=(13, 6), sharey=False)

for ax, label in zip(axes, scenarios):
    df = frames[label]
    obs = df["sync_delay_ms"]
    lower, upper = bounds[label]

    # shaded envelope band: [2*base, 2*(N-1)*base]
    ax.axhspan(
        lower,
        upper,
        color=COLOR_ENVELOPE,
        alpha=0.25,
        zorder=0,
        label="Theoretical envelope\n(concurrent \u2192 serial dispatch)",
    )
    ax.axhline(lower, color=COLOR_ENVELOPE, lw=1, ls="--")
    ax.axhline(upper, color=COLOR_ENVELOPE, lw=1, ls="--")

    # jittered strip of raw points + boxplot on top
    x_jitter = np.random.normal(1, 0.05, size=len(obs))
    ax.scatter(
        x_jitter,
        obs,
        s=14,
        alpha=0.5,
        color=COLORS_OBSERVED[label],
        zorder=2,
        linewidths=0,
    )
    bp = ax.boxplot(
        obs,
        positions=[1],
        widths=0.5,
        showfliers=False,
        patch_artist=True,
        zorder=3,
    )
    for box in bp["boxes"]:
        box.set(facecolor=COLORS_OBSERVED[label], alpha=0.35)
    for median in bp["medians"]:
        median.set(color="black", lw=2)

    ax.set_title(
        f"{label}\nbase latency = {BASE_LATENCY_MS[label]} ms",
        fontsize=12,
    )
    ax.set_xticks([])
    ax.set_ylabel("sync_delay_ms")
    ax.annotate(
        f"envelope: {lower:.0f}\u2013{upper:.0f} ms",
        xy=(1.28, upper),
        xycoords=("axes fraction", "data"),
        fontsize=8,
        color="#666666",
        ha="left",
        va="top",
        clip_on=False,
    )

fig.suptitle(
    "Ricart\u2013Agrawala Handoff Latency: Observed Distribution vs.\n"
    f"Concurrent\u2013Serial Theoretical Envelope (N={N_NODES} nodes)",
    fontsize=13,
)
fig.tight_layout(rect=[0, 0, 1, 0.9])
plt.savefig("sync_delay_vs_theoretical.png", dpi=180, bbox_inches="tight")
print("Saved sync_delay_vs_theoretical.png")
