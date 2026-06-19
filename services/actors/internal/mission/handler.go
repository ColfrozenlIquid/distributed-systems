package mission

import (
	pb "actors/gen"
	"actors/internal/actor"
	"actors/internal/transport"
	"context"
	"fmt"

	"go.uber.org/zap"
)

var _ pb.ActorServiceServer = (*ActorHandler)(nil)

type ActorHandler struct {
	pb.UnimplementedActorServiceServer
	simulation pb.WorldServiceClient
	mqtt       *transport.MQTTClient
	logger     *zap.Logger
	actor      actor.Actor
	mission    Mission
}

func NewActorHandler(logger *zap.Logger, actor actor.Actor, simulation pb.WorldServiceClient, mqtt *transport.MQTTClient) *ActorHandler {
	return &ActorHandler{
		logger:     logger,
		actor:      actor,
		simulation: simulation,
		mqtt:       mqtt,
	}
}

func (a *ActorHandler) AssignMission(ctx context.Context, req *pb.AssignMissionRequest) (*pb.AssignMissionResponse, error) {
	mission, err := NewMission(req, a.simulation, a.actor, a.mqtt)
	if err != nil {
		return &pb.AssignMissionResponse{
			Success: false,
		}, fmt.Errorf("create new mission: %w", err)
	}

	a.mission = mission

	go a.mission.RunMission(ctx)

	return &pb.AssignMissionResponse{
		Success: true,
	}, nil
}
