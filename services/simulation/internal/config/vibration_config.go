package config

import "github.com/spf13/viper"

const MAX_VIBRATION_INDEX float32 = 10.0
const MIN_VIBRATION_INDEX float32 = 0.0
const STARTING_VIBRATION_INDEX float32 = 0.0
const VIBRATION_INDEX_SMOOTHING float32 = 0.001

type VibrationSimulatableConfig struct {
	MaxVibrationIndex       float32
	MinVibrationIndex       float32
	StartingVibrationIndex  float32
	VibrationIndexSmoothing float32
}

func loadVibrationConfig(v *viper.Viper) *VibrationSimulatableConfig {
	v.SetDefault("VIBRATION_MAX_INDEX", MAX_VIBRATION_INDEX)
	v.SetDefault("VIBRATION_MIN_INDEX", MIN_VIBRATION_INDEX)
	v.SetDefault("VIBRATION_STARTING_INDEX", STARTING_VIBRATION_INDEX)
	v.SetDefault("VIBRATION_INDEX_SMOOTHING", VIBRATION_INDEX_SMOOTHING)

	return &VibrationSimulatableConfig{
		MaxVibrationIndex:       float32(v.GetFloat64("VIBRATION_MAX_INDEX")),
		MinVibrationIndex:       float32(v.GetFloat64("VIBRATION_MIN_INDEX")),
		StartingVibrationIndex:  float32(v.GetFloat64("VIBRATION_STARTING_INDEX")),
		VibrationIndexSmoothing: float32(v.GetFloat64("VIBRATION_INDEX_SMOOTHING")),
	}
}
