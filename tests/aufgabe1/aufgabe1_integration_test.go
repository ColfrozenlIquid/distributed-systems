package aufgabe1tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"testing"
	"time"
)

type position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type sensorInfo struct {
	ID         string   `json:"id"`
	Position   position `json:"position"`
	SensorType string   `json:"sensor_type"`
}

type actorInfo struct {
	ID        string   `json:"id"`
	Position  position `json:"position"`
	ActorType string   `json:"actor_type"`
}

type sensorRegistration struct {
	Sensor sensorInfo `json:"sensor"`
}

type actorRegistration struct {
	Actor actorInfo `json:"actor"`
}

type sensorReading struct {
	Timestamp time.Time  `json:"timestamp"`
	Sensor    sensorInfo `json:"sensor"`
	Value     float64    `json:"value"`
}

type sensorIncident struct {
	Timestamp time.Time  `json:"timestamp"`
	Sensor    sensorInfo `json:"sensor"`
	Incident  int        `json:"incident"`
}

type systemStatus struct {
	Sensors   []sensorInfo     `json:"sensors"`
	Actors    []actorInfo      `json:"actors"`
	Readings  []sensorReading  `json:"readings"`
	Incidents []sensorIncident `json:"incidents"`
}

func TestRegistrationCriticalReadingAndDashboard(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}
	baseURL := controlCenterURL()
	waitForHealth(t, client, baseURL)

	sensor := sensorInfo{
		ID:         fmt.Sprintf("it-water-%d", time.Now().UnixNano()),
		Position:   position{X: 3, Y: 3},
		SensorType: "water_level",
	}
	actor := actorInfo{
		ID:        fmt.Sprintf("it-actor-%d", time.Now().UnixNano()),
		Position:  position{X: 10, Y: 10},
		ActorType: "rescue_vehicle",
	}
	reading := sensorReading{
		Timestamp: time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		Sensor:    sensor,
		Value:     95,
	}

	doJSON(t, client, http.MethodPost, baseURL+"/register/sensor", sensorRegistration{Sensor: sensor}, http.StatusNoContent)
	doJSON(t, client, http.MethodPost, baseURL+"/register/actor", actorRegistration{Actor: actor}, http.StatusNoContent)
	doJSON(t, client, http.MethodPost, baseURL+"/reading", reading, http.StatusNoContent)

	status := getStatus(t, client, baseURL)
	assertContains(t, "registered sensor", slices.ContainsFunc(status.Sensors, func(s sensorInfo) bool { return s.ID == sensor.ID && s.SensorType == ("water_level") }))
	assertContains(t, "registered actor", slices.ContainsFunc(status.Actors, func(a actorInfo) bool { return a.ID == actor.ID && a.ActorType == ("rescue_vehicle") }))
	assertContains(t, "critical water reading", slices.ContainsFunc(status.Readings, func(r sensorReading) bool { return r.Sensor.ID == sensor.ID && r.Value == 95 }))
	assertContains(t, "water level incident", slices.ContainsFunc(status.Incidents, func(i sensorIncident) bool { return i.Sensor.ID == sensor.ID && i.Incident == 1 }))

	mapBody, mapContentType := doGET(t, client, baseURL+"/map", http.StatusOK)
	if !bytes.Contains(bytes.ToLower(mapBody), []byte("<html")) && !bytes.Contains(bytes.ToLower(mapBody), []byte("<!doctype html")) {
		t.Fatalf("expected /map to return renderable HTML, got first bytes: %.80q", string(mapBody))
	}

	t.Logf("functional result: registered_sensor=%s registered_actor=%s critical_value=%.1f incident=WaterLevelAlert", sensor.ID, actor.ID, reading.Value)
	t.Logf("status summary: sensors=%d actors=%d readings=%d incidents=%d", len(status.Sensors), len(status.Actors), len(status.Readings), len(status.Incidents))
	t.Logf("dashboard result: content_type=%q bytes=%d", mapContentType, len(mapBody))
}

func TestStatusEndpointAvailabilityAndLatency(t *testing.T) {
	client := &http.Client{Timeout: 3 * time.Second}
	baseURL := controlCenterURL()
	waitForHealth(t, client, baseURL)

	const totalRequests = 200
	latencies := make([]time.Duration, 0, totalRequests)
	statusCounts := map[int]int{}

	for i := 0; i < totalRequests; i++ {
		start := time.Now()
		resp, err := client.Get(baseURL + "/status")
		elapsed := time.Since(start)
		latencies = append(latencies, elapsed)

		if err != nil {
			statusCounts[0]++
			t.Fatalf("request %d failed after %s: %v", i+1, elapsed, err)
		}

		_, readErr := io.Copy(io.Discard, resp.Body)
		closeErr := resp.Body.Close()
		statusCounts[resp.StatusCode]++

		if readErr != nil {
			t.Fatalf("request %d body read failed: %v", i+1, readErr)
		}
		if closeErr != nil {
			t.Fatalf("request %d body close failed: %v", i+1, closeErr)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("request %d returned HTTP %d", i+1, resp.StatusCode)
		}
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	minimum := latencies[0]
	median := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	maximum := latencies[len(latencies)-1]
	average := averageDuration(latencies)

	t.Logf("non-functional result: requests=%d status_counts=%v", totalRequests, statusCounts)
	t.Logf("latency statistics: min_ms=%.2f avg_ms=%.2f p50_ms=%.2f p95_ms=%.2f max_ms=%.2f", millis(minimum), millis(average), millis(median), millis(p95), millis(maximum))

	if p95 > 200*time.Millisecond {
		t.Fatalf("p95 latency %.2fms exceeds threshold 200.00ms", millis(p95))
	}
}

func controlCenterURL() string {
	if value := os.Getenv("CONTROL_CENTER_URL"); value != "" {
		return value
	}
	return "http://control-center:8080"
}

func waitForHealth(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
		if err != nil {
			t.Fatalf("create health request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if ctx.Err() != nil {
			t.Fatalf("control center did not become healthy at %s within 30s", baseURL)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, expectedStatus int) {
	t.Helper()

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("create request %s %s: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, url, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body for %s %s: %v", method, url, err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("%s %s returned HTTP %d, want %d, body=%q", method, url, resp.StatusCode, expectedStatus, string(responseBody))
	}
}

func doGET(t *testing.T, client *http.Client, url string, expectedStatus int) ([]byte, string) {
	t.Helper()

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body for GET %s: %v", url, err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("GET %s returned HTTP %d, want %d, body=%q", url, resp.StatusCode, expectedStatus, string(body))
	}
	return body, resp.Header.Get("Content-Type")
}

func getStatus(t *testing.T, client *http.Client, baseURL string) systemStatus {
	t.Helper()

	body, _ := doGET(t, client, baseURL+"/status", http.StatusOK)
	var status systemStatus
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("decode /status JSON: %v; body=%q", err, string(body))
	}
	return status
}

func assertContains(t *testing.T, label string, ok bool) {
	t.Helper()
	if !ok {
		t.Fatalf("status does not contain expected %s", label)
	}
}

func percentile(sorted []time.Duration, p int) time.Duration {
	index := (len(sorted)*p + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}

func averageDuration(values []time.Duration) time.Duration {
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return total / time.Duration(len(values))
}

func millis(value time.Duration) float64 {
	return float64(value.Microseconds()) / 1000
}
