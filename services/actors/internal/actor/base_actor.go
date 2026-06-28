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

const MINIMUM_FUEL_ALERT float64 = 40.0

type BaseActor struct {
	id               string
	actorType        string
	position         shared.Position
	interval         time.Duration
	destination      shared.Position
	path             []shared.Position
	state            ActorState
	movableType      pb.MovableType
	simulation       pb.WorldServiceClient
	mqtt             *transport.MQTTClient
	logger           *zap.Logger
	fuelCapacity     float64 // Maximum fuel capacity
	fuel             float64 // Current fuel capacity
	fuelDrainRate    float64 // Fuel drain rate per second
	lastUpdate       time.Time
	chargingStations []shared.Position
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
		interval:         cfg.IntervalMs,
		state:            ActorStateIdle,
		simulation:       simulation,
		movableType:      movableType,
		mqtt:             mqtt,
		logger:           logger,
		fuel:             cfg.FuelCapacity,
		fuelCapacity:     cfg.FuelCapacity,
		fuelDrainRate:    cfg.FuelDrainRate,
		chargingStations: cfg.ChargingStations,
	}
}

func (a *BaseActor) MovableType() pb.MovableType {
	return a.movableType
}

func (a *BaseActor) Fuel() float64 {
	return a.fuel
}

func (a *BaseActor) FuelCapacity() float64 {
	return a.fuelCapacity
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

func (a *BaseActor) attemptRefueling() error {
	// Wait until refueling is allowed via Ricart Agrawala
	// Return the selected charging station that is being competed for

	if err := a.calculatePathToDestination(a.chargingStations[0]); err != nil { // switch this out with the selected charging station
		return fmt.Errorf("calculate path to charging station: %w", err)
	}

	a.state = ActorStateMovingToRefueling

}

func (a *BaseActor) refuelVehicle() {
	for {
		if a.fuel+5.0 < a.fuelCapacity {
			a.fuel += 5.0
		} else {
			a.fuel = a.fuelCapacity
			a.state = ActorStateIdle
			break
		}
	}
}

func (a *BaseActor) Update(ctx context.Context) {
	now := time.Now()
	dt := now.Sub(a.lastUpdate).Seconds()
	a.lastUpdate = now

	switch a.state {
	case ActorStateBlocked:
		a.send(shared.ActorStateError, "actor blocked")
	case ActorStateRefueling:
		a.refuelVehicle()
		a.send(shared.ActorStateRefueling, "")
	case ActorStateMovingToRefueling:
		a.move()
		a.send(shared.ActorStateRefueling, "")
	case ActorStateDone:
		a.state = ActorStateIdle
	case ActorStateIdle:
		a.drainFuel(0.2, dt)
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

	a.logger.Info("Current fuel", zap.Float64("fuel", a.fuel))
}

func (a *BaseActor) drainFuel(rate float64, dt float64) {
	a.fuel -= a.fuelDrainRate * rate

	if a.fuel < MINIMUM_FUEL_ALERT {
		if err := a.attemptRefueling(); err != nil {
			a.logger.Error("error while refueling")
		}
		a.logger.Warn("refueling should occur")
	}

	if a.fuel < 0 {
		a.fuel = 0
		a.state = ActorStateBlocked
	}
}

func (a *BaseActor) drainFuelAmount() {
	a.fuel -= 20
}

func (a *BaseActor) send(state shared.ActorState, reason string) error {
	// ActorStatus
	status := shared.ActorStatus{
		ActorID:   a.Id(),
		State:     state,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	if err := shared.MqttPublish(a.mqtt.MQTT, shared.ActorStatusParams(a.Id(), status)); err != nil {
		return err
	}

	// ActorTelemetry
	telemetry := shared.ActorTelemetry{
		ActorID:   a.Id(),
		Position:  a.Position(),
		Timestamp: time.Now(),
	}
	return shared.MqttPublish(a.mqtt.MQTT, shared.ActorTelemetryParams(a.Id(), telemetry))
}
