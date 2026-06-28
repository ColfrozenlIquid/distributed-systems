package actor

import (
	pb "actors/gen"
	"actors/internal/config"
	"actors/internal/transport"
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
	FuelCapacity() float64
	Fuel() float64

	MoveToLocation(destination shared.Position)
	AtPosition(pos shared.Position) bool
	ExecuteMission()
	CompleteTask()

	Update(ctx context.Context)
}

func NewActor(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger, mqtt *transport.MQTTClient) (Actor, error) {
	switch cfg.ActorType {
	case "rescue_vehicle":
		return NewRescueVehicle(cfg, simulation, logger, mqtt), nil
	case "vibration_vehicle":
		return NewVibrationVehicle(cfg, simulation, logger, mqtt), nil
	case "water_vehicle":
		return NewWaterVehicle(cfg, simulation, logger, mqtt), nil
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
	ActorStateRefueling
	ActorStateMovingToRefueling
)
