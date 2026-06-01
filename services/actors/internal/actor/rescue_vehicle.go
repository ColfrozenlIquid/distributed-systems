package actor

import (
	"actors/internal/config"
	"actors/internal/mission"
	"actors/internal/types"
	"context"
	"math/rand"
	"time"

	pb "actors/gen"
)

var _ Actor = (*RescueVehicle)(nil)

type RescueVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	actor      rescueVehicle
	mission    mission.Mission
	channel    chan<- *pb.AssignMissionResponse
}

func (v *RescueVehicle) RegisterMission(ctx context.Context, req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse) error {
	var missionRequest = req.GetMission()

	if err := mission.MissionWithinBounds(missionRequest.GetPosition()); err != nil {
		return err
	}

	missionType, err := mission.MissionTypeToDomain(missionRequest.GetType())
	if err != nil {
		return err
	}

	missionPriority, err := mission.MissionPriorityToDomain(missionRequest.GetPriority())
	if err != nil {
		return err
	}

	v.mission = mission.Mission{
		Id:   missionRequest.GetId().GetUuid(),
		Type: missionType,
		Position: types.Position{
			X: int(missionRequest.GetPosition().GetX()),
			Y: int(missionRequest.GetPosition().GetY()),
		},
		Priority: missionPriority,
		State:    mission.ActorStateAssigned,
		Duration: time.Duration(rand.Intn(10)+1) * time.Second,
	}

	v.channel = ch

	return nil
}

func (v *RescueVehicle) RunMission(ctx context.Context, out chan<- *pb.AssignMissionResponse) {
	defer close(out)
	ticker := time.NewTicker(v.actor.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := v.missionUpdate(out); err != nil {
				return
			}
		}
	}
}

func (v *RescueVehicle) StreamMission(ctx context.Context) <-chan *pb.AssignMissionResponse {
	ch := make(chan *pb.AssignMissionResponse, 8)
	v.channel = ch
	go v.RunMission(ctx, ch)
	return ch
}

func (v *RescueVehicle) missionUpdate(out chan<- *pb.AssignMissionResponse) error {
	switch v.mission.State {
	case mission.ActorStateAssigned:
		resp, err := v.buildResponse(v.mission.State)
		if err != nil {
			return err
		}
		out <- resp
	case mission.ActorStateBusy:
		v.runMissionActivity()
		select {
		case <-v.mission.MissionTimer.C:
			v.mission.MissionTimer = nil
			v.mission.State = mission.ActorStateDone
		default:
			// Do nothing
		}
	case mission.ActorStateDone:
	case mission.ActorStateError:
	case mission.ActorStateIdle:
		resp, err := v.buildResponse(v.mission.State)
		if err != nil {
			return err
		}
		out <- resp
	case mission.ActorStateOnTheWay:
		v.moveToLocation()
	default:
		return nil
	}
	return nil
}

func (v *RescueVehicle) moveToLocation() {
	// Actor is not yet at its target destination
	if v.actor.position.X == v.actor.destination.X && v.actor.position.Y == v.actor.destination.Y {
		v.mission.State = mission.ActorStateBusy
		return
	}

}

func (v *RescueVehicle) runMissionActivity() {
	if v.mission.MissionTimer == nil {
		v.mission.MissionTimer = time.NewTimer(v.mission.Duration)
	}
}

func (v *RescueVehicle) buildResponse(state mission.ActorState) (*pb.AssignMissionResponse, error) {
	actorState, err := mission.ActorStateToProto(state)
	if err != nil {
		return nil, err
	}

	return &pb.AssignMissionResponse{
		MissionId: &pb.MissionId{
			Uuid: v.mission.Id,
		},
		CurrentPosition: &pb.Position{
			X: uint32(v.actor.position.X),
			Y: uint32(v.actor.position.Y),
		},
		State: actorState,
	}, nil
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
		},
	}
}
