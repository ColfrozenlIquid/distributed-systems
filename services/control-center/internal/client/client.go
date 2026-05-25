package client

import (
	pb "control-center/gen"
	"control-center/internal/config"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	WorldClient WorldClient
	Logger      *zap.Logger
}

type WorldClient struct {
	grpcConn           *grpc.ClientConn
	WorldServiceClient pb.WorldServiceClient
}

func (c *WorldClient) Close() error {
	return c.grpcConn.Close()
}

func NewClient(cfg config.ControlCenterConfig, logger *zap.Logger) (*Client, error) {
	conn, err := grpc.NewClient(cfg.WorldSimulationHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	worldClient := &WorldClient{
		grpcConn:           conn,
		WorldServiceClient: pb.NewWorldServiceClient(conn),
	}

	return &Client{
		WorldClient: *worldClient,
		Logger:      logger,
	}, err
}
