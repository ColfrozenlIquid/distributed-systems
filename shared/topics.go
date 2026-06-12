package shared

import "fmt"

// Topic-Templates
const (
	topicTileReading    = "tiles/%d/%d/readings/%s"
	topicActorStatus    = "actors/%s/status"
	topicActorTelemetry = "actors/%s/telemetry"
	topicMissionUpdate  = "missions/%s/updates"
)

// Subscription-Patterns (with wildcards)
const (
	SubAllTileReadings   = "tiles/+/+/readings/#"
	SubAllActorStatus    = "actors/+/status"
	SubAllActorTelemetry = "actors/+/telemetry"
	SubAllMissions       = "missions/+/updates"
)

// Builder-Functions for each topic

func TileReadingTopic(x, y uint32, sensorType SensorType) string {
	return fmt.Sprintf(topicTileReading, x, y, sensorType)
}

func ActorTelemetryTopic(actorID string) string {
	return fmt.Sprintf(topicActorTelemetry, actorID)
}

func ActorStatusTopic(actorID string) string {
	return fmt.Sprintf(topicActorStatus, actorID)
}

func MissionUpdateTopic(missionID string) string {
	return fmt.Sprintf(topicMissionUpdate, missionID)
}
