package types

import (
	"context"
	"fmt"
	"sensors/config"
	pb "sensors/gen/proto"
	"shared"
	"time"
)

var _ Sensor = (*VibrationSensor)(nil)

type VibrationSensor struct {
	cfg        config.SensorConfig
	simulation pb.WorldServiceClient
	sensor     vibrationSensor
}

type vibrationSensor struct {
	id       string
	xpos     int
	ypos     int
	interval time.Duration
	reading  float32
}

func NewVibrationSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *VibrationSensor {
	return &VibrationSensor{
		cfg:        cfg,
		simulation: simulation,
		sensor: vibrationSensor{
			id:       cfg.Id,
			xpos:     int(cfg.XPosition),
			ypos:     int(cfg.YPosition),
			interval: cfg.IntervalMs,
			reading:  0,
		},
	}
}

func (v *VibrationSensor) ID() string {
	return v.sensor.id
}

func (v *VibrationSensor) Interval() time.Duration {
	return v.sensor.interval
}

func (v *VibrationSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       int32(v.sensor.xpos),
		YPos:       int32(v.sensor.ypos),
		SensorType: pb.SensorType_SENSOR_TYPE_VIBRATION,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := v.simulation.TileReading(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get tile reading: %w", err)
	}

	return &shared.SensorReading{
		Timestamp: time.Now(),
		Value:     resp.SensorReading,
		Sensor: shared.SensorInfo{
			Id:         v.sensor.id,
			Position:   shared.Position{X: v.sensor.xpos, Y: v.sensor.ypos},
			SensorType: shared.VibrationSensor,
		},
	}, nil
}

func (v *VibrationSensor) Type() string {
	return "vibration"
}
