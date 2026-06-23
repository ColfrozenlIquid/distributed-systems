package shared

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MqttLwtConfig struct {
	Topic   string
	Payload []byte
	Qos     byte
	Retain  bool
}

func ConnectToMqttBroker(brokerUrl MqttBrokerUrlProvider, clientID string, cleanSession bool, lwtConfig *MqttLwtConfig) (mqtt.Client, error) {
	mqttOptions := mqtt.NewClientOptions().
		SetClientID(clientID).
		AddBroker(brokerUrl.GetMqttBrokerUrl()).
		SetCleanSession(cleanSession).
		SetKeepAlive(30 * time.Second).
		SetConnectTimeout(10 * time.Second).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectRetry(true).
		SetAutoReconnect(true).
		SetMaxReconnectInterval(30 * time.Second).
		SetOrderMatters(false)

	if lwtConfig != nil {
		mqttOptions.SetBinaryWill(lwtConfig.Topic, lwtConfig.Payload, lwtConfig.Qos, lwtConfig.Retain)
	}

	mqttClient := mqtt.NewClient(mqttOptions)

	token := mqttClient.Connect()
	token.WaitTimeout(10 * time.Second)
	if token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}
	return mqttClient, nil
}
