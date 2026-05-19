package main

import (
	"simulation/internal/client"
	"simulation/internal/world"

	"go.uber.org/zap"
)

func main() {
	world := world.NewWorld(20, 20)

	logger, err := zap.NewDevelopment()
	if err != nil {
		logger.Fatal("failed to initialize zap logger")
	}

	grpcSrv, err := client.NewGRPCServer(&world, logger)

	if err != nil {
		logger.Fatal("initialize grpc server", zap.Error(err))
	}

	go func() {
		logger.Info("grpc server running...", zap.String("address", "0.0.0.0:50051"))
		if err := grpcSrv.Serve(); err != nil {
			logger.Fatal("gRPC serve error", zap.Error(err))
		}
	}()

	for {
		world.RunWorldLoop()
	}
}
