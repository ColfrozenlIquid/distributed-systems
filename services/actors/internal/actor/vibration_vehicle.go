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
	vehicle := &VibrationVehicle{
		cfg:        cfg,
		simulation: simulation,
		BaseActor:  NewBaseActor(cfg, simulation, logger, pb.MovableType_MOVABLE_TYPE_LAND, mqtt),
		logger:     logger,
	}
	vehicle.BaseActor.StartRicart()
	return vehicle
}

func (v *VibrationVehicle) ExecuteMission() {
	v.BaseActor.BeginMissionWork()
}

func (v *VibrationVehicle) CompleteTask() {
	if !v.BaseActor.CanCompleteMission() {
		return
	}
	v.performTask()
	v.BaseActor.MarkMissionDone()
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
