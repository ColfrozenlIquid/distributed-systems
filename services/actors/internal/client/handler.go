package client

import (
	pb "actors/gen"
	"actors/internal/actor"
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var _ pb.ActorServiceServer = (*ActorHandler)(nil)

type ActorHandler struct {
	pb.UnimplementedActorServiceServer
	logger *zap.Logger
	actor  actor.Actor
}

func (a *ActorHandler) AssignMission(req *pb.AssignMissionRequest, stream grpc.ServerStreamingServer[pb.AssignMissionResponse]) error {
	resultsCh := make(chan *pb.AssignMissionResponse)

	go a.actor.RegisterMission(stream.Context(), req, resultsCh)

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
