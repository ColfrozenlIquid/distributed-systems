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
	system      SystemStatus
	world       World
	mqttStreams *shared.Streams
}

func (s *State) System() *SystemStatus {
	return &s.system
}

func (s *State) World() *World {
	return &s.world
}

var state State

func NewState(worldClient pb.WorldServiceClient) *State {
	state = State{
		system:      SystemStatus{},
		world:       getWorld(worldClient),
		mqttStreams: shared.NewStreams(),
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
		func(ctx context.Context, update shared.MissionUpdate) { s.System().UpdateMissionNew(update) },
		func(ctx context.Context, err error) { slog.ErrorContext(ctx, "error in mission updates MQTT stream") },
		func(ctx context.Context) {})))

	<-ctx.Done()
}
