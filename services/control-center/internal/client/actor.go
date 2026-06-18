package client

import (
	"context"
	"control-center/internal/model"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	pb "control-center/gen"
	"shared"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultActorGRPCPort = "50051"

type ActorClient struct {
	port string
}

func NewActorClient() *ActorClient {
	return &ActorClient{
		port: defaultActorGRPCPort,
	}
}

func (c *ActorClient) AssignMission(ctx context.Context, actorId string, pbMission *pb.Mission) (<-chan model.Update, error) {
	host := strings.TrimSpace(actorId)
	if host == "" {
		return nil, errors.New("actor id is required for gRPC dialing")
	}

	conn, err := grpc.NewClient(net.JoinHostPort(host, c.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to actor %q over gRPC: %w", actorId, err)
	}

	stream, err := pb.NewActorServiceClient(conn).AssignMission(ctx, &pb.AssignMissionRequest{Mission: pbMission})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("assign mission over gRPC: %w", err)
	}

	updates := make(chan model.Update)
	go func() {
		defer close(updates)
		defer conn.Close()
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				fmt.Printf("error while receiving stream response: %v\n", err)
				updates <- model.Update{
					MissionID: pbMission.GetId().GetUuid(),
					State:     actorStateLabel(pb.ActorState_ACTOR_STATE_UNSPECIFIED),
					Position:  shared.Position{},
					UpdatedAt: time.Now(),
				}
				return
			}

			position := response.GetCurrentPosition()
			updates <- model.Update{
				MissionID: response.GetMissionId().GetUuid(),
				State:     actorStateLabel(response.GetState()),
				Position: shared.Position{
					X: position.GetX(),
					Y: position.GetY(),
				},
				UpdatedAt: time.Now(),
			}
		}
	}()

	return updates, nil
}

func actorStateLabel(state pb.ActorState) string {
	switch state {
	case pb.ActorState_ACTOR_STATE_IDLE:
		return "idle"
	case pb.ActorState_ACTOR_STATE_ASSIGNED:
		return "assigned"
	case pb.ActorState_ACTOR_STATE_ON_THE_WAY:
		return "on the way"
	case pb.ActorState_ACTOR_STATE_BUSY:
		return "busy"
	case pb.ActorState_ACTOR_STATE_DONE:
		return "done"
	case pb.ActorState_ACTOR_STATE_ERROR:
		return "error"
	default:
		return "unspecified"
	}
}
