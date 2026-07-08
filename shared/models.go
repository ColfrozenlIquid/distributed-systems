package shared

import "time"

type ActorState string

const (
	ActorStateIdle       ActorState = "idle"
	ActorStateOnAMission ActorState = "on_a_mission"
	ActorStateRefueling  ActorState = "refueling"
	ActorStateError      ActorState = "error"
	ActorStateOffline    ActorState = "offline"
)

type MissionState string

const (
	MissionStateAssigned  MissionState = "assigned"
	MissionStateOnTheWay  MissionState = "on_the_way"
	MissionStateBusy      MissionState = "busy"
	MissionStateParked    MissionState = "parked"
	MissionStateCompleted MissionState = "completed"
	MissionStateFailed    MissionState = "failed"
)

type ActorTelemetry struct {
	ActorID   string    `json:"actor_id"`
	Position  Position  `json:"position"`
	Timestamp time.Time `json:"timestamp"`
}

type ActorStatus struct {
	ActorID   string     `json:"actor_id"`
	State     ActorState `json:"state"`
	Reason    string     `json:"reason,omitempty"` // "lwt" bei unerwartetem Disconnect
	Timestamp time.Time  `json:"timestamp"`
}

type MissionUpdate struct {
	MissionID string       `json:"mission_id"`
	ActorID   string       `json:"actor_id"`
	State     MissionState `json:"state"`
	Timestamp time.Time    `json:"timestamp"`
}
