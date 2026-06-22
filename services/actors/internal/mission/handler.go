package mission

import (
	pb "actors/gen"
	"actors/internal/actor"
	"actors/internal/transport"
	"context"
	"fmt"

	"go.uber.org/zap"
)

var _ pb.ActorServiceServer = (*MissionHandler)(nil)

type MissionHandler struct {
	pb.UnimplementedActorServiceServer
	simulation pb.WorldServiceClient
	mqtt       *transport.MQTTClient
	logger     *zap.Logger
	actor      actor.Actor
	mission    Mission
}

func NewMissionHandler(logger *zap.Logger, actor actor.Actor, simulation pb.WorldServiceClient, mqtt *transport.MQTTClient) *MissionHandler {
	return &MissionHandler{
		logger:     logger,
		actor:      actor,
		simulation: simulation,
		mqtt:       mqtt,
	}
}

func (a *MissionHandler) AssignMission(ctx context.Context, req *pb.AssignMissionRequest) (*pb.AssignMissionResponse, error) {
	a.logger.Info("Calling assign mission")
	mission, err := NewMission(req, a.simulation, a.actor, a.mqtt, a.logger)
	if err != nil {
		return &pb.AssignMissionResponse{
			Success: false,
		}, fmt.Errorf("create new mission: %w", err)
	}

	a.mission = mission

	go a.mission.RunMission(context.Background())

	return &pb.AssignMissionResponse{
		Success: true,
	}, nil
}
