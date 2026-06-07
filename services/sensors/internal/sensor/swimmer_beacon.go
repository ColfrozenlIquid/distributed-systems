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

var _ Sensor = (*SwimmerBeaconSensor)(nil)

type SwimmerBeaconSensor struct {
	BaseSensor
	simulation pb.WorldServiceClient
}

func NewSwimmerBeaconSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *SwimmerBeaconSensor {
	return &SwimmerBeaconSensor{
		BaseSensor: NewBaseSensor(cfg),
		simulation: simulation,
	}
}

func (v *SwimmerBeaconSensor) Update() error {
	if rand.Float32() < 0.4 {
		return nil
	}

	newX := v.BaseSensor.position.X + (rand.Uint32N(3) - 1)
	newY := v.BaseSensor.position.Y + (rand.Uint32N(3) - 1)

	req := &pb.MoveToRequest{
		XPos:        newX,
		YPos:        newY,
		MovableType: pb.MovableType_MOVABLE_TYPE_WATER,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := v.simulation.MoveTo(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to move to tile: %w", err)
	}

	if resp.Allowed {
		v.BaseSensor.position.X = newX
		v.BaseSensor.position.Y = newY
	}

	return nil
}

func (v *SwimmerBeaconSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       v.BaseSensor.position.X,
		YPos:       v.BaseSensor.position.Y,
		SensorType: pb.SensorType_SENSOR_TYPE_WATER_LEVEL,
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
			Id:         v.BaseSensor.id,
			Position:   shared.Position{X: v.BaseSensor.position.X, Y: v.BaseSensor.position.Y},
			SensorType: shared.WaterLevelSensor,
		},
	}, nil
}
