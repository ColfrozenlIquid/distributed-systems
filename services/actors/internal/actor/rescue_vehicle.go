package actor

import (
	"actors/internal/config"
	"actors/internal/mission"
	"actors/internal/types"
	"context"
	"time"

	pb "actors/gen"
)

var _ Actor = (*RescueVehicle)(nil)

type RescueVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	actor      rescueVehicle
	mission    mission.Mission
}

func (v *RescueVehicle) RunMission(ctx context.Context, req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse) {
	defer close(ch)

}

func (v *RescueVehicle) RegisterMission(miss *pb.Mission) error {
	if err := mission.MissionWithinBounds(miss.GetPosition()); err != nil {
		return err
	}

	missionType, err := mission.MissionTypeToDomain(miss.GetType())
	if err != nil {
		return err
	}

	missionPriority, err := mission.MissionPriorityToDomain(miss.GetPriority())
	if err != nil {
		return err
	}

	v.mission = mission.Mission{
		Id:   miss.GetId().GetUuid(),
		Type: missionType,
		Position: types.Position{
			X: int(miss.GetPosition().GetX()),
			Y: int(miss.GetPosition().GetY()),
		},
		Priority: missionPriority,
	}

	return nil
}

func (v *RescueVehicle) Type() string {
	return "rescue_vehicle"
}

func (v *RescueVehicle) Id() string {
	return v.actor.id
}

func (v *RescueVehicle) Interval() time.Duration {
	return v.actor.interval
}

func (v *RescueVehicle) Position() types.Position {
	return v.actor.position
}

type rescueVehicle struct {
	id          string
	position    types.Position
	interval    time.Duration
	reading     float32
	destination types.Position
	status      ActorStatus
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
			destination: types.Position{
				X: 0,
				Y: 0,
			},
			status: ActorStatusIdle,
		},
	}
}
