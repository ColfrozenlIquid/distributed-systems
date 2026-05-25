package config

import "github.com/spf13/viper"

const WORLD_UPDATE_INTERVAL_MS float32 = 2000.0

type WorldConfig struct {
	UpdateIntervalMS float32
}

func loadWorldConfig(v *viper.Viper) *WorldConfig {
	v.SetDefault("WORLD_UPDATE_INTERVAL_MS", WORLD_UPDATE_INTERVAL_MS)

	return &WorldConfig{
		UpdateIntervalMS: float32(v.GetFloat64("WORLD_UPDATE_INTERVAL_MS")),
	}
}
