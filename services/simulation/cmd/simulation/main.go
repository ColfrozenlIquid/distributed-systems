package main

import (
	"context"
	"log"
	"simulation/internal/config"
	"simulation/internal/transport"
	"simulation/internal/world"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	world := world.NewWorld(20, 20, *cfg)

	logger, err := zap.NewDevelopment()
	if err != nil {
		logger.Fatal("failed to initialize zap logger")
	}

	grpcSrv, err := transport.NewGRPCServer(world, logger)

	if err != nil {
		logger.Fatal("initialize grpc server", zap.Error(err))
	}

	go func() {
		logger.Info("grpc server running...", zap.String("address", "0.0.0.0:50051"))
		if err := grpcSrv.Serve(); err != nil {
			logger.Fatal("gRPC serve error", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	world.RunWorldLoop(ctx)
}
