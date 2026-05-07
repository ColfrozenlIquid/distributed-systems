package main

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"world/gen/proto"
	pb "world/gen/proto"
)

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	logger   *zap.Logger
}

type grpcServer struct {
	pb.UnimplementedWorldServiceServer
	world  *World
	logger *zap.Logger
}

func NewGRPCServer(world *World, logger *zap.Logger) (*GRPCServer, error) {
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

	sensorType, err := ToDomainSensorType(req.GetSensorType())
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

func ToDomainSensorType(s pb.SensorType) (SensorType, error) {
	switch s {
	case proto.SensorType_SENSOR_TYPE_SUPPLY_LEVEL:
		return SensorSupplyLevel, nil
	case proto.SensorType_SENSOR_TYPE_VIBRATION:
		return SensorVibration, nil
	case proto.SensorType_SENSOR_TYPE_WATER_LEVEL:
		return SensorWaterLevel, nil
	default:
		return 0, fmt.Errorf("unknown or unspecified sensor type: %v", s)
	}
}

func ToProtoSensorType(s SensorType) (pb.SensorType, error) {
	switch s {
	case SensorSupplyLevel:
		return pb.SensorType_SENSOR_TYPE_SUPPLY_LEVEL, nil
	case SensorVibration:
		return pb.SensorType_SENSOR_TYPE_VIBRATION, nil
	case SensorWaterLevel:
		return pb.SensorType_SENSOR_TYPE_WATER_LEVEL, nil
	default:
		return pb.SensorType_SENSOR_TYPE_UNSPECIFIED, fmt.Errorf("unknown sensor type: %v", s)
	}
}
