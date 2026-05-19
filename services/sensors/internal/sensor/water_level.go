package sensor

import (
	"context"
	"fmt"
	"sensors/internal/config"
	"shared"
	"time"

	pb "sensors/gen"
)

var _ Sensor = (*WaterLevelSensor)(nil)

type WaterLevelSensor struct {
	cfg        config.SensorConfig
	simulation pb.WorldServiceClient
	sensor     waterLevelSensor
}

func (w *WaterLevelSensor) Position() Position {
	return w.sensor.position
}

type waterLevelSensor struct {
	id       string
	position Position
	interval time.Duration
	reading  float32
}

func NewWaterLevelSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *WaterLevelSensor {
	return &WaterLevelSensor{
		cfg:        cfg,
		simulation: simulation,
		sensor: waterLevelSensor{
			id: cfg.Id,
			position: Position{
				X: int(cfg.XPosition),
				Y: int(cfg.YPosition),
			},
			interval: cfg.IntervalMs,
			reading:  0,
		},
	}
}

func (w *WaterLevelSensor) ID() string {
	return w.sensor.id
}

func (w *WaterLevelSensor) Interval() time.Duration {
	return w.sensor.interval
}

func (w *WaterLevelSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       int32(w.sensor.position.X),
		YPos:       int32(w.sensor.position.Y),
		SensorType: pb.SensorType_SENSOR_TYPE_WATER_LEVEL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := w.simulation.TileReading(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tile reading: %w", err)
	}

	return &shared.SensorReading{
		Timestamp: time.Now(),
		Value:     resp.SensorReading,
		Sensor: shared.SensorInfo{
			Id:         w.sensor.id,
			Position:   shared.Position{X: w.sensor.position.X, Y: w.sensor.position.Y},
			SensorType: shared.WaterLevelSensor,
		},
	}, nil
}

func (w *WaterLevelSensor) Type() string {
	return "water_level"
}
