package mission

import (
	pb "actors/gen"
	proto "actors/gen"
	"actors/internal/types"
	"time"

	"fmt"
)

type Mission struct {
	Id           string
	Type         MissionType
	Position     types.Position
	Priority     MissionPriority
	State        ActorState
	Duration     time.Duration
	MissionTimer *time.Timer
}

type MissionType int

const (
	MissionTypePumpWater MissionType = iota
	MissionTypeStabilizeVibration
)

type MissionPriority int

const (
	MissionPriorityLow MissionPriority = iota
	MissionPriorityMedium
	MissionPriorityHigh
)

type ActorState int

const (
	ActorStateIdle ActorState = iota
	ActorStateAssigned
	ActorStateOnTheWay
	ActorStateBusy
	ActorStateDone
	ActorStateError
)

func ActorStateToProto(actorState ActorState) (pb.ActorState, error) {
	switch actorState {
	case ActorStateAssigned:
		return pb.ActorState_ACTOR_STATE_ASSIGNED, nil
	case ActorStateBusy:
		return pb.ActorState_ACTOR_STATE_BUSY, nil
	case ActorStateDone:
		return pb.ActorState_ACTOR_STATE_DONE, nil
	case ActorStateError:
		return pb.ActorState_ACTOR_STATE_ERROR, nil
	case ActorStateIdle:
		return pb.ActorState_ACTOR_STATE_IDLE, nil
	case ActorStateOnTheWay:
		return pb.ActorState_ACTOR_STATE_ON_THE_WAY, nil
	default:
		return pb.ActorState_ACTOR_STATE_ASSIGNED, fmt.Errorf("unknown actor state")
	}
}

func MissionTypeToDomain(missionType pb.MissionType) (MissionType, error) {
	switch missionType {
	case pb.MissionType_MISSION_TYPE_PUMP_WATER:
		return MissionTypePumpWater, nil
	case pb.MissionType_MISSION_TYPE_STABILIZE_VIBRATION:
		return MissionTypeStabilizeVibration, nil
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
