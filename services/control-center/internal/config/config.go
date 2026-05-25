package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type ControlCenterConfig struct {
	WorldSimulationHost string
}

func LoadConfig() (*ControlCenterConfig, error) {
	v := viper.New()
	v.AutomaticEnv()

	worldSimulationHostKey := "CONTROL_CENTER_WORLD_SIMULATION_HOST"
	var worldSimulationHost string
	if worldSimulationHost = v.GetString(worldSimulationHostKey); worldSimulationHost == "" {
		return nil, fmt.Errorf("missing required env var: %s", worldSimulationHostKey)
	}

	return &ControlCenterConfig{
		WorldSimulationHost: worldSimulationHost,
	}, nil
}
