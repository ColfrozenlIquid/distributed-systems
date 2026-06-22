package model

import (
	"context"
	pb "control-center/gen"
	"control-center/internal/mqttstream"
	"fmt"
	"log/slog"
	"shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/samber/ro"
)

type State struct {
	system      SystemStatus
	world       World
	mqttStreams *mqttstream.Streams
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
		mqttStreams: mqttstream.NewStreams(),
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

	s.mqttStreams.SensorReadings().SubscribeWithContext(ctx, ro.NewObserver(
		func(r shared.SensorReading) { state.System().AddReading(r) },
		func(err error) { slog.ErrorContext(ctx, "error in sensor readings MQTT stream") },
		func() {}))

	// TODO Subscribe to all other streams/topics & handle state accordingly
}
