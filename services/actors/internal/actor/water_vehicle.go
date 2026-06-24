package actor

import (
	"actors/internal/config"
	"actors/internal/transport"
	"context"
	"fmt"

	pb "actors/gen"

	"go.uber.org/zap"
)

var _ Actor = (*VibrationVehicle)(nil)

type WaterVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	BaseActor
	logger *zap.Logger
}

func NewWaterVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger, mqtt *transport.MQTTClient) *WaterVehicle {
	return &WaterVehicle{
		cfg:        cfg,
		simulation: simulation,
		BaseActor:  NewBaseActor(cfg, simulation, logger, pb.MovableType_MOVABLE_TYPE_WATER, mqtt),
		logger:     logger,
	}
}

func (w *WaterVehicle) ExecuteMission() {
	w.BaseActor.state = ActorStateWorking
}

func (w *WaterVehicle) CompleteTask() {
	w.performTask()
	w.BaseActor.state = ActorStateDone
}

func (w *WaterVehicle) performTask() error {
	ctx := context.Background()

	_, err := w.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
		XPos:       w.BaseActor.position.X,
		YPos:       w.BaseActor.position.Y,
		SensorType: pb.SensorType_SENSOR_TYPE_WATER_LEVEL,
		Value:      0,
		Default:    true,
	})
	if err != nil {
		return fmt.Errorf("tile update: %w", err)
	}

	return nil
}
