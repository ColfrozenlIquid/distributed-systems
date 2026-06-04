package actor

import (
	"actors/internal/config"
	"actors/internal/types"
	"context"
	"fmt"
	"time"

	pb "actors/gen"

	"go.uber.org/zap"
)

var _ Actor = (*RescueVehicle)(nil)

const MAX_VEHICLE_CAPACITY = 20

type RescueVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	Actor      rescueVehicle
	logger     *zap.Logger
}

type rescueVehicle struct {
	id          string
	position    types.Position
	interval    time.Duration
	destination types.Position
	path        []types.Position
	state       ActorState
	capacity    int // How many persons this vehicle can transport
	count       int // How many persons are currently in the vehicle
}

func NewRescueVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger) *RescueVehicle {
	return &RescueVehicle{
		cfg:        cfg,
		simulation: simulation,
		Actor: rescueVehicle{
			id: cfg.Id,
			position: types.Position{
				X: int(cfg.XPosition),
				Y: int(cfg.YPosition),
			},
			interval: cfg.IntervalMs,
			capacity: MAX_VEHICLE_CAPACITY,
			count:    0,
			state:    ActorStateIdle,
		},
		logger: logger,
	}
}

func (v *RescueVehicle) Type() string {
	return "rescue_vehicle"
}

func (v *RescueVehicle) Id() string {
	return v.Actor.id
}

func (v *RescueVehicle) Interval() time.Duration {
	return v.Actor.interval
}

func (v *RescueVehicle) Position() types.Position {
	return v.Actor.position
}

func (v *RescueVehicle) MovableType() pb.MovableType {
	return pb.MovableType_MOVABLE_TYPE_LAND
}

func (v *RescueVehicle) MoveToLocation(destination types.Position) {
	v.calculatePathToDestination(destination)
	v.Actor.state = ActorStateMoving
}

func (v *RescueVehicle) ExecuteMission() {
	v.Actor.state = ActorStateWorking
}

func (v *RescueVehicle) CompleteTask() {
	v.performTask()
	v.Actor.state = ActorStateDone
}

func (v *RescueVehicle) performTask() error {
	ctx := context.Background()

	reading, err := v.simulation.TileReading(ctx, &pb.TileReadingRequest{
		XPos:       int32(v.Actor.position.X),
		YPos:       int32(v.Actor.position.Y),
		SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
	})
	if err != nil {
		return fmt.Errorf("tile reading: %w", err)
	}

	persons := int(reading.GetSensorReading())
	if v.Actor.count+persons <= v.Actor.capacity {
		v.Actor.count += persons

		_, err := v.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
			XPos:       int32(v.Actor.position.X),
			YPos:       int32(v.Actor.position.Y),
			SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
			Value:      0,
		})
		if err != nil {
			return fmt.Errorf("tile update: %w", err)
		}
	} else {
		remaining := persons - (MAX_VEHICLE_CAPACITY - v.Actor.count)
		v.Actor.count = MAX_VEHICLE_CAPACITY

		_, err := v.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
			XPos:       int32(v.Actor.position.X),
			YPos:       int32(v.Actor.position.Y),
			SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
			Value:      float32(remaining),
		})
		if err != nil {
			return fmt.Errorf("tile update: %w", err)
		}
	}

	return nil
}

func (v *RescueVehicle) calculatePathToDestination(destination types.Position) error {
	resp, err := v.simulation.PathTo(context.Background(), &pb.PathToRequest{
		Source: &pb.WorldPosition{
			X: int32(v.Actor.position.X),
			Y: int32(v.Actor.position.Y),
		},
		Destination: &pb.WorldPosition{
			X: int32(destination.X),
			Y: int32(destination.Y),
		},
		MovableType: v.MovableType(),
	})
	if err != nil {
		return fmt.Errorf("calculate path to destination: %w", err)
	}

	path := make([]types.Position, 0, len(resp.GetPath()))
	for _, p := range resp.GetPath() {
		path = append(path, types.Position{
			X: int(p.GetX()),
			Y: int(p.GetY()),
		})
	}

	v.Actor.destination = destination
	v.Actor.path = path

	return nil
}

func (v *RescueVehicle) move() {
	if v.Actor.position == v.Actor.destination {
		v.logger.Info("Actor arrived at destination")
		return
	}

	v.Actor.position = v.Actor.path[0]

	v.Actor.path = v.Actor.path[1:]
}

func (v *RescueVehicle) AtPosition(pos types.Position) bool {
	return v.Position() == pos
}

func (v *RescueVehicle) Update(ctx context.Context) {
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

// func (p *RescueVehicle) RegisterMission(ctx context.Context, req *pb.AssignMissionRequest, ch chan<- *pb.AssignMissionResponse) error {
// 	// Generate new mission
// 	newMission, err := mission.NewMission(req, ch, p.simulation)
// 	if err != nil {
// 		return fmt.Errorf("create new mission: %w")
// 	}

// 	// Assign new mission inside of the actor
// 	p.mission = newMission

// 	p.mission.RunMission()

// 	return nil
// }
