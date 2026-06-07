package actor

import (
	"actors/internal/config"
	"context"
	"fmt"
	"shared"
	"time"

	pb "actors/gen"

	"go.uber.org/zap"
)

var _ Actor = (*VibrationVehicle)(nil)

type VibrationVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	Actor      vibrationVehicle
	logger     *zap.Logger
}

type vibrationVehicle struct {
	id          string
	position    shared.Position
	interval    time.Duration
	destination shared.Position
	path        []shared.Position
	state       ActorState
}

func NewVibrationVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger) *VibrationVehicle {
	return &VibrationVehicle{
		cfg:        cfg,
		simulation: simulation,
		Actor: vibrationVehicle{
			id: cfg.Id,
			position: shared.Position{
				X: cfg.XPosition,
				Y: cfg.YPosition,
			},
			interval: cfg.IntervalMs,
			state:    ActorStateIdle,
		},
		logger: logger,
	}
}

func (v *VibrationVehicle) Type() string {
	return "vibration_vehicle"
}

func (v *VibrationVehicle) Id() string {
	return v.Actor.id
}

func (v *VibrationVehicle) Interval() time.Duration {
	return v.Actor.interval
}

func (v *VibrationVehicle) Position() shared.Position {
	return v.Actor.position
}

func (v *VibrationVehicle) MovableType() pb.MovableType {
	return pb.MovableType_MOVABLE_TYPE_LAND
}

func (v *VibrationVehicle) MoveToLocation(destination shared.Position) {
	v.calculatePathToDestination(destination)
	v.Actor.state = ActorStateMoving
}

func (v *VibrationVehicle) ExecuteMission() {
	v.Actor.state = ActorStateWorking
}

func (v *VibrationVehicle) CompleteTask() {
	v.performTask()
	v.Actor.state = ActorStateDone
}

func (v *VibrationVehicle) performTask() error {
	ctx := context.Background()

	_, err := v.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
		XPos:       v.Actor.position.X,
		YPos:       v.Actor.position.Y,
		SensorType: pb.SensorType_SENSOR_TYPE_VIBRATION,
		Value:      0,
		Default:    true,
	})
	if err != nil {
		return fmt.Errorf("tile update: %w", err)
	}

	return nil
}

func (v *VibrationVehicle) calculatePathToDestination(destination shared.Position) error {
	resp, err := v.simulation.PathTo(context.Background(), &pb.PathToRequest{
		Source: &pb.WorldPosition{
			X: v.Actor.position.X,
			Y: v.Actor.position.Y,
		},
		Destination: &pb.WorldPosition{
			X: destination.X,
			Y: destination.Y,
		},
		MovableType: v.MovableType(),
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

	v.Actor.destination = destination
	v.Actor.path = path

	return nil
}

func (v *VibrationVehicle) move() {
	if v.Actor.position == v.Actor.destination {
		v.logger.Info("Actor arrived at destination")
		return
	}

	v.Actor.position = v.Actor.path[0]

	v.Actor.path = v.Actor.path[1:]
}

func (v *VibrationVehicle) AtPosition(pos shared.Position) bool {
	return v.Position() == pos
}

func (v *VibrationVehicle) Update(ctx context.Context) {
	switch v.Actor.state {
	case ActorStateBlocked:
		v.logger.Info("actor is blocked")
	case ActorStateDone:
		v.logger.Info("actor is done")
		v.Actor.state = ActorStateIdle
	case ActorStateIdle:
		v.logger.Info("actor is idle")
	case ActorStateWorking:
		v.logger.Info("actor is executing the mission")
	case ActorStateMoving:
		v.logger.Info("actor is moving")
		v.move()
	default:
		v.logger.Debug("unknown actor state ")
	}
}
