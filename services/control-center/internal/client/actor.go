package client

import (
	"context"
	pb "control-center/gen"
	"control-center/internal/model"
	"errors"
	"fmt"
	"net"
	"strings"

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

func (c *ActorClient) AssignMission(ctx context.Context, mission model.Mission) error {
	host := strings.TrimSpace(mission.ActorId)
	if host == "" {
		return errors.New("actor id is required for gRPC dialing")
	}

	conn, err := grpc.NewClient(net.JoinHostPort(host, c.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connect to actor %q over gRPC: %w", mission.ActorId, err)
	}

	response, err := pb.NewActorServiceClient(conn).AssignMission(ctx, &pb.AssignMissionRequest{Mission: mission.ToProtobufMission()})
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("assign mission over gRPC: %w", err)
	}
	if !response.GetSuccess() {
		_ = conn.Close()
		return fmt.Errorf("mission rejected by actor %q", mission.ActorId)
	}
	return nil
}
