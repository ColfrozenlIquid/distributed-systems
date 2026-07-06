import json

import matplotlib.pyplot as plt
import pandas as pd

# -----------------------------
# Load log
# -----------------------------
FILE = "./metrics/mqtt_trace_chaos_1783348429.jsonl"  # change to your filename

records = []

with open(FILE) as f:
    for line in f:
        if line.strip():
            records.append(json.loads(line))

df = pd.DataFrame(records)

# -----------------------------
# Parse timestamps
# -----------------------------
df["arrival_time"] = pd.to_datetime(df["arrival_time"])

if "origin_timestamp" in df.columns:
    df["origin_timestamp"] = pd.to_datetime(df["origin_timestamp"])

# -----------------------------
# Network delay
# -----------------------------
if "origin_timestamp" in df.columns:
    df["network_delay_ms"] = (
        df["arrival_time"] - df["origin_timestamp"]
    ).dt.total_seconds() * 1000

# -----------------------------
# Overall statistics
# -----------------------------
print("\n===== Overall =====")

print("Messages:", len(df))
print("Bytes:", df["size_bytes"].sum())
print("Average message size:", df["size_bytes"].mean())

duration = (df["arrival_time"].max() - df["arrival_time"].min()).total_seconds()

print("Duration:", duration)
print("Messages/sec:", len(df) / duration)

# -----------------------------
# Message types
# -----------------------------
print("\n===== Message Types =====")

print(df["msg_type"].value_counts())

# -----------------------------
# Topic statistics
# -----------------------------
print("\n===== Topics =====")

topic_stats = (
    df.groupby("topic")
    .agg(
        messages=("topic", "count"),
        bytes=("size_bytes", "sum"),
        avg_size=("size_bytes", "mean"),
    )
    .sort_values("messages", ascending=False)
)

print(topic_stats)

# -----------------------------
# Sender statistics
# -----------------------------
print("\n===== Senders =====")

sender_col = None

for c in ["from_id", "actor_id"]:
    if c in df.columns:
        sender_col = c
        break

if sender_col:
    print(df[sender_col].value_counts())

# -----------------------------
# Delay statistics
# -----------------------------
if "network_delay_ms" in df.columns:
    print("\n===== Network Delay =====")

    print(df["network_delay_ms"].describe())

# -----------------------------
# Messages over time
# -----------------------------
timeline = df.set_index("arrival_time").resample("1s").size()

plt.figure(figsize=(12, 5))
timeline.plot()
plt.title("Messages per Second")
plt.ylabel("Messages")
plt.tight_layout()

# -----------------------------
# Bytes over time
# -----------------------------
bytes_timeline = df.set_index("arrival_time").resample("1s")["size_bytes"].sum()

plt.figure(figsize=(12, 5))
bytes_timeline.plot()
plt.title("Bytes per Second")
plt.ylabel("Bytes")
plt.tight_layout()

# -----------------------------
# Delay histogram
# -----------------------------
if "network_delay_ms" in df.columns:
    plt.figure(figsize=(8, 5))
    df["network_delay_ms"].hist(bins=50)

    plt.title("Network Delay")
    plt.xlabel("ms")
    plt.ylabel("Messages")

# -----------------------------
# Message type distribution
# -----------------------------
plt.figure(figsize=(6, 6))
df["msg_type"].value_counts().plot(kind="pie", autopct="%1.1f%%")

plt.ylabel("")
plt.title("Message Types")

plt.show()
