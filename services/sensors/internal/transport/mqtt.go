package transport

import (
	"fmt"
	"shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Handle MQTT transport layer
type MQTTClient struct {
	MQTT mqtt.Client
}

func NewMQTTClient(cfg TransportConfig) (*MQTTClient, error) {
	mqtt, err := shared.ConnectToMqttBroker(cfg.MQTTConfig, cfg.ClientId, true, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to mqtt broker: %w", err)
	}

	return &MQTTClient{
		MQTT: mqtt,
	}, nil
}
