package types

import (
	"context"
	"fmt"
	"sensors/config"
	"time"

	pb "sensors/gen/proto"
)

var _ Sensor = (*WaterLevelSensor)(nil)

type WaterLevelSensor struct {
	cfg        config.SensorConfig
	simulation pb.WorldServiceClient
	sensor     waterLevelSensor
}

type waterLevelSensor struct {
	id       string
	xpos     int
	ypos     int
	interval time.Duration
	reading  float32
}

func NewWaterLevelSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *WaterLevelSensor {
	return &WaterLevelSensor{
		cfg:        cfg,
		simulation: simulation,
		sensor: waterLevelSensor{
			id:       cfg.Id,
			xpos:     int(cfg.XPosition),
			ypos:     int(cfg.YPosition),
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

func (w *WaterLevelSensor) Read() (*Reading, error) {
	req := &pb.TileReadingRequest{
		XPos:       int32(w.sensor.xpos),
		YPos:       int32(w.sensor.ypos),
		SensorType: pb.SensorType_SENSOR_TYPE_WATER_LEVEL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := w.simulation.TileReading(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tile reading: %w", err)
	}

	return &Reading{
		timstamp: time.Now(),
		value: map[string]any{
			"reading": resp.SensorReading,
		},
	}, nil
}

func (w *WaterLevelSensor) Type() string {
	return "water_level"
}
