package mission

import (
	pb "actors/gen"
	proto "actors/gen"
	"actors/internal/actor"
	"actors/internal/transport"
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
	simulation pb.WorldServiceClient,
	actor actor.Actor,
	mqtt *transport.MQTTClient,
) (Mission, error) {
	missionType, err := MissionTypeToDomain(req.GetMission().GetType())
	if err != nil {
		return nil, fmt.Errorf("unknown mission type: %w", err)
	}
	missionPriority, err := MissionPriorityToDomain(req.GetMission().GetPriority())
	if err != nil {
		return nil, fmt.Errorf("unknown mission priority: %w", err)
	}

	cfg := MissionConfig{
		Id:   req.Mission.GetId().GetUuid(),
		Type: missionType,
		Position: shared.Position{
			X: req.GetMission().GetPosition().GetX(),
			Y: req.GetMission().GetPosition().GetY(),
		},
		Priority: missionPriority,
	}
	base := NewBaseMission(cfg, simulation, actor, mqtt)

	switch missionType {
	case MissionTypePumpWater:
		return &PumpWaterMission{base}, nil
	case MissionTypeRescuePersons:
		return &RescuePersonsMission{base}, nil
	case MissionTypeStabilizeVibration:
		return &StabilizeVibrationMission{base}, nil
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
