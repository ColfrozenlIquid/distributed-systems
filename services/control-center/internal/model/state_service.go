package model

import (
	"context"
	pb "control-center/gen"
	"fmt"
	"log/slog"
	"shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/samber/ro"
)

type State struct {
	system        SystemStatus
	world         World
	ricart        RicartStatus
	mqttStreams   *shared.Streams
	ricartStreams *RicartStreams
}

func (s *State) System() *SystemStatus {
	return &s.system
}

func (s *State) World() *World {
	return &s.world
}

func (s *State) Ricart() *RicartStatus {
	return &s.ricart
}

var state State

func NewState(worldClient pb.WorldServiceClient) *State {
	state = State{
		system:        NewSystemStatus(),
		world:         getWorld(worldClient),
		ricart:        NewRicartStatus(),
		mqttStreams:   shared.NewStreams(),
		ricartStreams: NewRicartStreams(),
	}
	return &state
}

func getWorld(worldClient pb.WorldServiceClient) World {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worldDataResponse, err := worldClient.WorldData(ctx, &pb.WorldDataRequest{})
	if err != nil {
		panic(fmt.Errorf("failed to get world data: %w", err))
	}
	return NewWorldFromWorldDataResponse(worldDataResponse)
}

func (s *State) StartReactiveMessageProcessing(ctx context.Context, client mqtt.Client) {
	err := s.mqttStreams.Subscribe(client)
	if err != nil {
		panic(fmt.Errorf("failed to subscribe to MQTT streams: %w", err))
	}
	defer s.mqttStreams.Unsubscribe(client)
	if err = s.ricartStreams.Subscribe(client); err != nil {
		panic(fmt.Errorf("failed to subscribe to Ricart MQTT streams: %w", err))
	}
	defer s.ricartStreams.Unsubscribe(client)
	subscriptions := ro.NewSubscription(nil)
	defer subscriptions.Unsubscribe()

	subscriptions.AddUnsubscribable(s.mqttStreams.SensorReadings().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, r shared.SensorReading) { s.System().AddReading(r) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in sensor readings MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.mqttStreams.ActorStatus().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, status shared.ActorStatus) { s.System().UpdateActorFromStatus(status) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in actor status MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.mqttStreams.ActorTelemetries().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, telemetry shared.ActorTelemetry) {
			s.System().UpdateActorFromTelemetry(telemetry)
		},
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in actor telemetry MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.mqttStreams.MissionUpdates().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, update shared.MissionUpdate) { s.System().UpdateMission(update) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in mission updates MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.ricartStreams.Presences().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, presence shared.RicartPresence) { s.Ricart().HandlePresence(presence) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in ricart presences MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.ricartStreams.Requests().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, request shared.RicartRequest) { s.Ricart().HandleRequest(request) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in ricart requests MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.ricartStreams.Replies().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, reply shared.RicartReply) { s.Ricart().HandleReply(reply) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in ricart replies MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.ricartStreams.CsEnter().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, event shared.RicartCsEvent) { s.Ricart().HandleCSEnter(event) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in ricart cs enter MQTT stream") },
		func(ctx context.Context) {})))

	subscriptions.AddUnsubscribable(s.ricartStreams.CsExit().SubscribeWithContext(ctx, ro.NewObserverWithContext(
		func(ctx context.Context, event shared.RicartCsEvent) { s.Ricart().HandleCSExit(event) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in ricart cs exit MQTT stream") },
		func(ctx context.Context) {})))

	<-ctx.Done()
}
