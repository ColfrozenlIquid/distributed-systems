package config

import (
	"fmt"
	"shared"

	"github.com/spf13/viper"
)

type ControlCenterConfig struct {
	WorldSimulationHostAndPort string
	Mqtt                       shared.MqttConfig
}

func LoadConfig() (*ControlCenterConfig, error) {
	v := viper.New()
	v.AutomaticEnv()

	worldSimKey := "WORLD_SIMULATION_HOST_AND_PORT"
	value := viper.GetString(worldSimKey)
	if value == "" {
		panic(fmt.Errorf("missing required env var: %s", worldSimKey))
	}

	mqttConfig, err := shared.LoadMqttConfig(v)
	if err != nil {
		return nil, err
	}

	return &ControlCenterConfig{
		WorldSimulationHostAndPort: value,
		Mqtt:                       *mqttConfig,
	}, nil
}

func (c ControlCenterConfig) GetMqttBrokerUrl() string {
	return c.Mqtt.GetMqttBrokerUrl()
}
