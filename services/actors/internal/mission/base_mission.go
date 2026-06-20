// base_mission.go
package mission

import (
	pb "actors/gen"
	"actors/internal/actor"
	"actors/internal/transport"
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
	State        shared.MissionState
	simulation   pb.WorldServiceClient
	actor        actor.Actor
	mqtt         *transport.MQTTClient
}

func NewBaseMission(
	cfg MissionConfig,
	simulation pb.WorldServiceClient,
	actor actor.Actor,
	mqtt *transport.MQTTClient,
) BaseMission {
	return BaseMission{
		Id:          cfg.Id,
		Type:        cfg.Type,
		Destination: shared.Position{X: cfg.Position.X, Y: cfg.Position.Y},
		Priority:    cfg.Priority,
		Duration:    time.Duration(rand.Intn(10)+1) * time.Second,
		State:       shared.MissionStateAssigned,
		simulation:  simulation,
		actor:       actor,
		mqtt:        mqtt,
	}
}

func (b *BaseMission) RunMission(ctx context.Context) error {
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
	case shared.MissionStateAssigned:
		if err := b.send(b.State); err != nil {
			return err
		}
		if !b.actor.AtPosition(b.Destination) {
			b.State = shared.MissionStateOnTheWay
			b.actor.MoveToLocation(b.Destination)
		} else {
			b.State = shared.MissionStateBusy
		}

	case shared.MissionStateOnTheWay:
		if err := b.send(b.State); err != nil {
			return err
		}
		if b.actor.AtPosition(b.Destination) {
			b.State = shared.MissionStateBusy
		}

	case shared.MissionStateBusy:
		b.actor.ExecuteMission()
		b.startTimer()
		select {
		case <-b.MissionTimer.C:
			b.actor.CompleteTask()
			b.MissionTimer = nil
			b.State = shared.MissionStateCompleted
		default:
			if err := b.send(b.State); err != nil {
				return err
			}
		}

	case shared.MissionStateCompleted:
		if err := b.send(b.State); err != nil {
			return err
		}
		return fmt.Errorf("mission done")

	case shared.MissionStateFailed:
		if err := b.send(b.State); err != nil {
			return err
		}
		return fmt.Errorf("mission error")
	}
	return nil
}

func (b *BaseMission) startTimer() {
	if b.MissionTimer == nil {
		b.MissionTimer = time.NewTimer(b.Duration)
	}
}

func (b *BaseMission) send(state shared.MissionState) error {
	// retain = false, Qos = 1
	b.mqtt.MQTT.Publish(shared.MissionUpdateTopic(b.Id), byte(1), false, shared.MissionUpdate{
		MissionID: b.Id,
		ActorID:   b.actor.Id(),
		State:     b.State,
		Timestamp: time.Now(),
	})

	return nil
}
