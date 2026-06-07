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
	BaseSensor
	simulation pb.WorldServiceClient
}

func NewVibrationSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *VibrationSensor {
	return &VibrationSensor{
		BaseSensor: NewBaseSensor(cfg),
		simulation: simulation,
	}
}

func (v *VibrationSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       v.BaseSensor.position.X,
		YPos:       v.BaseSensor.position.Y,
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
			Id:         v.BaseSensor.id,
			Position:   shared.Position{X: v.BaseSensor.position.X, Y: v.BaseSensor.position.Y},
			SensorType: shared.VibrationSensor,
		},
	}, nil
}

func (v *VibrationSensor) Update() error {
	return nil
}
