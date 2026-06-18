package transport

import (
	pb "actors/gen"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type GRPCClient struct {
	grpcConn           *grpc.ClientConn
	WorldServiceClient pb.WorldServiceClient
	Server             *grpc.Server
	Listener           net.Listener
}

func NewGRPCClient(cfg TransportConfig, logger *zap.Logger) (*GRPCClient, error) {
	conn, err := grpc.NewClient(cfg.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		logger.Fatal("failed to listen for gRPC", zap.Error(err))
		return nil, err
	}

	srv := grpc.NewServer()

	reflection.Register(srv)

	return &GRPCClient{
		grpcConn:           conn,
		WorldServiceClient: pb.NewWorldServiceClient(conn),
		Server:             srv,
		Listener:           lis,
	}, nil
}

func (c *GRPCClient) Close() error {
	return c.grpcConn.Close()
}

func (g *GRPCClient) RegisterActorHandler(handler pb.ActorServiceServer) {
	pb.RegisterActorServiceServer(g.Server, handler)
}

func (g *GRPCClient) Serve() error {
	return g.Server.Serve(g.Listener)
}
