package actor

import (
	pb "actors/gen"
	"actors/internal/config"
	"context"
	"fmt"
	"shared"
	"time"

	"go.uber.org/zap"
)

type Actor interface {
	Interval() time.Duration
	Id() string
	Position() shared.Position
	Type() string
	MovableType() pb.MovableType

	MoveToLocation(destination shared.Position)
	AtPosition(pos shared.Position) bool
	ExecuteMission()
	CompleteTask()

	Update(ctx context.Context)
}

func NewActor(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger) (Actor, error) {
	switch cfg.ActorType {
	case "rescue_vehicle":
		return NewRescueVehicle(cfg, simulation, logger), nil
	case "vibration_vehicle":
		return NewVibrationVehicle(cfg, simulation, logger), nil
	case "water_vehicle":
		return NewWaterVehicle(cfg, simulation, logger), nil
	default:
		return nil, fmt.Errorf("unknown sensor type: %s", cfg.ActorType)
	}
}

type ActorState int

const (
	ActorStateIdle ActorState = iota
	ActorStateMoving
	ActorStateBlocked
	ActorStateDone
	ActorStateWorking
)
