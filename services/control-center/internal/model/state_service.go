package model

import (
	"context"
	pb "control-center/gen"
	"fmt"
)

type State struct {
	system SystemStatus
	world  World
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
		system: SystemStatus{},
		world:  getWorld(worldClient),
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
