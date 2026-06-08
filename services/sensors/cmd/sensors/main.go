package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sensors/internal/client"
	"sensors/internal/config"
	"sensors/internal/sensor"
	"shared"
	"time"

	"go.uber.org/zap"
)

type App struct {
	logger *zap.Logger
	client client.Client
	sensor sensor.Sensor
}

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
	defer client.Grpc.Close()

	sensor, err := sensor.NewSensor(*sensorCfg, client.Grpc.Grpc)
	if err != nil {
		logger.Fatal("failed to create new sensor", zap.Error(err))
	}

	app := App{
		logger: logger,
		client: *client,
		sensor: sensor,
	}

	err = app.Register()
	if err != nil {
		logger.Fatal("failed to register sensor")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app.Run(ctx, sensor, clientCfg.HttpAddress)
}

func (s *App) Register() error {
	payload := shared.SensorRegistration{
		Sensor: shared.SensorInfo{
			Id: s.sensor.ID(),
			Position: shared.Position{
				X: s.sensor.Position().X,
				Y: s.sensor.Position().Y,
			},
			SensorType: shared.SensorType(s.sensor.Type()),
		},
	}

	resp, err := s.client.Http.Post("/register/sensor", payload)
	if err != nil {
		return fmt.Errorf("post register sensor: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		s.logger.Error("registration failed", zap.Any("response", resp))
	}

	return nil
}

func (s *App) Reading(reading float32) error {
	payload := shared.SensorReading{
		Timestamp: time.Now(),
		Sensor: shared.SensorInfo{
			Id: s.sensor.ID(),
			Position: shared.Position{
				X: s.sensor.Position().X,
				Y: s.sensor.Position().Y,
			},
			SensorType: shared.SensorType(s.sensor.Type()),
		},
		Value: reading,
	}

	resp, err := s.client.Http.Post("/reading", payload)
	if err != nil {
		return fmt.Errorf("post sensor reading: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		s.logger.Error("sensor reading failed", zap.Any("response", resp))
	}

	return nil
}

func (s *App) Run(ctx context.Context, sensor sensor.Sensor, endpoint string) {
	ticker := time.NewTicker(sensor.Interval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := sensor.Update()
			if err != nil {
				log.Printf("update failed")
			}
			reading, err := sensor.Read()
			if err != nil {
				log.Printf("[%s] read error: %v", sensor.ID(), err)
				err = s.Reading(0.0)
				continue
			}
			err = s.Reading(reading.Value)
			if err != nil {
				s.logger.Error("failed to send sensor reading", zap.Error(err))
			}
		}
	}
}
