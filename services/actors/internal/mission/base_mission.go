// base_mission.go
package mission

import (
	pb "actors/gen"
	"actors/internal/actor"
	"context"
	"fmt"
	"math/rand"
	"shared"
	"time"
)

type BaseMission struct {
	Id           string
	Type         MissionType
	Destination  shared.Position
	Priority     MissionPriority
	Duration     time.Duration
	MissionTimer *time.Timer
	State        MissionState
	simulation   pb.WorldServiceClient
	actor        actor.Actor
	channel      chan<- *pb.AssignMissionResponse
}

func NewBaseMission(
	cfg MissionConfig,
	ch chan<- *pb.AssignMissionResponse,
	simulation pb.WorldServiceClient,
	actor actor.Actor,
) BaseMission {
	return BaseMission{
		Id:          cfg.Id,
		Type:        cfg.Type,
		Destination: shared.Position{X: cfg.Position.X, Y: cfg.Position.Y},
		Priority:    cfg.Priority,
		Duration:    time.Duration(rand.Intn(10)+1) * time.Second,
		State:       MissionStateAssigned,
		simulation:  simulation,
		actor:       actor,
		channel:     ch,
	}
}

func (b *BaseMission) RunMission(ctx context.Context) error {
	defer close(b.channel)
	ticker := time.NewTicker(b.actor.Interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := b.missionUpdate(); err != nil {
				return nil
			}
		}
	}
}

func (b *BaseMission) missionUpdate() error {
	switch b.State {
	case MissionStateAssigned:
		if err := b.send(b.State); err != nil {
			return err
		}
		if !b.actor.AtPosition(b.Destination) {
			b.State = MissionStateOnTheWay
			b.actor.MoveToLocation(b.Destination)
		} else {
			b.State = MissionStateBusy
		}

	case MissionStateOnTheWay:
		if err := b.send(b.State); err != nil {
			return err
		}
		if b.actor.AtPosition(b.Destination) {
			b.State = MissionStateBusy
		}

	case MissionStateBusy:
		b.actor.ExecuteMission()
		b.startTimer()
		select {
		case <-b.MissionTimer.C:
			b.actor.CompleteTask()
			b.MissionTimer = nil
			b.State = MissionStateDone
		default:
			if err := b.send(b.State); err != nil {
				return err
			}
		}

	case MissionStateDone:
		if err := b.send(b.State); err != nil {
			return err
		}
		return fmt.Errorf("mission done")

	case MissionStateError:
		if err := b.send(b.State); err != nil {
			return err
		}
		return fmt.Errorf("mission error")

	case MissionStateIdle:
		if err := b.send(b.State); err != nil {
			return err
		}
	}
	return nil
}

func (b *BaseMission) startTimer() {
	if b.MissionTimer == nil {
		b.MissionTimer = time.NewTimer(b.Duration)
	}
}

func (b *BaseMission) send(state MissionState) error {
	actorState, err := MissionStateToProto(state)
	if err != nil {
		return err
	}
	b.channel <- &pb.AssignMissionResponse{
		MissionId:       &pb.MissionId{Uuid: b.Id},
		CurrentPosition: &pb.Position{X: uint32(b.actor.Position().X), Y: uint32(b.actor.Position().Y)},
		State:           actorState,
	}
	return nil
}
