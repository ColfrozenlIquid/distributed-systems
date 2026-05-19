package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	World     *WorldConfig
	Water     *WaterSimulatableConfig
	Vibration *VibrationSimulatableConfig
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	cfg := &Config{}

	cfg.World = loadWorldConfig(v)
	cfg.Water = loadWaterConfig(v)
	cfg.Vibration = loadVibrationConfig(v)

	return cfg, nil
}
