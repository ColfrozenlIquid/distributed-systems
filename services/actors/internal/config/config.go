package config

import (
	"fmt"
	"shared"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type ActorConfig struct {
	Id                      string
	ActorType               string
	IntervalMs              time.Duration
	XPosition               uint32
	YPosition               uint32
	FuelCapacity            float64
	FuelDrainRate           float64
	ChargingStations        []shared.Position
	BaseStationEndpoint     string
	WorldSimulationEndpoint string
	MQTTConfig              *shared.MqttConfig
}

func LoadConfig() (*ActorConfig, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("ACTOR_INTERVAL_MS", 1000)
	v.SetDefault("ACTOR_POSITION_X", 0)
	v.SetDefault("ACTOR_POSITION_Y", 0)
	v.SetDefault("ACTOR_FUEL_CAPACITY", 100.0)
	v.SetDefault("ACTOR_FUEL_DRAIN_RATE", 4.0)
	v.SetDefault("CHARGING_STATIONS", "8,10")

	for _, key := range []string{"ACTOR_ID", "ACTOR_TYPE", "CONTROL_CENTER_URL", "WORLD_SIMULATION_HOST_AND_PORT"} {
		if v.GetString(key) == "" {
			return nil, fmt.Errorf("missing required env var: %s", key)
		}
	}

	mqttConfig, err := shared.LoadMqttConfig(v)
	if err != nil {
		return nil, fmt.Errorf("mqtt config missing: %w", err)
	}

	chargingStations, err := parsePositions(v.GetString("CHARGING_STATIONS"))
	if err != nil {
		return nil, fmt.Errorf("parse charging station positions: %w", err)
	}

	return &ActorConfig{
		Id:                      v.GetString("ACTOR_ID"),
		ActorType:               v.GetString("ACTOR_TYPE"),
		IntervalMs:              time.Duration(v.GetInt("ACTOR_INTERVAL_MS")) * time.Millisecond,
		XPosition:               v.GetUint32("ACTOR_POSITION_X"),
		YPosition:               v.GetUint32("ACTOR_POSITION_Y"),
		FuelCapacity:            v.GetFloat64("ACTOR_FUEL_CAPACITY"),
		FuelDrainRate:           v.GetFloat64("ACTOR_FUEL_DRAIN_RATE"),
		ChargingStations:        chargingStations,
		BaseStationEndpoint:     v.GetString("CONTROL_CENTER_URL"),
		WorldSimulationEndpoint: v.GetString("WORLD_SIMULATION_HOST_AND_PORT"),
		MQTTConfig:              mqttConfig,
	}, nil
}

func parsePositions(raw string) ([]shared.Position, error) {
	if raw == "" {
		return nil, nil
	}

	var positions []shared.Position
	for _, pair := range strings.Split(raw, ";") {
		parts := strings.Split(pair, ",")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid position pair: %q", pair)
		}

		x, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid argument x in %q: %w", pair, err)
		}

		y, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid argument y in %q: %w", pair, err)
		}
		positions = append(positions, shared.Position{
			X: uint32(x),
			Y: uint32(y),
		})
	}
	return positions, nil
}
