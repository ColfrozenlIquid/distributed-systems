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
	BaseSensor
	simulation pb.WorldServiceClient
}

func NewWaterLevelSensor(cfg config.SensorConfig, simulation pb.WorldServiceClient) *WaterLevelSensor {
	return &WaterLevelSensor{
		BaseSensor: NewBaseSensor(cfg),
		simulation: simulation,
	}
}

func (w *WaterLevelSensor) Read() (*shared.SensorReading, error) {
	req := &pb.TileReadingRequest{
		XPos:       w.BaseSensor.position.X,
		YPos:       w.BaseSensor.position.Y,
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
			Id:         w.BaseSensor.id,
			Position:   shared.Position{X: w.BaseSensor.position.X, Y: w.BaseSensor.position.Y},
			SensorType: shared.WaterLevelSensor,
		},
	}, nil
}

func (w *WaterLevelSensor) Update() error {
	return nil
}
