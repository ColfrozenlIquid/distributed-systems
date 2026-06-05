package sensor

import (
	"context"
	"fmt"
	"math/rand/v2"
	pb "sensors/gen"
	"sensors/internal/config"
	"shared"
	"time"
)

var _ Sensor = (*DroneSensor)(nil)

type DroneSensor struct {
	cfg        config.SensorConfig
	simulation pb.WorldServiceClient
	sensor     droneSensor
}

func (v *DroneSensor) Update() error {
	if rand.Float32() < 0.4 {
		return nil
	}

	newX := v.sensor.position.X + (rand.IntN(3) - 1)
	newY := v.sensor.position.Y + (rand.IntN(3) - 1)

	req := &pb.MoveToRequest{
		XPos:        int32(newX),
		YPos:        int32(newY),
		MovableType: pb.MovableType_MOVABLE_TYPE_AIR,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := v.simulation.MoveTo(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to move to tile: %w", err)
	}

	if resp.Allowed {
		v.sensor.position.X = newX
		v.sensor.position.Y = newY

	}

	return nil
}

func (v *DroneSensor) Position() Position {
	return v.sensor.position
}

type droneSensor struct {
	id       string
	position Position
	interval time.Duration
	reading  float32
}

func NewDroneSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *DroneSensor {
	return &DroneSensor{
		cfg:        cfg,
		simulation: simulation,
		sensor: droneSensor{
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

func (v *DroneSensor) ID() string {
	return v.sensor.id
}

func (v *DroneSensor) Interval() time.Duration {
	return v.sensor.interval
}

func (v *DroneSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       int32(v.sensor.position.X),
		YPos:       int32(v.sensor.position.Y),
		SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
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
			SensorType: shared.PersonConfidenceSensor,
		},
	}, nil
}

func (v *DroneSensor) Type() string {
	return "drone"
}
