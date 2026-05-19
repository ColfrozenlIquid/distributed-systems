package actor

import (
	pb "actors/gen"
	"actors/internal/config"
	"fmt"
	"time"
)

type Actor interface {
	Interval() time.Duration
	Id() string
	Position() Position
	Type() string
}

type Position struct {
	X int
	Y int
}

type ActorStatus int

const (
	ActorStatusIdle ActorStatus = iota
	ActorStatusAssigned
	ActorStatusBusy
	ActorStatusError
)

func NewActor(cfg config.ActorConfig, simulation pb.WorldServiceClient) (Actor, error) {
	switch cfg.ActorType {
	case "rescue_vehicle":
		return NewRescueVehicle(cfg, simulation), nil
	default:
		return nil, fmt.Errorf("unknown sensor type: %s", cfg.ActorType)
	}
}
