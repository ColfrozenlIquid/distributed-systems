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
	RegisterMission(ctx context.Context, req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse) error
	RunMission(ctx context.Context, out chan<- *pb.AssignMissionResponse)
}

func NewActor(cfg config.ActorConfig, simulation pb.WorldServiceClient) (Actor, error) {
	switch cfg.ActorType {
	case "rescue_vehicle":
		return NewRescueVehicle(cfg, simulation), nil
	default:
		return nil, fmt.Errorf("unknown sensor type: %s", cfg.ActorType)
	}
}
