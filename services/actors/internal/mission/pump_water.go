package mission

import (
	"actors/internal/actorstate"
	"actors/internal/types"
	"context"
	"fmt"
	"math/rand"
	"time"

	pb "actors/gen"
)

type PumpWaterMission struct {
	simulation pb.WorldServiceClient
	actor      *actorstate.State
	mission    pumpWaterMission
	channel    chan<- *pb.AssignMissionResponse
}

type pumpWaterMission struct {
	Id           string
	Type         MissionType
	Position     types.Position
	Priority     MissionPriority
	State        ActorState
	Duration     time.Duration
	MissionTimer *time.Timer
}

func NewPumpWaterMission(
	req *pb.AssignMissionRequest,
	ch chan<- *pb.AssignMissionResponse,
	simulation pb.WorldServiceClient,
	actor *actorstate.State,
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
			Position: types.Position{
				X: int(req.GetMission().GetPosition().GetX()),
				Y: int(req.GetMission().GetPosition().GetY()),
			},
			Priority: missionPriority,
			State:    ActorStateAssigned,
			Duration: time.Duration(rand.Intn(10)+1) * time.Second,
		},
		channel: ch,
	}
}

func (p *PumpWaterMission) RunMission(ctx context.Context) error {
	defer close(p.channel)
	ticker := time.NewTicker(p.actorCtx.Interval)
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

func (p *PumpWaterMission) calculatePathToDestination(ctx context.Context) error {
	resp, err := p.simulation.PathTo(ctx, &pb.PathToRequest{
		Source: &pb.WorldPosition{
			X: int32(p.actorCtx.Position.X),
			Y: int32(p.actorCtx.Position.Y),
		},
		Destination: &pb.WorldPosition{
			X: int32(p.actorCtx.Destination.X),
			Y: int32(p.actorCtx.Destination.Y),
		},
		MovableType: pb.MovableType_MOVABLE_TYPE_LAND,
	})
	if err != nil {
		return err
	}

	path := make([]types.Position, 0, len(resp.GetPath()))
	for _, p := range resp.GetPath() {
		path = append(path, types.Position{
			X: int(p.GetX()),
			Y: int(p.GetY()),
		})
	}

	p.actorCtx.Path = &path

	return nil
}

func (p *PumpWaterMission) StreamMission(ctx context.Context) <-chan *pb.AssignMissionResponse {
	ch := make(chan *pb.AssignMissionResponse, 8)
	p.channel = ch
	go p.RunMission(ctx, ch)
	return ch
}

func (p *PumpWaterMission) missionUpdate(out chan<- *pb.AssignMissionResponse) error {
	switch p.State {
	case ActorStateAssigned:
		resp, err := p.buildResponse(p.State)
		if err != nil {
			return err
		}
		out <- resp
		p.State = ActorStateOnTheWay

	case ActorStateBusy:
		p.runMissionActivity()
		select {
		case <-p.MissionTimer.C:
			p.MissionTimer = nil
			p.State = ActorStateDone
		default:
			// Do nothing
		}

	case ActorStateDone:
		resp, err := p.buildResponse(p.State)
		if err != nil {
			return err
		}
		out <- resp
		return fmt.Errorf("mission done")

	case ActorStateError:
		resp, err := p.buildResponse(p.State)
		if err != nil {
			return err
		}
		out <- resp
		return fmt.Errorf("mission error")

	case ActorStateIdle:
		resp, err := p.buildResponse(p.State)
		if err != nil {
			return err
		}
		out <- resp

	case ActorStateOnTheWay:
		p.moveToLocation()

		resp, err := p.buildResponse(p.State)
		if err != nil {
			return err
		}
		out <- resp

	default:
		return nil
	}
	return nil
}

func (p *PumpWaterMission) moveToLocation() {
	if len(p.actorCtx.Path) == 0 {
		return
	}

	if p.actor.position == p.actor.destination {
		p.State = ActorStateBusy
		return
	}

	p.actor.position = p.actor.Path[0]

	p.actor.Path = p.actor.Path[1:]
}

func (p *PumpWaterMission) runMissionActivity() {
	if p.MissionTimer == nil {
		p.MissionTimer = time.NewTimer(p.Duration)
	}
}

func (p *PumpWaterMission) buildResponse(state ActorState) (*pb.AssignMissionResponse, error) {
	actorState, err := ActorStateToProto(state)
	if err != nil {
		return nil, err
	}

	return &pb.AssignMissionResponse{
		MissionId: &pb.MissionId{
			Uuid: p.Id,
		},
		CurrentPosition: &pb.Position{
			X: uint32(p.actor.position.X),
			Y: uint32(p.actor.position.Y),
		},
		State: actorState,
	}, nil
}
