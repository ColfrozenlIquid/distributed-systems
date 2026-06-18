package model

import (
	pb "control-center/gen"
	"shared"
	"time"
)

type Actor struct {
	Id        string            `json:"id"`
	Type      shared.ActorType  `json:"type"`
	Position  shared.Position   `json:"position"`
	State     shared.ActorState `json:"state"`
	Reason    string            `json:"reason,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func ActorFromActorInfo(ai shared.ActorInfo) Actor {
	return Actor{
		Id:        ai.Id,
		Type:      ai.ActorType,
		Position:  ai.Position,
		State:     shared.ActorStateIdle,
		UpdatedAt: time.Now(),
	}
}

func (a Actor) UpdateWithStatus(status shared.ActorStatus) Actor {
	a.State = status.State
	a.Reason = status.Reason
	a.UpdatedAt = status.Timestamp
	return a
}

func (a Actor) UpdateWithTelemetry(telemetry shared.ActorTelemetry) Actor {
	a.Position = telemetry.Position
	a.UpdatedAt = telemetry.Timestamp
	return a
}

type Mission struct {
	Id        string                `json:"id"`
	ActorId   string                `json:"actor_id"`
	Incident  shared.SensorIncident `json:"incident"`
	State     shared.MissionState   `json:"state"`
	UpdatedAt time.Time             `json:"updated_at"`
}

func (m Mission) Update(update shared.MissionUpdate) Mission {
	m.State = update.State
	m.ActorId = update.ActorID
	m.UpdatedAt = update.Timestamp
	return m
}

func (m Mission) ToProtobufMission() *pb.Mission {
	return &pb.Mission{
		Id:       &pb.MissionId{Uuid: m.Id},
		Type:     missionTypeForIncident(m.Incident.Incident),
		Position: positionToProto(m.Incident.Sensor.Position),
		Priority: pb.MissionPriority_MISSION_PRIORITY_HIGH,
	}
}

func missionTypeForIncident(incident shared.Incident) pb.MissionType {
	switch incident {
	case shared.WaterLevelAlert:
		return pb.MissionType_MISSION_TYPE_PUMP_WATER
	case shared.VibrationTooHigh:
		return pb.MissionType_MISSION_TYPE_STABILIZE_VIBRATION
	case shared.PersonDetected:
		return pb.MissionType_MISSION_TYPE_RESCUE_PERSONS
	default:
		return pb.MissionType_MISSION_TYPE_UNSPECIFIED
	}
}

func positionToProto(position shared.Position) *pb.Position {
	return &pb.Position{
		X: position.X,
		Y: position.Y,
	}
}
