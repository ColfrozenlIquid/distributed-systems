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

type PumpWaterMission struct {
	simulation pb.WorldServiceClient
	actor      actor.Actor
	mission    pumpWaterMission
	channel    chan<- *pb.AssignMissionResponse
}

type pumpWaterMission struct {
	Id           string
	Type         MissionType
	Destination  types.Position
	Priority     MissionPriority
	Duration     time.Duration
	MissionTimer *time.Timer
	State        MissionState
}

func NewPumpWaterMission(
	req *pb.AssignMissionRequest,
	ch chan<- *pb.AssignMissionResponse,
	simulation pb.WorldServiceClient,
	actor actor.Actor,
) *PumpWaterMission {
	missionType, err := MissionTypeToDomain(req.GetMission().GetType())
	if err != nil {
		return nil
	}

	missionPriority, err := MissionPriorityToDomain(req.GetMission().GetPriority())
	if err != nil {
		return nil
	}

	return &PumpWaterMission{
		simulation: simulation,
		actor:      actor,
		mission: pumpWaterMission{
			Id:   req.GetMission().Id.GetUuid(),
			Type: missionType,
			Destination: types.Position{
				X: int(req.GetMission().GetPosition().GetX()),
				Y: int(req.GetMission().GetPosition().GetY()),
			},
			Priority: missionPriority,
			Duration: time.Duration(rand.Intn(10)+1) * time.Second,
		},
		channel: ch,
	}
}

func (p *PumpWaterMission) RunMission(ctx context.Context) error {
	defer close(p.channel)
	ticker := time.NewTicker(p.actor.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := p.missionUpdate(p.channel); err != nil {
				return nil
			}
		}
	}
}

func (p *PumpWaterMission) StreamMission(ctx context.Context) <-chan *pb.AssignMissionResponse {
	ch := make(chan *pb.AssignMissionResponse, 8)
	p.channel = ch
	go p.RunMission(ctx)
	return ch
}

func (p *PumpWaterMission) missionUpdate(out chan<- *pb.AssignMissionResponse) error {
	switch p.mission.State {
	case MissionStateAssigned:
		resp, err := p.buildResponse(p.mission.State)
		if err != nil {
			return err
		}
		out <- resp

		if !p.actor.AtPosition(p.mission.Destination) {
			p.mission.State = MissionStateOnTheWay
			p.actor.MoveToLocation(p.mission.Destination)
		} else {
			p.mission.State = MissionStateBusy
		}

	case MissionStateBusy:
		p.runMissionActivity()
		select {
		case <-p.mission.MissionTimer.C:
			p.mission.MissionTimer = nil
			p.mission.State = MissionStateDone
		default:
			// Do nothing
		}

	case MissionStateDone:
		resp, err := p.buildResponse(p.mission.State)
		if err != nil {
			return err
		}
		out <- resp
		return fmt.Errorf("mission done")

	case MissionStateError:
		resp, err := p.buildResponse(p.mission.State)
		if err != nil {
			return err
		}
		out <- resp
		return fmt.Errorf("mission error")

	case MissionStateIdle:
		resp, err := p.buildResponse(p.mission.State)
		if err != nil {
			return err
		}
		out <- resp

	case MissionStateOnTheWay:

		resp, err := p.buildResponse(p.mission.State)
		if err != nil {
			return err
		}
		out <- resp

	default:
		return nil
	}
	return nil
}

func (p *PumpWaterMission) runMissionActivity() {
	if p.mission.MissionTimer == nil {
		p.mission.MissionTimer = time.NewTimer(p.mission.Duration)
	}
}

func (p *PumpWaterMission) buildResponse(state MissionState) (*pb.AssignMissionResponse, error) {
	actorState, err := MissionStateToProto(state)
	if err != nil {
		return nil, err
	}

	return &pb.AssignMissionResponse{
		MissionId: &pb.MissionId{
			Uuid: p.mission.Id,
		},
		CurrentPosition: &pb.Position{
			X: uint32(p.actor.Position().X),
			Y: uint32(p.actor.Position().Y),
		},
		State: actorState,
	}, nil
}
