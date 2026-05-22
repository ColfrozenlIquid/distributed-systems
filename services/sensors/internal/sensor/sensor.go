package sensor

import (
	"fmt"
	"shared"
	"time"

	pb "sensors/gen"
	"sensors/internal/config"
)

type Position struct {
	X int
	Y int
}

type Sensor interface {
	ID() string
	Type() string
	Read() (*shared.SensorReading, error)
	Interval() time.Duration
	Position() Position
	Update() error
}

func NewSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) (Sensor, error) {
	switch cfg.SensorType {
	case "vibration":
		return NewVibrationSensor(cfg, simulation), nil
	case "water_level":
		return NewWaterLevelSensor(cfg, simulation), nil
	case "swimmer_beacon":
		return NewSwimmerBeaconSensor(cfg, simulation), nil
	default:
		return nil, fmt.Errorf("unknown sensor type: %s", cfg.SensorType)
	}
}
