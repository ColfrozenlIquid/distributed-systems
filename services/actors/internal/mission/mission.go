package mission

import (
	pb "actors/gen"
	proto "actors/gen"
	"actors/internal/actor"
	"actors/internal/types"
	"context"
	"shared"

	"fmt"
)

type Mission interface {
	RunMission(ctx context.Context) error
}

type MissionConfig struct {
	Id       string
	Type     MissionType
	Position shared.Position
	Priority MissionPriority
}

func NewMission(
	req *pb.AssignMissionRequest,
	ch chan<- *pb.AssignMissionResponse,
	simulation pb.WorldServiceClient,
	actor actor.Actor,
) (Mission, error) {
	missionType, err := MissionTypeToDomain(req.GetMission().GetType())
	if err != nil {
		return nil, fmt.Errorf("unknown mission type: %w")
	}

	switch missionType {
	case MissionTypePumpWater:
		return NewPumpWaterMission(req, ch, simulation, actor), nil

	case MissionTypeRescuePersons:
		return NewRescuePersonsMission(req, ch, simulation, actor), nil

	// case MissionTypeStabilizeVibration:
	// 	return NewStabilizeVibrationMission(req, ch, simulation), nil

	default:
		return nil, fmt.Errorf("unknown mission type: %s", missionType)
	}
}

type MissionType int

const (
	MissionTypePumpWater MissionType = iota
	MissionTypeStabilizeVibration
	MissionTypeRescuePersons
)

type MissionPriority int

const (
	MissionPriorityLow MissionPriority = iota
	MissionPriorityMedium
	MissionPriorityHigh
)

func MissionTypeToDomain(missionType pb.MissionType) (MissionType, error) {
	switch missionType {
	case pb.MissionType_MISSION_TYPE_PUMP_WATER:
		return MissionTypePumpWater, nil
	case pb.MissionType_MISSION_TYPE_STABILIZE_VIBRATION:
		return MissionTypeStabilizeVibration, nil
	case pb.MissionType_MISSION_TYPE_RESCUE_PERSONS:
		return MissionTypeRescuePersons, nil
	default:
		return MissionTypePumpWater, fmt.Errorf("unknown mission type")
	}
}

func MissionTypeToProto(missionType MissionType) (pb.MissionType, error) {
	switch missionType {
	case MissionTypePumpWater:
		return pb.MissionType_MISSION_TYPE_PUMP_WATER, nil
	case MissionTypeStabilizeVibration:
		return pb.MissionType_MISSION_TYPE_STABILIZE_VIBRATION, nil
	case MissionTypeRescuePersons:
		return pb.MissionType_MISSION_TYPE_RESCUE_PERSONS, nil
	default:
		return pb.MissionType_MISSION_TYPE_STABILIZE_VIBRATION, fmt.Errorf("unknown mission type")
	}
}

func MissionPriorityToDomain(missionPriority pb.MissionPriority) (MissionPriority, error) {
	switch missionPriority {
	case proto.MissionPriority_MISSION_PRIORITY_HIGH:
		return MissionPriorityHigh, nil
	case proto.MissionPriority_MISSION_PRIORITY_LOW:
		return MissionPriorityLow, nil
	case proto.MissionPriority_MISSION_PRIORITY_MEDIUM:
		return MissionPriorityMedium, nil
	default:
		return MissionPriorityLow, fmt.Errorf("unknown mission priority")
	}
}

func MissionPriorityToProto(missionPriority MissionPriority) (pb.MissionPriority, error) {
	switch missionPriority {

	case MissionPriorityHigh:
		return pb.MissionPriority_MISSION_PRIORITY_HIGH, nil
	case MissionPriorityLow:
		return pb.MissionPriority_MISSION_PRIORITY_LOW, nil
	case MissionPriorityMedium:
		return pb.MissionPriority_MISSION_PRIORITY_MEDIUM, nil
	default:
		return pb.MissionPriority_MISSION_PRIORITY_MEDIUM, fmt.Errorf("unknown mission priority")
	}

}

func MissionWithinBounds(position *pb.Position) error {
	if position.GetX() > uint32(types.WORLD_WIDTH) || position.GetX() < 0 {
		return fmt.Errorf("mission position outside of world bounds")
	}
	if position.GetY() > uint32(types.WORLD_WIDTH) || position.GetY() < 0 {
		return fmt.Errorf("mission position outside of world bounds")
	}
	return nil
}

type MissionState int

const (
	MissionStateIdle MissionState = iota
	MissionStateAssigned
	MissionStateOnTheWay
	MissionStateBusy
	MissionStateDone
	MissionStateError
)

func MissionStateToProto(actorState MissionState) (pb.ActorState, error) {
	switch actorState {
	case MissionStateAssigned:
		return pb.ActorState_ACTOR_STATE_ASSIGNED, nil
	case MissionStateBusy:
		return pb.ActorState_ACTOR_STATE_BUSY, nil
	case MissionStateDone:
		return pb.ActorState_ACTOR_STATE_DONE, nil
	case MissionStateError:
		return pb.ActorState_ACTOR_STATE_ERROR, nil
	case MissionStateIdle:
		return pb.ActorState_ACTOR_STATE_IDLE, nil
	case MissionStateOnTheWay:
		return pb.ActorState_ACTOR_STATE_ON_THE_WAY, nil
	default:
		return pb.ActorState_ACTOR_STATE_ASSIGNED, fmt.Errorf("unknown actor state")
	}
}
