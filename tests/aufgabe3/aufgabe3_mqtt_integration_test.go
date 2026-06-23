package aufgabe3tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	topicAllReadings    = "tiles/+/+/readings/#"
	topicAllStatuses    = "actors/+/status"
	topicAllTelemetries = "actors/+/telemetry"
)

type position struct {
	X uint32 `json:"x"`
	Y uint32 `json:"y"`
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

type actorStatus struct {
	ActorID   string    `json:"actor_id"`
	State     string    `json:"state"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type actorTelemetry struct {
	ActorID   string    `json:"actor_id"`
	Position  position  `json:"position"`
	Timestamp time.Time `json:"timestamp"`
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
	ID        string     `json:"id"`
	Timestamp time.Time  `json:"timestamp"`
	Sensor    sensorInfo `json:"sensor"`
	Incident  int        `json:"incident"`
}

type systemActor struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Position  position  `json:"position"`
	State     string    `json:"state"`
	Reason    string    `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type systemStatus struct {
	Sensors   []sensorInfo     `json:"sensors"`
	Actors    []systemActor    `json:"actors"`
	Readings  []sensorReading  `json:"readings"`
	Incidents []sensorIncident `json:"incidents"`
}

type capturedMessage struct {
	Topic   string
	Payload []byte
}

func TestMQTTSensorAndActorMessagesReachControlCenter(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}
	baseURL := controlCenterURL()
	waitForHealth(t, client, baseURL)

	brokerURL := mqttBrokerURL()
	subscriber := connectMQTT(t, brokerURL, fmt.Sprintf("it3-sub-%d", time.Now().UnixNano()), true)
	defer subscriber.Disconnect(250)

	messages := make(chan capturedMessage, 8)
	subscribe(t, subscriber, map[string]byte{
		topicAllReadings:    0,
		topicAllStatuses:    1,
		topicAllTelemetries: 0,
	}, func(_ mqtt.Client, message mqtt.Message) {
		payload := append([]byte(nil), message.Payload()...)
		messages <- capturedMessage{Topic: message.Topic(), Payload: payload}
	})

	publisher := connectMQTT(t, brokerURL, fmt.Sprintf("it3-pub-%d", time.Now().UnixNano()), true)
	defer publisher.Disconnect(250)

	now := time.Now().UTC()
	sensor := sensorInfo{
		ID:         fmt.Sprintf("it3-water-%d", now.UnixNano()),
		Position:   position{X: 6, Y: 6},
		SensorType: "water_level",
	}
	actor := actorInfo{
		ID:        fmt.Sprintf("it3-actor-%d", now.UnixNano()),
		Position:  position{X: 10, Y: 10},
		ActorType: "water_vehicle",
	}
	doJSON(t, client, http.MethodPost, baseURL+"/register/sensor", sensorRegistration{Sensor: sensor}, http.StatusNoContent)
	doJSON(t, client, http.MethodPost, baseURL+"/register/actor", actorRegistration{Actor: actor}, http.StatusNoContent)

	readingTopic := fmt.Sprintf("tiles/%d/%d/readings/%s", sensor.Position.X, sensor.Position.Y, sensor.SensorType)
	statusTopic := fmt.Sprintf("actors/%s/status", actor.ID)
	telemetryTopic := fmt.Sprintf("actors/%s/telemetry", actor.ID)

	reading := sensorReading{Timestamp: now, Sensor: sensor, Value: 95}
	status := actorStatus{ActorID: actor.ID, State: "on_a_mission", Timestamp: now.Add(time.Second)}
	telemetry := actorTelemetry{ActorID: actor.ID, Position: position{X: 6, Y: 6}, Timestamp: now.Add(2 * time.Second)}

	publishJSON(t, publisher, readingTopic, 0, false, reading)
	publishJSON(t, publisher, statusTopic, 1, true, status)
	publishJSON(t, publisher, telemetryTopic, 0, false, telemetry)

	expectMQTTMessages(t, messages, map[string]string{
		readingTopic:   sensor.ID,
		statusTopic:    actor.ID,
		telemetryTopic: actor.ID,
	})

	finalStatus := waitForStatus(t, client, baseURL, 10*time.Second, func(status systemStatus) bool {
		hasReading := slices.ContainsFunc(status.Readings, func(r sensorReading) bool {
			return r.Sensor.ID == sensor.ID && r.Value == 95
		})
		hasIncident := slices.ContainsFunc(status.Incidents, func(i sensorIncident) bool {
			return i.Sensor.ID == sensor.ID && i.Incident == 1
		})
		hasActorUpdate := slices.ContainsFunc(status.Actors, func(a systemActor) bool {
			return a.ID == actor.ID && a.State == "on_a_mission" && a.Position == telemetry.Position
		})
		return hasReading && hasIncident && hasActorUpdate
	})

	t.Logf("functional result: mqtt_broker=%s reading_topic=%s status_topic=%s telemetry_topic=%s", brokerURL, readingTopic, statusTopic, telemetryTopic)
	t.Logf("status summary: sensors=%d actors=%d readings=%d incidents=%d", len(finalStatus.Sensors), len(finalStatus.Actors), len(finalStatus.Readings), len(finalStatus.Incidents))
	t.Logf("verified mqtt control-center projection: sensor_id=%s value=%.1f incident=WaterLevelAlert actor_id=%s state=%s position=(%d,%d)", sensor.ID, reading.Value, actor.ID, status.State, telemetry.Position.X, telemetry.Position.Y)
}

func connectMQTT(t *testing.T, brokerURL, clientID string, cleanSession bool) mqtt.Client {
	t.Helper()

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID).
		SetCleanSession(cleanSession).
		SetConnectTimeout(5 * time.Second).
		SetOrderMatters(false)

	mqttClient := mqtt.NewClient(opts)
	token := mqttClient.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		t.Fatalf("connect MQTT client %q to %s timed out", clientID, brokerURL)
	}
	if err := token.Error(); err != nil {
		t.Fatalf("connect MQTT client %q to %s: %v", clientID, brokerURL, err)
	}
	return mqttClient
}

func subscribe(t *testing.T, client mqtt.Client, filters map[string]byte, handler mqtt.MessageHandler) {
	t.Helper()

	var once sync.Once
	ready := make(chan struct{})
	token := client.SubscribeMultiple(filters, func(c mqtt.Client, m mqtt.Message) {
		once.Do(func() { close(ready) })
		handler(c, m)
	})
	if !token.WaitTimeout(5 * time.Second) {
		t.Fatalf("subscribe to MQTT filters %v timed out", filters)
	}
	if err := token.Error(); err != nil {
		t.Fatalf("subscribe to MQTT filters %v: %v", filters, err)
	}
	_ = ready
}

func publishJSON(t *testing.T, client mqtt.Client, topic string, qos byte, retained bool, payload any) {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal MQTT payload for %s: %v", topic, err)
	}
	token := client.Publish(topic, qos, retained, body)
	if !token.WaitTimeout(5 * time.Second) {
		t.Fatalf("publish MQTT message to %s timed out", topic)
	}
	if err := token.Error(); err != nil {
		t.Fatalf("publish MQTT message to %s: %v", topic, err)
	}
}

func expectMQTTMessages(t *testing.T, messages <-chan capturedMessage, expected map[string]string) {
	t.Helper()

	timeout := time.After(5 * time.Second)
	for {
		if len(expected) == 0 {
			return
		}
		select {
		case message := <-messages:
			payloadContains, ok := expected[message.Topic]
			if ok && bytes.Contains(message.Payload, []byte(payloadContains)) {
				delete(expected, message.Topic)
			}
		case <-timeout:
			t.Fatalf("did not receive expected MQTT messages: %v", expected)
		}
	}
}

func controlCenterURL() string {
	if value := os.Getenv("CONTROL_CENTER_URL"); value != "" {
		return value
	}
	return "http://control-center:8080"
}

func mqttBrokerURL() string {
	if value := os.Getenv("MQTT_BROKER_URL"); value != "" {
		return value
	}
	host := os.Getenv("MQTT_BROKER_HOST")
	if host == "" {
		host = "mqtt-broker"
	}
	port := os.Getenv("MQTT_BROKER_PORT")
	if port == "" {
		port = "1883"
	}
	return fmt.Sprintf("tcp://%s:%s", host, port)
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

func getStatus(t *testing.T, client *http.Client, baseURL string) systemStatus {
	t.Helper()

	resp, err := client.Get(baseURL + "/status")
	if err != nil {
		t.Fatalf("GET /status failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /status body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /status returned HTTP %d, want %d, body=%q", resp.StatusCode, http.StatusOK, string(body))
	}

	var status systemStatus
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("decode /status JSON: %v; body=%q", err, string(body))
	}
	return status
}

func waitForStatus(t *testing.T, client *http.Client, baseURL string, timeout time.Duration, predicate func(systemStatus) bool) systemStatus {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var last systemStatus
	for time.Now().Before(deadline) {
		last = getStatus(t, client, baseURL)
		if predicate(last) {
			return last
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("status did not contain expected MQTT projection within %s; last summary: sensors=%d actors=%d readings=%d incidents=%d", timeout, len(last.Sensors), len(last.Actors), len(last.Readings), len(last.Incidents))
	return systemStatus{}
}
