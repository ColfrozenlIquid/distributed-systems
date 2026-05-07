package main

import (
	"go.uber.org/zap"
)

func main() {
	world := NewWorld(20, 20)

	logger, err := zap.NewDevelopment()
	if err != nil {
		logger.Fatal("failed to initialize zap logger")
	}

	grpcSrv, err := NewGRPCServer(&world, logger)

	if err != nil {
		logger.Fatal("initializze grpc server", zap.Error(err))
	}

	go func() {
		logger.Info("grpc server running...", zap.String("address", "localhost:50051"))
		if err := grpcSrv.Serve(); err != nil {
			logger.Fatal("gRPC serve error", zap.Error(err))
		}
	}()

	for {
		world.RunWorldLoop()
	}
}
