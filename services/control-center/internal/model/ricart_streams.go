package model

import (
	"encoding/json"
	"log/slog"

	"shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/samber/lo"
	"github.com/samber/ro"
)

type RicartStreams struct {
	presences ro.Subject[shared.RicartPresence]
	requests  ro.Subject[shared.RicartRequest]
	replies   ro.Subject[shared.RicartReply]
	csEnter   ro.Subject[shared.RicartCsEvent]
	csExit    ro.Subject[shared.RicartCsEvent]
}

func NewRicartStreams() *RicartStreams {
	return &RicartStreams{
		presences: ro.NewPublishSubject[shared.RicartPresence](),
		requests:  ro.NewPublishSubject[shared.RicartRequest](),
		replies:   ro.NewPublishSubject[shared.RicartReply](),
		csEnter:   ro.NewPublishSubject[shared.RicartCsEvent](),
		csExit:    ro.NewPublishSubject[shared.RicartCsEvent](),
	}
}

func (s *RicartStreams) Presences() ro.Observable[shared.RicartPresence] {
	return s.presences
}

func (s *RicartStreams) Requests() ro.Observable[shared.RicartRequest] {
	return s.requests
}

func (s *RicartStreams) Replies() ro.Observable[shared.RicartReply] {
	return s.replies
}

func (s *RicartStreams) CsEnter() ro.Observable[shared.RicartCsEvent] {
	return s.csEnter
}

func (s *RicartStreams) CsExit() ro.Observable[shared.RicartCsEvent] {
	return s.csExit
}

func (s *RicartStreams) Subscribe(client mqtt.Client) error {
	topics := shared.RicartSubscriptionsWithQoSLevels()
	filters := lo.MapKeys(topics, func(_ byte, k shared.TopicSubscription) string {
		return k.String()
	})
	if token := client.SubscribeMultiple(filters, nil); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	addRicartRoute(client, shared.SubAllRicartPresence, s.presences)
	addRicartRoute(client, shared.SubAllRicartRequest, s.requests)
	addRicartRoute(client, shared.SubAllRicartReply, s.replies)
	addRicartRoute(client, shared.SubAllRicartCsEnter, s.csEnter)
	addRicartRoute(client, shared.SubAllRicartCsExit, s.csExit)

	return nil
}

func addRicartRoute[T any](client mqtt.Client, topic string, subject ro.Subject[T]) {
	client.AddRoute(topic, func(_ mqtt.Client, msg mqtt.Message) {
		var payload T
		if err := json.Unmarshal(msg.Payload(), &payload); err != nil {
			slog.Warn("malformed ricart payload", "topic", msg.Topic(), "err", err)
			return
		}
		subject.Next(payload)
	})
}

func (s *RicartStreams) Unsubscribe(client mqtt.Client) {
	client.Unsubscribe(
		shared.SubAllRicartPresence,
		shared.SubAllRicartRequest,
		shared.SubAllRicartReply,
		shared.SubAllRicartCsEnter,
		shared.SubAllRicartCsExit,
	)
}
