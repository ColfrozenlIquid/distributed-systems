package shared

import "time"

type RicartActorState string

const (
	RicartActorStateIdle    RicartActorState = "IDLE"
	RicartActorStateWanting RicartActorState = "WANTING"
	RicartActorStateInCS    RicartActorState = "IN_CS"
)

// RicartPresence — published on ricart/presence/{actor_id}
// QoS 1, retain=true
type RicartPresence struct {
	ActorID   string    `json:"actor_id"`
	ActorType ActorType `json:"actor_type"`
	Timestamp time.Time `json:"timestamp"`
	Online    bool      `json:"online"` // false = graceful leave
}

// RicartRequest — published on ricart/request
// QoS 2, retain=false
type RicartRequest struct {
	FromID       string    `json:"from_id"`
	ResourceID   string    `json:"resource_id"` // z.B. "ladestation_nord"
	LogicalClock uint64    `json:"logical_clock"`
	Timestamp    time.Time `json:"timestamp"`
}

// RicartReply — published on ricart/reply/{target_id}
// QoS 2, retain=false
type RicartReply struct {
	FromID       string `json:"from_id"`
	ToID         string `json:"to_id"`
	ResourceID   string `json:"resource_id"`
	LogicalClock uint64 `json:"logical_clock"`
}

// RicartCsEvent — published on ricart/cs/enter & ricart/cs/exit
// QoS 1, retain=false — only for observability by control-center
type RicartCsEvent struct {
	ActorID      string    `json:"actor_id"`
	ResourceID   string    `json:"resource_id"`
	LogicalClock uint64    `json:"logical_clock"`
	Timestamp    time.Time `json:"timestamp"`
}
