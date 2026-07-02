package actor

import (
	"actors/internal/config"
	"actors/internal/transport"
	"context"
	"fmt"

	pb "actors/gen"

	"go.uber.org/zap"
)

var _ Actor = (*RescueVehicle)(nil)

const MAX_VEHICLE_CAPACITY = 20

type RescueVehicle struct {
	simulation pb.WorldServiceClient
	BaseActor
	Actor  rescueVehicle
	logger *zap.Logger
}

type rescueVehicle struct {
	capacity int // How many persons this vehicle can transport
	count    int // How many persons are currently in the vehicle
}

func NewRescueVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient, logger *zap.Logger, mqtt *transport.MQTTClient) *RescueVehicle {
	vehicle := &RescueVehicle{
		simulation: simulation,
		BaseActor:  NewBaseActor(cfg, simulation, logger, pb.MovableType_MOVABLE_TYPE_LAND, mqtt),
		Actor: rescueVehicle{
			capacity: MAX_VEHICLE_CAPACITY,
			count:    0,
		},
		logger: logger,
	}
	vehicle.BaseActor.StartRicart()
	return vehicle
}

func (v *RescueVehicle) ExecuteMission() {
	v.BaseActor.state = ActorStateWorking
}

func (v *RescueVehicle) CompleteTask() {
	v.performTask()
	v.BaseActor.state = ActorStateDone
}

func (v *RescueVehicle) performTask() error {
	ctx := context.Background()

	reading, err := v.simulation.TileReading(ctx, &pb.TileReadingRequest{
		XPos:       v.BaseActor.position.X,
		YPos:       v.BaseActor.position.Y,
		SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
	})
	if err != nil {
		return fmt.Errorf("tile reading: %w", err)
	}

	persons := int(reading.GetSensorReading())
	if v.Actor.count+persons <= v.Actor.capacity {
		v.Actor.count += persons

		_, err := v.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
			XPos:       v.BaseActor.position.X,
			YPos:       v.BaseActor.position.Y,
			SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
			Value:      0,
			Default:    false,
		})
		if err != nil {
			return fmt.Errorf("tile update: %w", err)
		}
	} else {
		remaining := persons - (MAX_VEHICLE_CAPACITY - v.Actor.count)
		v.Actor.count = MAX_VEHICLE_CAPACITY

		_, err := v.simulation.TileUpdate(ctx, &pb.TileUpdateRequest{
			XPos:       v.BaseActor.position.X,
			YPos:       v.BaseActor.position.Y,
			SensorType: pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
			Value:      float32(remaining),
		})
		if err != nil {
			return fmt.Errorf("tile update: %w", err)
		}
	}

	return nil
}
