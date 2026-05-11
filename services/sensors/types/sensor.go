package types

import (
	"fmt"
	"sensors/config"
	"time"

	pb "sensors/gen/proto"
)

type Sensor interface {
	ID() string
	Type() string
	Read() (*Reading, error)
	Interval() time.Duration
}

type Reading struct {
	timstamp time.Time
	value    map[string]any
}

type Incident struct {
	timestamp time.Time
	value     map[string]any
}

func NewSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) (Sensor, error) {
	switch cfg.SensorType {
	case "vibration":
		return NewVibrationSensor(cfg, simulation), nil
	case "water_level":
		return NewWaterLevelSensor(cfg, simulation), nil
	default:
		return nil, fmt.Errorf("unknown sensor type: %s", cfg.SensorType)
	}
}
