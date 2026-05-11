package main

import (
	"context"
	"log"
	"sensors/client"
	"sensors/config"
	"sensors/types"
	"time"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	sensorCfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("failed to read sensor config", zap.Error(err))
	}

	clientCfg := client.ClientConfig{
		HttpAddress: sensorCfg.BaseStationEndpoint,
		GrpcAddress: sensorCfg.WorldSimulationEndpoint,
	}

	client, err := client.NewClient(clientCfg, logger)
	defer client.Close()

	sensor, err := types.NewSensor(*sensorCfg, client.Grpc)
	if err != nil {
		logger.Fatal("failed to create new sensor", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	Run(ctx, sensor, clientCfg.HttpAddress)
}

func Run(ctx context.Context, s types.Sensor, endpoint string) {
	ticker := time.NewTicker(s.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := s.Read()
			if err != nil {
				log.Printf("[%s] read error: %v", s.ID(), err)
				continue
			}
		}
	}
}
