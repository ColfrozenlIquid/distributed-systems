package client

import (
	pb "actors/gen"
	"actors/internal/actor"
	"actors/internal/mission"
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var _ pb.ActorServiceServer = (*ActorHandler)(nil)

type ActorHandler struct {
	pb.UnimplementedActorServiceServer
	simulation pb.WorldServiceClient
	logger     *zap.Logger
	actor      actor.Actor
	mission    mission.Mission
}

func (a *ActorHandler) AssignMission(req *pb.AssignMissionRequest, stream grpc.ServerStreamingServer[pb.AssignMissionResponse]) error {
	resultsCh := make(chan *pb.AssignMissionResponse)

	mission, err := mission.NewMission(req, resultsCh, a.simulation, a.actor)
	if err != nil {
		return fmt.Errorf("create new mission: %w", err)
	}

	a.mission = mission

	go a.mission.RunMission(stream.Context())

	for res := range resultsCh {
		if err := stream.Send(res); err != nil {
			return err
		}
	}

	return nil
}

func NewActorHandler(logger *zap.Logger) *ActorHandler {
	return &ActorHandler{
		logger: logger,
	}
}

func (a *ActorHandler) runMission(ctx context.Context, req *pb.AssignMissionRequest, out chan<- *pb.AssignMissionResponse) {
	defer close(out)

}
