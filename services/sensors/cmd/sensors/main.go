package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sensors/internal/config"
	"sensors/internal/sensor"
	"sensors/internal/transport"
	"shared"
	"time"

	"go.uber.org/zap"
)

type App struct {
	logger    *zap.Logger
	transport transport.Transport
	sensor    sensor.Sensor
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

	clientCfg := transport.TransportConfig{
		HttpAddress: sensorCfg.BaseStationEndpoint,
		GrpcAddress: sensorCfg.WorldSimulationEndpoint,
		MQTTConfig:  sensorCfg.MQTTConfig,
		ClientId:    sensorCfg.Id,
	}

	transport, err := transport.NewTransport(clientCfg, logger)
	defer transport.Grpc.Close()

	sensor, err := sensor.NewSensor(*sensorCfg, transport.Grpc.Grpc)
	if err != nil {
		logger.Fatal("failed to create new sensor", zap.Error(err))
	}

	app := App{
		logger:    logger,
		transport: *transport,
		sensor:    sensor,
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

	resp, err := s.transport.Http.Post("/register/sensor", payload)
	if err != nil {
		return fmt.Errorf("post register sensor: %w", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		s.logger.Error("registration failed", zap.Any("response", resp))
	}

	return nil
}

func (s *App) Reading(reading float32) error {
	sensorReading := shared.SensorReading{
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
	payload, _ := json.Marshal(sensorReading)

	// Replaced with MQTT Publish
	// resp, err := s.client.Http.Post("/reading", payload)
	// if err != nil {
	// 	return fmt.Errorf("post sensor reading: %w", err)
	// }

	// if resp.StatusCode != http.StatusNoContent {
	// 	s.logger.Error("sensor reading failed", zap.Any("response", resp))
	// }

	// Retain = false, QoS = 0
	token := s.transport.Mqtt.MQTT.Publish(
		shared.TileReadingTopic(s.sensor.Position().X, s.sensor.Position().Y, shared.SensorType(s.sensor.Type())),
		byte(0),
		false,
		payload,
	)
	token.Wait()
	return token.Error()
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
