package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"shared"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

// Each container runs exactly ONE node, identified by NODE_ID.
// This matches the "20 nodes in 20 containers" topology and lets
// each node be independently network-partitioned via its own
// toxiproxy proxy (see proxy-setup/).

func main() {
	ctx := context.Background()
	resourceID := envOr("RESOURCE_ID", "charging-station")

	id := os.Getenv("NODE_ID")
	if id == "" {
		log.Fatal("NODE_ID env var is required (e.g. node-0)")
	}

	runNode(ctx, id, resourceID)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type CSEvent struct {
	NodeID string    `json:"node_id"`
	Type   string    `json:"type"` // "enter" | "release"
	TS     time.Time `json:"ts"`
	K      int       `json:"k"` // <-- ADD: only meaningful for "release"
}

func publishEvent(client mqtt.Client, nodeID, evtType string, k int) { // <-- add k param
	evt := CSEvent{NodeID: nodeID, Type: evtType, TS: time.Now(), K: k}
	payload, _ := json.Marshal(evt)
	token := client.Publish("cs/events", 1, false, payload)
	token.Wait()
}

func runNode(ctx context.Context, id string, resourceId string) {
	client, err := ConnectMQTT(id)
	if err != nil {
		log.Fatalf("%s: mqtt connect failed: %v", id, err)
	}
	streams := shared.NewRicartStreams()

	requestAgain := make(chan struct{}, 1) // buffered so we never block the callback

	var node *shared.RicartNode
	node = shared.NewRicartNode(id, shared.RescueVehicleActor, resourceId, client, func() {
		publishEvent(client, id, "enter", 0) // <-- k unused on enter, pass 0
		holdTime := time.Duration(500) * time.Millisecond
		log.Printf("%s: ENTERED CRITICAL section, holding for %v", id, holdTime)
		time.Sleep(holdTime)
		log.Printf("%s: releasing critical section", id)

		k, err := node.ReleaseCS() // <-- capture k
		if err != nil {
			log.Printf("%s: release error: %v", id, err)
		}
		log.Printf("%s: RELEASING with k=%d waiters", id, k)
		publishEvent(client, id, "release", k) // <-- pass k through

		requestAgain <- struct{}{}
	})

	log.Printf("%s: about to start, IsConnected=%v", id, client.IsConnected())
	if err := node.Start(ctx, streams); err != nil {
		log.Fatalf("%s: start failed: %v", id, err)
	}

	time.Sleep(1 * time.Second)

	// small jitter to avoid a perfectly synchronized thundering herd,
	// but nowhere near seconds — keeps contention "hot"
	jitter := func() time.Duration {
		return time.Duration(rand.Intn(50)) * time.Millisecond
	}

	if err := node.RequestCS(); err != nil {
		log.Printf("%s: request error: %v", id, err)
	}

	for {
		<-requestAgain
		time.Sleep(jitter())
		if err := node.RequestCS(); err != nil {
			log.Printf("%s: request error: %v", id, err)
		}
	}
}

func ConnectMQTT(id string) (mqtt.Client, error) {
	v := viper.New()
	v.AutomaticEnv()

	log.Printf("%s: MQTT_BROKER_HOST=%s MQTT_BROKER_PORT=%s",
		id, v.GetString("MQTT_BROKER_HOST"), v.GetString("MQTT_BROKER_PORT"))

	mqttConfig, err := shared.LoadMqttConfig(v)
	if err != nil {
		return nil, fmt.Errorf("mqtt config missing: %w", err)
	}

	lastActorStatus := shared.ActorStatus{
		ActorID:   id,
		State:     shared.ActorStateOffline,
		Reason:    "",
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(lastActorStatus)

	actorStatusParams := shared.ActorStatusParams(id, lastActorStatus)
	lwtConfig := &shared.MqttLwtConfig{
		Topic:   actorStatusParams.Topic,
		Payload: payload,
		Qos:     actorStatusParams.QoS,
		Retain:  actorStatusParams.Retained,
	}

	client, err := shared.ConnectToMqttBroker(mqttConfig, id, true, lwtConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to mqtt broker: %w", err)
	}

	return client, nil
}

func ConnectToMqttBroker(brokerUrl shared.MqttBrokerUrlProvider, clientID string, cleanSession bool, lwtConfig *shared.MqttLwtConfig) (mqtt.Client, error) {
	mqttOptions := mqtt.NewClientOptions().
		SetClientID(clientID).
		AddBroker(brokerUrl.GetMqttBrokerUrl()).
		SetCleanSession(cleanSession).
		SetKeepAlive(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetOrderMatters(false)

	if lwtConfig != nil {
		mqttOptions.SetBinaryWill(lwtConfig.Topic, lwtConfig.Payload, lwtConfig.Qos, lwtConfig.Retain)
	}

	mqttClient := mqtt.NewClient(mqttOptions)
	token := mqttClient.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("timed out connecting to MQTT broker")
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}
	return mqttClient, nil
}
