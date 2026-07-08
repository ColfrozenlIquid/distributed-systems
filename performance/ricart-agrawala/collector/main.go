package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type CSEvent struct {
	NodeID string    `json:"node_id"`
	Type   string    `json:"type"`
	TS     time.Time `json:"ts"`
	K      int       `json:"k"` // <-- ADD
}

func main() {
	// IMPORTANT: point this at the real broker (mqtt-broker:1883),
	// not at any per-node toxiproxy proxy. The collector is an
	// external observer and shouldn't have its own view of events
	// artificially delayed by a scenario meant to affect the nodes
	// under test.
	brokerURL := envOr("MQTT_BROKER_URL", "tcp://mqtt-broker:1883")
	outPath := envOr("METRICS_CSV_PATH", "/data/sync_delay.csv")

	f, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("create csv: %v", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"release_node", "release_ts", "enter_node", "enter_ts", "sync_delay_ms", "recv_delay_ms", "k"})

	var mu sync.Mutex
	var lastRelease *CSEvent
	var lastReleaseRecvTS time.Time

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("metrics-collector").
		SetCleanSession(true).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("collector connect failed: %v", token.Error())
	}

	handler := func(c mqtt.Client, msg mqtt.Message) {
		recvTS := time.Now()
		var evt CSEvent
		if err := json.Unmarshal(msg.Payload(), &evt); err != nil {
			log.Printf("bad event payload: %v", err)
			return
		}

		mu.Lock()
		defer mu.Unlock()

		switch evt.Type {
		case "release":
			lastRelease = &evt
			lastReleaseRecvTS = recvTS
		case "enter":
			if lastRelease == nil {
				return
			}
			syncDelay := evt.TS.Sub(lastRelease.TS)
			recvDelay := recvTS.Sub(lastReleaseRecvTS)
			log.Printf("HANDOFF: %s -> %s | sync_delay=%v recv_delay=%v k=%d",
				lastRelease.NodeID, evt.NodeID, syncDelay, recvDelay, lastRelease.K)
			_ = w.Write([]string{
				lastRelease.NodeID,
				lastRelease.TS.Format(time.RFC3339Nano),
				evt.NodeID,
				evt.TS.Format(time.RFC3339Nano),
				fmt.Sprintf("%.3f", float64(syncDelay.Microseconds())/1000.0),
				fmt.Sprintf("%.3f", float64(recvDelay.Microseconds())/1000.0),
				fmt.Sprintf("%d", lastRelease.K), // <-- new column
			})
			w.Flush()
			lastRelease = nil
		}
	}

	if token := client.Subscribe("cs/events", 1, handler); token.Wait() && token.Error() != nil {
		log.Fatalf("subscribe failed: %v", token.Error())
	}

	log.Println("collector running, writing to", outPath)
	select {}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
