package aufgabe2tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

type sensorReading struct {
	Timestamp time.Time  `json:"timestamp"`
	Sensor    sensorInfo `json:"sensor"`
	Value     float64    `json:"value"`
}

type sensorIncident struct {
	Id        string     `json:"id"`
	Timestamp time.Time  `json:"timestamp"`
	Sensor    sensorInfo `json:"sensor"`
	Incident  int        `json:"incident"`
}

type missionUpdate struct {
	MissionID string    `json:"mission_id"`
	ActorID   string    `json:"actor_id"`
	ActorType string    `json:"actor_type"`
	State     string    `json:"state"`
	Position  position  `json:"position"`
	UpdatedAt time.Time `json:"updated_at"`
}

type systemStatus struct {
	Sensors   []sensorInfo     `json:"sensors"`
	Actors    []actorInfo      `json:"actors"`
	Readings  []sensorReading  `json:"readings"`
	Incidents []sensorIncident `json:"incidents"`
	Missions  []missionUpdate  `json:"missions"`
}

func TestAssignWaterIncidentMissionViaGRPC(t *testing.T) {
	client := &http.Client{Timeout: 15 * time.Second}
	baseURL := controlCenterURL()
	waitForHealth(t, client, baseURL)

	sensor := sensorInfo{
		ID:         fmt.Sprintf("it2-water-%d", time.Now().UnixNano()),
		Position:   position{X: 1, Y: 1},
		SensorType: "water_level",
	}

	doJSON(t, client, http.MethodPost, baseURL+"/register/sensor", sensorRegistration{Sensor: sensor}, http.StatusNoContent)
	doJSON(t, client, http.MethodPost, baseURL+"/reading", sensorReading{
		Timestamp: time.Now().UTC(),
		Sensor:    sensor,
		Value:     95,
	}, http.StatusNoContent)

	statusBefore := getStatus(t, client, baseURL)
	incident, ok := findIncidentForSensor(statusBefore.Incidents, sensor.ID, 1)
	if !ok {
		t.Fatalf("expected water level incident for sensor %q in /status", sensor.ID)
	}

	incidentID := incident.Id
	assignMission(t, client, baseURL, incidentID, 20*time.Second)
}

func TestMissionAssignmentStreamLatencyAndReliability(t *testing.T) {
	client := &http.Client{Timeout: 20 * time.Second}
	baseURL := controlCenterURL()
	waitForHealth(t, client, baseURL)

	const attempts = 3
	firstUpdateLatencies := make([]time.Duration, 0, attempts)
	completionLatencies := make([]time.Duration, 0, attempts)

	for i := 0; i < attempts; i++ {
		sensor := sensorInfo{
			ID:         fmt.Sprintf("it2-water-latency-%d-%d", i+1, time.Now().UnixNano()),
			Position:   position{X: 1, Y: 1},
			SensorType: "water_level",
		}

		doJSON(t, client, http.MethodPost, baseURL+"/register/sensor", sensorRegistration{Sensor: sensor}, http.StatusNoContent)
		doJSON(t, client, http.MethodPost, baseURL+"/reading", sensorReading{
			Timestamp: time.Now().UTC(),
			Sensor:    sensor,
			Value:     95,
		}, http.StatusNoContent)

		status := getStatus(t, client, baseURL)
		incident, ok := findIncidentForSensor(status.Incidents, sensor.ID, 1)
		if !ok {
			t.Fatalf("attempt %d: expected water level incident for sensor %q", i+1, sensor.ID)
		}

		incidentID := fmt.Sprintf("%s-%d", incident.Sensor.ID, incident.Incident)
		startedAt := time.Now()
		updates := assignMissionAndReadStream(t, client, baseURL, incidentID, 20*time.Second)
		if len(updates) == 0 {
			t.Fatalf("attempt %d: expected mission updates", i+1)
		}

		firstUpdateLatencies = append(firstUpdateLatencies, updates[0].UpdatedAt.Sub(startedAt))
		completionLatencies = append(completionLatencies, updates[len(updates)-1].UpdatedAt.Sub(startedAt))

		if updates[len(updates)-1].State != "done" {
			t.Fatalf("attempt %d: expected final state done, got %q", i+1, updates[len(updates)-1].State)
		}
	}

	sort.Slice(firstUpdateLatencies, func(i, j int) bool { return firstUpdateLatencies[i] < firstUpdateLatencies[j] })
	sort.Slice(completionLatencies, func(i, j int) bool { return completionLatencies[i] < completionLatencies[j] })

	p95First := percentile(firstUpdateLatencies, 95)
	p95Completion := percentile(completionLatencies, 95)

	t.Logf("non-functional result: attempts=%d", attempts)
	t.Logf("stream latency statistics: first_update_p95_ms=%.2f completion_p95_ms=%.2f", millis(p95First), millis(p95Completion))

	if p95First > 2*time.Second {
		t.Fatalf("first update p95 latency %.2fms exceeds threshold 2000.00ms", millis(p95First))
	}
	if p95Completion > 15*time.Second {
		t.Fatalf("completion p95 latency %.2fms exceeds threshold 15000.00ms", millis(p95Completion))
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

func doGET(t *testing.T, client *http.Client, url string, expectedStatus int) []byte {
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
	return body
}

func getStatus(t *testing.T, client *http.Client, baseURL string) systemStatus {
	t.Helper()

	body := doGET(t, client, baseURL+"/status", http.StatusOK)
	var status systemStatus
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("decode /status JSON: %v; body=%q", err, string(body))
	}
	return status
}

func findIncidentForSensor(incidents []sensorIncident, sensorID string, incidentType int) (sensorIncident, bool) {
	for _, incident := range incidents {
		if incident.Sensor.ID == sensorID && incident.Incident == incidentType {
			return incident, true
		}
	}
	return sensorIncident{}, false
}

func assignMissionAndReadStream(t *testing.T, client *http.Client, baseURL, incidentID string, timeout time.Duration) []missionUpdate {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/incidents/"+incidentID+"/mission", nil)
	if err != nil {
		t.Fatalf("create mission assignment request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /incidents/:incidentID/mission failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /incidents/:incidentID/mission returned HTTP %d, want %d, body=%q", resp.StatusCode, http.StatusOK, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	updates := make([]missionUpdate, 0, 8)
	for {
		var update missionUpdate
		if err := decoder.Decode(&update); err != nil {
			if err == io.EOF {
				return updates
			}
			t.Fatalf("decode mission stream: %v", err)
		}
		updates = append(updates, update)
	}
}

func assignMission(t *testing.T, client *http.Client, baseURL, incidentID string, timeout time.Duration) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/incidents/"+incidentID+"/mission", nil)
	if err != nil {
		t.Fatalf("create mission assignment request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST /incidents/:incidentID/mission failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /incidents/:incidentID/mission returned HTTP %d, want %d, body=%q", resp.StatusCode, http.StatusOK, string(body))
	}
	return string(body)
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

func millis(value time.Duration) float64 {
	return float64(value.Microseconds()) / 1000
}
