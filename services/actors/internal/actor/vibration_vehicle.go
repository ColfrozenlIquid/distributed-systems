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

type VibrationVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	BaseActor
	logger *zap.Logger
}

func NewVibrationVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger, mqtt *transport.MQTTClient) *VibrationVehicle {
	return &VibrationVehicle{
		cfg:        cfg,
		simulation: simulation,
		BaseActor:  NewBaseActor(cfg, simulation, logger, pb.MovableType_MOVABLE_TYPE_LAND, mqtt),
		logger:     logger,
	}
}

func (v *VibrationVehicle) ExecuteMission() {
	v.BaseActor.state = ActorStateWorking
}

func (v *VibrationVehicle) CompleteTask() {
	v.performTask()
	v.BaseActor.state = ActorStateDone
}

func (v *VibrationVehicle) performTask() error {
	ctx := context.Background()

	_, err := v.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
		XPos:       v.BaseActor.position.X,
		YPos:       v.BaseActor.position.Y,
		SensorType: pb.SensorType_SENSOR_TYPE_VIBRATION,
		Value:      0,
		Default:    true,
	})
	if err != nil {
		return fmt.Errorf("tile update: %w", err)
	}

	return nil
}
