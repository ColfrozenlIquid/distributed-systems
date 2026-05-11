package types

import (
	"fmt"
	"sensors/config"
	"time"
)

type Sensor interface {
	ID() string
	Type() string
	Read() (Reading, error)
	Interval() time.Duration
}

type Reading struct {
	timstamp time.Time
	value    float32
}

func NewSensor(cfg config.SensorConfig) (Sensor, error) {
	switch cfg.SensorType {
	case "vibration":
		return &VibrationSensor{cfg}, nil
	case "water_level":
		return &WaterLevelSensor{cfg}, nil
	default:
		return nil, fmt.Errorf("unknown sensor type: %s", cfg.SensorType)
	}
}
