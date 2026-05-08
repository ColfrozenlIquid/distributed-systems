package main

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "simulation/gen/proto"
	"simulation/sensors"
	"simulation/world"
)

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	logger   *zap.Logger
}

type grpcServer struct {
	pb.UnimplementedWorldServiceServer
	world  *world.World
	logger *zap.Logger
}

func NewGRPCServer(world *world.World, logger *zap.Logger) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", "localhost:50051")
	if err != nil {
		return nil, fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	srv := grpc.NewServer()
	reflection.Register(srv)

	handler := &grpcServer{
		world:  world,
		logger: logger,
	}

	pb.RegisterWorldServiceServer(srv, handler)

	return &GRPCServer{
		server:   srv,
		listener: lis,
		logger:   logger,
	}, nil
}

func (g *GRPCServer) Serve() error {
	return g.server.Serve(g.listener)
}

func (g *grpcServer) TileReading(ctx context.Context, req *pb.TileReadingRequest) (*pb.TileReadingResponse, error) {
	tile := g.world.GetTile(int(req.GetXPos()), int(req.GetYPos()))

	sensorType, err := sensors.ToDomainSensorType(req.GetSensorType())
	if err != nil {
		return nil, fmt.Errorf("unknown sensor type: %w", err)
	}

	tileReading, ok := tile.Read(sensorType)
	if !ok {
		return nil, fmt.Errorf("tile does not have this sensor")
	}

	return &pb.TileReadingResponse{
		SensorType:    req.GetSensorType(),
		SensorReading: tileReading,
	}, nil
}

func (g *grpcServer) TileUpdate(ctx context.Context, in *pb.TileUpdateRequest) (*pb.TileUpdateResponse, error) {
	panic("unimplemented")
}
