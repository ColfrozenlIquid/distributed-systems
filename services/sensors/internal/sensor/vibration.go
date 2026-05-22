package sensor

import (
	"context"
	"fmt"
	pb "sensors/gen"
	"sensors/internal/config"
	"shared"
	"time"
)

var _ Sensor = (*VibrationSensor)(nil)

type VibrationSensor struct {
	cfg        config.SensorConfig
	simulation pb.WorldServiceClient
	sensor     vibrationSensor
}

func (v *VibrationSensor) Update() error {
	return nil
}

func (v *VibrationSensor) Position() Position {
	return v.sensor.position
}

type vibrationSensor struct {
	id       string
	position Position
	interval time.Duration
	reading  float32
}

func NewVibrationSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *VibrationSensor {
	return &VibrationSensor{
		cfg:        cfg,
		simulation: simulation,
		sensor: vibrationSensor{
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

func (v *VibrationSensor) ID() string {
	return v.sensor.id
}

func (v *VibrationSensor) Interval() time.Duration {
	return v.sensor.interval
}

func (v *VibrationSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       int32(v.sensor.position.X),
		YPos:       int32(v.sensor.position.Y),
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
			Position:   shared.Position{X: v.sensor.position.X, Y: v.sensor.position.Y},
			SensorType: shared.VibrationSensor,
		},
	}, nil
}

func (v *VibrationSensor) Type() string {
	return "vibration"
}
