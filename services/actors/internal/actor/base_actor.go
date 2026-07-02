package actor

import (
	pb "actors/gen"
	"actors/internal/config"
	"actors/internal/transport"
	"context"
	"fmt"
	"shared"
	"sync"
	"time"

	"go.uber.org/zap"
)

const MINIMUM_FUEL_ALERT float64 = 40.0

type BaseActor struct {
	mu                sync.Mutex
	id                string
	actorType         string
	position          shared.Position
	interval          time.Duration
	destination       shared.Position
	path              []shared.Position
	state             ActorState
	movableType       pb.MovableType
	simulation        pb.WorldServiceClient
	mqtt              *transport.MQTTClient
	logger            *zap.Logger
	fuelCapacity      float64 // Maximum fuel capacity
	fuel              float64 // Current fuel capacity
	fuelDrainRate     float64 // Fuel drain rate per second
	lastUpdate        time.Time
	chargingStations  []shared.Position
	chargingStation   shared.Position
	ricartNode        *shared.RicartNode
	refuelRequested   bool
	resumeState       ActorState
	resumeDestination shared.Position
	resumePath        []shared.Position
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
		lastUpdate:       time.Now(),
		chargingStations: cfg.ChargingStations,
	}
}

func (a *BaseActor) StartRicart() {
	if len(a.chargingStations) == 0 {
		a.logger.Warn("ricart disabled: no charging stations configured")
		return
	}

	a.chargingStation = a.chargingStations[0]
	resourceID := shared.ChargingStationResourceID(a.chargingStation)
	streams := shared.NewRicartStreams()
	node := shared.NewRicartNode(a.id, shared.ActorType(a.actorType), resourceID, a.mqtt.MQTT, a.enterRefuelingCriticalSection)
	if err := node.Start(context.Background(), streams); err != nil {
		a.logger.Error("start ricart node", zap.Error(err))
		return
	}
	a.ricartNode = node
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
	if len(a.path) == 0 {
		a.logger.Warn("actor has no path to destination", zap.Any("destination", a.destination))
		a.state = ActorStateBlocked
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
	if a.refuelRequested ||
		a.state == ActorStateWaitingForRefueling ||
		a.state == ActorStateMovingToRefueling ||
		a.state == ActorStateRefueling {
		return nil
	}

	if a.ricartNode == nil {
		return fmt.Errorf("ricart node is not available")
	}
	if len(a.chargingStations) == 0 {
		return fmt.Errorf("no charging stations configured")
	}

	a.chargingStation = a.chargingStations[0]
	a.resumeState = a.state
	a.resumeDestination = a.destination
	a.resumePath = append([]shared.Position(nil), a.path...)

	if err := a.calculatePathToDestination(a.chargingStation); err != nil {
		return fmt.Errorf("calculate path to charging station: %w", err)
	}

	a.refuelRequested = true
	a.state = ActorStateWaitingForRefueling

	return a.ricartNode.RequestCS()
}

func (a *BaseActor) enterRefuelingCriticalSection() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.refuelRequested || a.state != ActorStateWaitingForRefueling {
		return
	}
	a.state = ActorStateMovingToRefueling
}

func (a *BaseActor) refuelVehicle() {
	if a.fuel+5.0 < a.fuelCapacity {
		a.fuel += 5.0
		return
	}

	a.fuel = a.fuelCapacity
	a.refuelRequested = false
	if a.ricartNode != nil {
		if err := a.ricartNode.ReleaseCS(); err != nil {
			a.logger.Error("release ricart critical section", zap.Error(err))
		}
	}
	a.restoreStateAfterRefueling()
}

func (a *BaseActor) restoreStateAfterRefueling() {
	a.destination = a.resumeDestination
	a.path = append([]shared.Position(nil), a.resumePath...)
	if a.resumeState == ActorStateWaitingForRefueling ||
		a.resumeState == ActorStateMovingToRefueling ||
		a.resumeState == ActorStateRefueling {
		a.state = ActorStateIdle
		return
	}
	a.state = a.resumeState
}

func (a *BaseActor) Update(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	dt := now.Sub(a.lastUpdate).Seconds()
	a.lastUpdate = now

	switch a.state {
	case ActorStateBlocked:
		a.recoverFromFuelOutage()
	case ActorStateWaitingForRefueling:
		a.drainFuelWhileWaitingForRefueling(0.2, dt)
		a.send(shared.ActorStateRefueling, "waiting for charging station")
	case ActorStateRefueling:
		a.refuelVehicle()
		a.send(shared.ActorStateRefueling, "")
	case ActorStateMovingToRefueling:
		a.move()
		if a.position == a.chargingStation {
			a.state = ActorStateRefueling
		}
		a.send(shared.ActorStateRefueling, "")
	case ActorStateDone:
		a.state = ActorStateIdle
	case ActorStateIdle:
		a.drainFuel(0.2, dt)
		a.send(shared.ActorStateIdle, "")
	case ActorStateWorking:
		a.drainFuel(1, dt)
		a.send(shared.ActorStateOnAMission, "")
	case ActorStateMoving:
		a.drainFuel(1, dt)
		a.move()
		a.send(shared.ActorStateOnAMission, "")
	default:
		a.logger.Debug("unknown actor state ")
		a.send(shared.ActorStateError, "unknown actor state")
	}

	a.logger.Info("Current fuel", zap.Float64("fuel", a.fuel))
}

func (a *BaseActor) drainFuel(rate float64, dt float64) {
	a.fuel -= a.fuelDrainRate * rate * dt

	if a.fuel < MINIMUM_FUEL_ALERT {
		if err := a.attemptRefueling(); err != nil {
			a.logger.Error("error while refueling", zap.Error(err))
		}
		a.logger.Warn("refueling should occur")
	}

	if a.fuel < 0 {
		a.fuel = 0
		if a.refuelRequested {
			return
		}
		a.state = ActorStateBlocked
	}
}

func (a *BaseActor) drainFuelWhileWaitingForRefueling(rate float64, dt float64) {
	a.fuel -= a.fuelDrainRate * rate * dt
	if a.fuel < 0 {
		a.fuel = 0
	}
}

func (a *BaseActor) recoverFromFuelOutage() {
	if a.fuel > 0 {
		a.state = ActorStateIdle
		a.send(shared.ActorStateIdle, "")
		return
	}

	if err := a.attemptRefueling(); err != nil {
		a.logger.Error("error while recovering from fuel outage", zap.Error(err))
		a.send(shared.ActorStateError, "actor blocked")
		return
	}
	a.send(shared.ActorStateRefueling, "waiting for charging station")
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
