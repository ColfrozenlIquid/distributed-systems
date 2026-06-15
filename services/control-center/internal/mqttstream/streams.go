package mqttstream

import (
	"encoding/json"
	"log/slog"
	"shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/samber/lo"
	"github.com/samber/ro"
)

type Streams struct {
	sensorReadings   ro.Subject[shared.SensorReading]
	actorStatus      ro.Subject[shared.ActorStatus]
	actorTelemetries ro.Subject[shared.ActorTelemetry]
	missionUpdates   ro.Subject[shared.MissionUpdate]
}

func NewStreams() *Streams {
	return &Streams{
		sensorReadings:   ro.NewPublishSubject[shared.SensorReading](),
		actorStatus:      ro.NewPublishSubject[shared.ActorStatus](),
		actorTelemetries: ro.NewPublishSubject[shared.ActorTelemetry](),
		missionUpdates:   ro.NewPublishSubject[shared.MissionUpdate](),
	}
}

func (s *Streams) SensorReadings() ro.Observable[shared.SensorReading] {
	return s.sensorReadings
}

func (s *Streams) ActorStatus() ro.Observable[shared.ActorStatus] {
	return s.actorStatus
}

func (s *Streams) ActorTelemetries() ro.Observable[shared.ActorTelemetry] {
	return s.actorTelemetries
}

func (s *Streams) MissionUpdates() ro.Observable[shared.MissionUpdate] {
	return s.missionUpdates
}

func (s *Streams) Subscribe(client mqtt.Client) error {
	topics := shared.TopicSubscriptionsWithQoSLevels()
	filters := lo.MapKeys(topics, func(_ byte, k shared.TopicSubscription) string {
		return k.String()
	})

	token := client.SubscribeMultiple(filters, nil) // nil = DefaultPublishHandler als Fallback
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}

	addRoute(client, shared.SubAllTileReadings, s.sensorReadings)
	addRoute(client, shared.SubAllActorStatus, s.actorStatus)
	addRoute(client, shared.SubAllActorTelemetry, s.actorTelemetries)
	addRoute(client, shared.SubAllMissions, s.missionUpdates)

	return nil
}

func addRoute[T any](client mqtt.Client, topic string, subject ro.Subject[T]) {
	client.AddRoute(topic, func(_ mqtt.Client, msg mqtt.Message) {
		var payload T
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			slog.Warn("malformed payload", "topic", msg.Topic(), "err", err)
			return
		}
		subject.Next(payload)
	})
}
