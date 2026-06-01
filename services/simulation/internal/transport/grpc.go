package transport

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	pb "simulation/gen"
	"simulation/internal/world"
)

var _ pb.WorldServiceServer = (*grpcServer)(nil)

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	logger   *zap.Logger
}

type grpcServer struct {
	pb.WorldServiceServer
	world  world.World
	logger *zap.Logger
}

func NewGRPCServer(world world.World, logger *zap.Logger) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		return nil, fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	srv := grpc.NewServer()
	reflection.Register(srv)

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

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
	return g.world.TileReading(ctx, req)
}

func (g *grpcServer) TileUpdate(ctx context.Context, req *pb.TileUpdateRequest) (*pb.TileUpdateResponse, error) {
	return g.world.TileUpdate(ctx, req)
}

func (g *grpcServer) WorldData(ctx context.Context, req *pb.WorldDataRequest) (*pb.WorldDataResponse, error) {
	return g.world.WorldData(ctx, req)
}

func (g *grpcServer) MoveTo(ctx context.Context, req *pb.MoveToRequest) (*pb.MoveToResponse, error) {
	return g.world.MoveTo(ctx, req)
}

func (g *grpcServer) PathTo(ctx context.Context, req *pb.PathToRequest) (*pb.PathToResponse, error) {
	return g.world.PathTo(ctx, req)
}
