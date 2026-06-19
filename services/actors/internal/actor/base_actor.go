package actor

import (
	pb "actors/gen"
	"actors/internal/config"
	"actors/internal/transport"
	"context"
	"fmt"
	"shared"
	"time"

	"go.uber.org/zap"
)

type BaseActor struct {
	id          string
	actorType   string
	position    shared.Position
	interval    time.Duration
	destination shared.Position
	path        []shared.Position
	state       ActorState
	movableType pb.MovableType
	simulation  pb.WorldServiceClient
	mqtt        *transport.MQTTClient
	logger      *zap.Logger
}

func NewBaseActor(
	cfg config.ActorConfig,
	simulation pb.WorldServiceClient,
	logger *zap.Logger,
	movableType pb.MovableType,
	mqtt *transport.MQTTClient,
) BaseActor {
	return BaseActor{
		id:        cfg.Id,
		actorType: cfg.ActorType,
		position: shared.Position{
			X: cfg.XPosition,
			Y: cfg.YPosition,
		},
		interval:    cfg.IntervalMs,
		state:       ActorStateIdle,
		simulation:  simulation,
		movableType: movableType,
		mqtt:        mqtt,
		logger:      logger,
	}
}

func (a *BaseActor) MovableType() pb.MovableType {
	return a.movableType
}

func (a *BaseActor) Type() string {
	return a.actorType
}

func (a *BaseActor) Id() string {
	return a.id
}

func (a *BaseActor) Interval() time.Duration {
	return a.interval
}

func (a *BaseActor) Position() shared.Position {
	return a.position
}

func (a *BaseActor) MoveToLocation(destination shared.Position) {
	a.calculatePathToDestination(destination)
	a.state = ActorStateMoving
}

func (a *BaseActor) AtPosition(pos shared.Position) bool {
	return a.position == pos
}

func (a *BaseActor) move() {
	if a.position == a.destination {
		a.logger.Info("actor arrived at destination")
		return
	}
	a.position = a.path[0]
	a.path = a.path[1:]
}

func (a *BaseActor) calculatePathToDestination(destination shared.Position) error {
	resp, err := a.simulation.PathTo(context.Background(), &pb.PathToRequest{
		Source: &pb.WorldPosition{
			X: a.position.X,
			Y: a.position.Y,
		},
		Destination: &pb.WorldPosition{
			X: destination.X,
			Y: destination.Y,
		},
		MovableType: a.movableType,
	})
	if err != nil {
		return fmt.Errorf("calculate path to destination: %w", err)
	}

	path := make([]shared.Position, 0, len(resp.GetPath()))
	for _, p := range resp.GetPath() {
		path = append(path, shared.Position{
			X: p.GetX(),
			Y: p.GetY(),
		})
	}

	a.destination = destination
	a.path = path

	return nil
}

func (a *BaseActor) Update(ctx context.Context) {
	switch a.state {
	case ActorStateBlocked:
		a.send(shared.ActorStateError, "actor blocked")
	case ActorStateDone:
		a.state = ActorStateIdle
	case ActorStateIdle:
		a.send(shared.ActorStateIdle, "")
	case ActorStateWorking:
		a.send(shared.ActorStateOnAMission, "")
	case ActorStateMoving:
		a.move()
		a.send(shared.ActorStateOnAMission, "")
	default:
		a.logger.Debug("unknown actor state ")
		a.send(shared.ActorStateError, "unknown actor state")
	}
}

func (a *BaseActor) send(state shared.ActorState, reason string) error {
	// ActorStatus
	status := shared.ActorStatus{
		ActorID:   a.Id(),
		State:     state,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	// Retain = true, QoS = 1
	a.mqtt.MQTT.Publish(shared.ActorStatusTopic(a.Id()), byte(1), true, status)

	// ActorTelemetry
	telemetry := shared.ActorTelemetry{
		ActorID:   a.Id(),
		Position:  a.Position(),
		Timestamp: time.Now(),
	}
	// Retain = false, QoS = 0
	a.mqtt.MQTT.Publish(shared.ActorTelemetryTopic(a.Id()), byte(0), false, telemetry)

	return nil
}
