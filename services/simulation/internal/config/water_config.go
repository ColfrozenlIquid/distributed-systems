package config

import "github.com/spf13/viper"

const MAX_WATER_LEVEL_CM float32 = 100.0
const MIN_WATER_LEVEL_CM float32 = 0.0
const STARTING_WATER_LEVEL_CM float32 = 50.0
const WATER_LEVEL_SMOOTHING float32 = 0.001

type WaterSimulatableConfig struct {
	MaxWaterLevelCM      float32
	MinWaterLevelCM      float32
	StartingWaterLevelCM float32
	WaterLevelSmoothing  float32
}

func loadWaterConfig(v *viper.Viper) *WaterSimulatableConfig {
	v.SetDefault("WATER_MAX_LEVEL_CM", MAX_WATER_LEVEL_CM)
	v.SetDefault("WATER_MIN_LEVEL_CM", MIN_WATER_LEVEL_CM)
	v.SetDefault("WATER_STARTING_LEVEL_CM", STARTING_WATER_LEVEL_CM)
	v.SetDefault("WATER_LEVEL_SMOOTHING", WATER_LEVEL_SMOOTHING)

	return &WaterSimulatableConfig{
		MaxWaterLevelCM:      float32(v.GetFloat64("WATER_MAX_LEVEL_CM")),
		MinWaterLevelCM:      float32(v.GetFloat64("WATER_MIN_LEVEL_CM")),
		StartingWaterLevelCM: float32(v.GetFloat64("WATER_STARTING_LEVEL_CM")),
		WaterLevelSmoothing:  float32(v.GetFloat64("WATER_LEVEL_SMOOTHING")),
	}
}
