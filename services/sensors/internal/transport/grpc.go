package transport

import (
	"fmt"
	pb "sensors/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Handle GRPC transport layer
type GRPCClient struct {
	grpcConn *grpc.ClientConn
	Grpc     pb.WorldServiceClient
}

func NewGRPCClient(cfg TransportConfig) (*GRPCClient, error) {
	conn, err := grpc.NewClient(cfg.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}
	return &GRPCClient{
		grpcConn: conn,
		Grpc:     pb.NewWorldServiceClient(conn),
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.grpcConn.Close()
}
