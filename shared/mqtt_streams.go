package shared

import (
	"encoding/json"
	"log/slog"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/samber/lo"
	"github.com/samber/ro"
)

type Streams struct {
	sensorReadings   ro.Subject[SensorReading]
	actorStatus      ro.Subject[ActorStatus]
	actorTelemetries ro.Subject[ActorTelemetry]
	missionUpdates   ro.Subject[MissionUpdate]
}

func NewStreams() *Streams {
	return &Streams{
		sensorReadings:   ro.NewPublishSubject[SensorReading](),
		actorStatus:      ro.NewPublishSubject[ActorStatus](),
		actorTelemetries: ro.NewPublishSubject[ActorTelemetry](),
		missionUpdates:   ro.NewPublishSubject[MissionUpdate](),
	}
}

func (s *Streams) SensorReadings() ro.Observable[SensorReading] {
	return s.sensorReadings
}

func (s *Streams) ActorStatus() ro.Observable[ActorStatus] {
	return s.actorStatus
}

func (s *Streams) ActorTelemetries() ro.Observable[ActorTelemetry] {
	return s.actorTelemetries
}

func (s *Streams) MissionUpdates() ro.Observable[MissionUpdate] {
	return s.missionUpdates
}

func (s *Streams) Subscribe(client mqtt.Client) error {
	topics := TopicSubscriptionsWithQoSLevels()
	filters := lo.MapKeys(topics, func(_ byte, k TopicSubscription) string {
		return k.String()
	})

	token := client.SubscribeMultiple(filters, nil) // nil = DefaultPublishHandler als Fallback
	token.Wait()
	if err := token.Error(); err != nil {
		return err
	}

	addRoute(client, SubAllTileReadings, s.sensorReadings)
	addRoute(client, SubAllActorStatus, s.actorStatus)
	addRoute(client, SubAllActorTelemetry, s.actorTelemetries)
	addRoute(client, SubAllMissions, s.missionUpdates)

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

func (s *Streams) Unsubscribe(client mqtt.Client) {
	client.Unsubscribe(SubAllTileReadings)
	client.Unsubscribe(SubAllActorStatus)
	client.Unsubscribe(SubAllActorTelemetry)
	client.Unsubscribe(SubAllMissions)
}
