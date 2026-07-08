package transport

import (
	"encoding/json"
	"fmt"
	"shared"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Handle MQTT transport layer
type MQTTClient struct {
	MQTT mqtt.Client
}

func NewMQTTClient(cfg TransportConfig) (*MQTTClient, error) {
	lastActorStatus := shared.ActorStatus{
		ActorID:   cfg.ClientId,
		State:     shared.ActorStateOffline,
		Reason:    "",
		Timestamp: time.Now(),
	}
	payload, _ := json.Marshal(lastActorStatus)

	actorStatusParams := shared.ActorStatusParams(cfg.ClientId, lastActorStatus)
	lwtConfig := &shared.MqttLwtConfig{
		Topic:   actorStatusParams.Topic,
		Payload: payload,
		Qos:     actorStatusParams.QoS,
		Retain:  actorStatusParams.Retained,
	}

	mqtt, err := shared.ConnectToMqttBroker(cfg.MQTTConfig, cfg.ClientId, true, lwtConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to mqtt broker: %w", err)
	}

	return &MQTTClient{
		MQTT: mqtt,
	}, nil
}
