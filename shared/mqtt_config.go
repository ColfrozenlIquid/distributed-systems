package shared

import (
	"fmt"

	"github.com/spf13/viper"
)

type MqttBrokerUrlProvider interface {
	GetMqttBrokerUrl() string
}

type MqttConfig struct {
	BrokerHost string
	BrokerPort uint16
}

func (c MqttConfig) GetMqttBrokerUrl() string {
	return fmt.Sprintf("tcp://%s:%d", c.BrokerHost, c.BrokerPort)
}

func LoadMqttConfig(v *viper.Viper) (*MqttConfig, error) {
	return &MqttConfig{
		BrokerHost: getStringValue(v, "MQTT_BROKER_HOST"),
		BrokerPort: getUint16Value(v, "MQTT_BROKER_PORT"),
	}, nil
}

func getStringValue(v *viper.Viper, key string) string {
	value := v.GetString(key)
	if value == "" {
		panic(fmt.Errorf("missing required env var: %s", key))
	}
	return value
}

func getUint16Value(v *viper.Viper, key string) uint16 {
	value := v.GetUint16(key)
	if value == 0 {
		panic(fmt.Errorf("missing required env var: %s", key))
	}
	return value
}
