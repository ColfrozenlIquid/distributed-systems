package internal

import (
	"encoding/json"
	"os"
	"time"
)

type BenchmarkResult struct {
	Label          string  `json:"label"`
	NumMessages    int     `json:"num_messages"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	MsgsPerSec     float64 `json:"msgs_per_sec"`
}

type BenchmarkRun struct {
	RunID     int                        `json:"run_id"`
	Timestamp string                     `json:"timestamp"`
	Results   map[string]BenchmarkResult `json:"results"`
}

type BenchmarkFile struct {
	Runs []BenchmarkRun `json:"benchmark_runs"`
}

func SaveResult(key, label string, numMessages int, elapsed time.Duration) {
	const path = "benchmark_results.json"

	var data BenchmarkFile
	if raw, err := os.ReadFile(path); err == nil {
		json.Unmarshal(raw, &data)
	}

	runID := len(data.Runs) + 1
	if len(data.Runs) > 0 {
		last := &data.Runs[len(data.Runs)-1]
		if last.Timestamp[:10] == time.Now().UTC().Format("2006-01-02") {
			last.Results[key] = makeResult(label, numMessages, elapsed)
			writeFile(path, data)
			return
		}
	}

	run := BenchmarkRun{
		RunID:     runID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Results:   map[string]BenchmarkResult{key: makeResult(label, numMessages, elapsed)},
	}
	data.Runs = append(data.Runs, run)
	writeFile(path, data)
}

func makeResult(label string, n int, elapsed time.Duration) BenchmarkResult {
	return BenchmarkResult{
		Label:          label,
		NumMessages:    n,
		ElapsedSeconds: elapsed.Seconds(),
		MsgsPerSec:     float64(n) / elapsed.Seconds(),
	}
}

func writeFile(path string, data BenchmarkFile) {
	raw, _ := json.MarshalIndent(data, "", "  ")
	os.WriteFile(path, raw, 0644)
}
