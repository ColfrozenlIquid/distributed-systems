package mission

import (
	"actors/internal/actor"
	"actors/internal/types"
	"context"
	"fmt"
	"math/rand"
	"time"

	pb "actors/gen"
)

type RescuePersonsMission struct {
	simulation pb.WorldServiceClient
	actor      actor.Actor
	mission    rescuePersonsMission
	channel    chan<- *pb.AssignMissionResponse
}

type rescuePersonsMission struct {
	Id           string
	Type         MissionType
	Destination  types.Position
	Priority     MissionPriority
	Duration     time.Duration
	MissionTimer *time.Timer
	State        MissionState
}

func NewRescuePersonsMission(
	req *pb.AssignMissionRequest,
	ch chan<- *pb.AssignMissionResponse,
	simulation pb.WorldServiceClient,
	actor actor.Actor,
) *RescuePersonsMission {
	missionType, err := MissionTypeToDomain(req.GetMission().GetType())
	if err != nil {
		return nil
	}

	missionPriority, err := MissionPriorityToDomain(req.GetMission().GetPriority())
	if err != nil {
		return nil
	}

	return &RescuePersonsMission{
		simulation: simulation,
		actor:      actor,
		mission: rescuePersonsMission{
			Id:   req.GetMission().Id.GetUuid(),
			Type: missionType,
			Destination: types.Position{
				X: int(req.GetMission().GetPosition().GetX()),
				Y: int(req.GetMission().GetPosition().GetY()),
			},
			Priority: missionPriority,
			Duration: time.Duration(rand.Intn(10)+1) * time.Second,
			State:    MissionStateAssigned,
		},
		channel: ch,
	}
}

func (r *RescuePersonsMission) RunMission(ctx context.Context) error {
	defer close(r.channel)
	ticker := time.NewTicker(r.actor.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := r.missionUpdate(r.channel); err != nil {
				return nil
			}
		}
	}
}

func (r *RescuePersonsMission) StreamMission(ctx context.Context) <-chan *pb.AssignMissionResponse {
	ch := make(chan *pb.AssignMissionResponse, 8)
	r.channel = ch
	go r.RunMission(ctx)
	return ch
}

func (r *RescuePersonsMission) missionUpdate(out chan<- *pb.AssignMissionResponse) error {
	switch r.mission.State {
	case MissionStateAssigned:
		resp, err := r.buildResponse(r.mission.State)
		if err != nil {
			return err
		}
		out <- resp

		if !r.actor.AtPosition(r.mission.Destination) {
			r.mission.State = MissionStateOnTheWay
			r.actor.MoveToLocation(r.mission.Destination)
		} else {
			r.mission.State = MissionStateBusy
		}

	case MissionStateBusy:
		r.actor.ExecuteMission()
		r.runMissionActivity()
		select {
		case <-r.mission.MissionTimer.C:
			r.actor.CompleteTask()
			r.mission.MissionTimer = nil
			r.mission.State = MissionStateDone

		default:
			resp, err := r.buildResponse(r.mission.State)
			if err != nil {
				return err
			}
			out <- resp
		}

	case MissionStateDone:
		resp, err := r.buildResponse(r.mission.State)
		if err != nil {
			return err
		}
		out <- resp
		return fmt.Errorf("mission done")

	case MissionStateError:
		resp, err := r.buildResponse(r.mission.State)
		if err != nil {
			return err
		}
		out <- resp
		return fmt.Errorf("mission error")

	case MissionStateIdle:
		resp, err := r.buildResponse(r.mission.State)
		if err != nil {
			return err
		}
		out <- resp

	case MissionStateOnTheWay:

		resp, err := r.buildResponse(r.mission.State)
		if err != nil {
			return err
		}
		out <- resp

		if r.actor.AtPosition(r.mission.Destination) {
			r.mission.State = MissionStateBusy
		}

	default:
		return nil
	}
	return nil
}

func (r *RescuePersonsMission) runMissionActivity() {
	if r.mission.MissionTimer == nil {
		r.mission.MissionTimer = time.NewTimer(r.mission.Duration)
	}
}

func (r *RescuePersonsMission) buildResponse(state MissionState) (*pb.AssignMissionResponse, error) {
	actorState, err := MissionStateToProto(state)
	if err != nil {
		return nil, err
	}

	return &pb.AssignMissionResponse{
		MissionId: &pb.MissionId{
			Uuid: r.mission.Id,
		},
		CurrentPosition: &pb.Position{
			X: uint32(r.actor.Position().X),
			Y: uint32(r.actor.Position().Y),
		},
		State: actorState,
	}, nil
}
