package actor

import (
	pb "actors/gen"
	"actors/internal/config"
	"actors/internal/types"
	"context"
	"fmt"
	"time"
)

type Actor interface {
	Interval() time.Duration
	Id() string
	Position() types.Position
	Type() string
	RegisterMission(mission *pb.Mission) error
	RunMission(ctx context.Context, req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse)
}

type ActorStatus int

const (
	ActorStatusIdle ActorStatus = iota
	ActorStatusAssigned
	ActorStatsusOnTheWay
	ActorStatusBusy
	ActorStatusDone
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
