package config

import (
	"fmt"
	"shared"
	"time"

	"github.com/spf13/viper"
)

type SensorConfig struct {
	Id                      string
	SensorType              string
	IntervalMs              time.Duration
	XPosition               uint32
	YPosition               uint32
	BaseStationEndpoint     string
	WorldSimulationEndpoint string
	MQTTConfig              *shared.MqttConfig
}

func LoadConfig() (*SensorConfig, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("SENSOR_INTERVAL_MS", 1000)
	v.SetDefault("SENSOR_POSITION_X", 0)
	v.SetDefault("SENSOR_POSITION_Y", 0)

	for _, key := range []string{"SENSOR_ID", "SENSOR_TYPE", "CONTROL_CENTER_URL", "WORLD_SIMULATION_HOST_AND_PORT"} {
		if v.GetString(key) == "" {
			return nil, fmt.Errorf("missing required env var: %s", key)
		}
	}

	mqttConfig, err := shared.LoadMqttConfig(v)
	if err != nil {
		return nil, fmt.Errorf("mqtt config missing: %w", err)
	}

	return &SensorConfig{
		Id:                      v.GetString("SENSOR_ID"),
		SensorType:              v.GetString("SENSOR_TYPE"),
		IntervalMs:              time.Duration(v.GetInt("SENSOR_INTERVAL_MS")) * time.Millisecond,
		XPosition:               uint32(v.GetInt("SENSOR_POSITION_X")),
		YPosition:               uint32(v.GetInt("SENSOR_POSITION_Y")),
		BaseStationEndpoint:     v.GetString("CONTROL_CENTER_URL"),
		WorldSimulationEndpoint: v.GetString("WORLD_SIMULATION_HOST_AND_PORT"),
		MQTTConfig:              mqttConfig,
	}, nil
}
