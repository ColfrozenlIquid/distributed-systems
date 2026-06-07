package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type ActorConfig struct {
	Id                      string
	ActorType               string
	IntervalMs              time.Duration
	XPosition               uint32
	YPosition               uint32
	BaseStationEndpoint     string
	WorldSimulationEndpoint string
}

func LoadConfig() (*ActorConfig, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("ACTOR_INTERVAL_MS", 1000)
	v.SetDefault("ACTOR_POSITION_X", 0)
	v.SetDefault("ACTOR_POSITION_Y", 0)

	for _, key := range []string{"ACTOR_ID", "ACTOR_TYPE", "ACTOR_BASE_STATION_HOST", "ACTOR_WORLD_SIMULATION_HOST"} {
		if v.GetString(key) == "" {
			return nil, fmt.Errorf("missing required env var: %s", key)
		}
	}

	return &ActorConfig{
		Id:                      v.GetString("ACTOR_ID"),
		ActorType:               v.GetString("ACTOR_TYPE"),
		IntervalMs:              time.Duration(v.GetInt("ACTOR_INTERVAL_MS")) * time.Millisecond,
		XPosition:               v.GetUint32("ACTOR_POSITION_X"),
		YPosition:               v.GetUint32("ACTOR_POSITION_Y"),
		BaseStationEndpoint:     v.GetString("ACTOR_BASE_STATION_HOST"),
		WorldSimulationEndpoint: v.GetString("ACTOR_WORLD_SIMULATION_HOST"),
	}, nil
}
