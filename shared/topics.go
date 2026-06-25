package shared

import (
	"encoding/json"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type TopicSubscription string

func (s TopicSubscription) String() string {
	return string(s)
}

// Subscription-Patterns (with wildcards)
const (
	SubAllTileReadings   = "tiles/+/+/readings/#"
	SubAllActorStatus    = "actors/+/status"
	SubAllActorTelemetry = "actors/+/telemetry"
	SubAllMissions       = "missions/+/updates"
)

func TopicSubscriptionsWithQoSLevels() map[TopicSubscription]byte {
	return map[TopicSubscription]byte{
		SubAllTileReadings:   0,
		SubAllActorStatus:    1,
		SubAllActorTelemetry: 0,
		SubAllMissions:       1,
	}
}

type PublishParams struct {
	Topic    string
	QoS      byte
	Retained bool
	Payload  interface{}
}

func MqttPublish(client mqtt.Client, params PublishParams) error {
	payload, err := json.Marshal(params.Payload)
	if err != nil {
		return err
	}
	token := client.Publish(params.Topic, params.QoS, params.Retained, payload)
	token.Wait()
	return token.Error()
}

func TileReadingParams(x, y uint32, sensorType SensorType, payload interface{}) PublishParams {
	return PublishParams{
		Topic:    fmt.Sprintf("tiles/%d/%d/readings/%s", x, y, sensorType),
		QoS:      0,
		Retained: false,
		Payload:  payload,
	}
}

func ActorTelemetryParams(actorID string, payload interface{}) PublishParams {
	return PublishParams{
		Topic:    fmt.Sprintf("actors/%s/telemetry", actorID),
		QoS:      0,
		Retained: false,
		Payload:  payload,
	}
}

func ActorStatusParams(actorID string, payload interface{}) PublishParams {
	return PublishParams{
		Topic:    fmt.Sprintf("actors/%s/status", actorID),
		QoS:      1,
		Retained: true,
		Payload:  payload,
	}
}

func MissionUpdateParams(missionID string, payload interface{}) PublishParams {
	return PublishParams{
		Topic:    fmt.Sprintf("missions/%s/updates", missionID),
		QoS:      1,
		Retained: false,
		Payload:  payload,
	}
}
