package actor

import (
	"actors/internal/actorstate"
	"actors/internal/config"
	"actors/internal/mission"
	"actors/internal/types"
	"context"
	"fmt"
	"time"

	pb "actors/gen"
)

var _ Actor = (*RescueVehicle)(nil)

type RescueVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	state      *actorstate.State
	mission    mission.Mission
	reading    float32
}

func NewRescueVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient) *RescueVehicle {
	return &RescueVehicle{
		cfg:        cfg,
		simulation: simulation,
		actor: rescueVehicle{
			id: cfg.Id,
			position: types.Position{
				X: int(cfg.XPosition),
				Y: int(cfg.YPosition),
			},
			interval: cfg.IntervalMs,
			reading:  0,
		},
	}
}

func (v *RescueVehicle) Type() string {
	return "rescue_vehicle"
}

func (v *RescueVehicle) Id() string {
	return v.state.GetID()
}

func (v *RescueVehicle) Interval() time.Duration {
	return v.actor.interval
}

func (v *RescueVehicle) Position() types.Position {
	return v.actor.position
}

func (p *RescueVehicle) RegisterMission(ctx context.Context, req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse) error {
	// Generate new mission
	newMission, err := mission.NewMission(req, ch, p.simulation)
	if err != nil {
		return fmt.Errorf("create new mission: %w")
	}

	// Assign new mission inside of the actor
	p.mission = newMission

	p.mission.RunMission()

	return nil
}
