package transport

import (
	"fmt"
	"shared"
	"time"

	"go.uber.org/zap"
)

type Transport struct {
	Http   HTTPClient
	Grpc   GRPCClient
	Mqtt   MQTTClient
	Logger *zap.Logger
}

type TransportConfig struct {
	ClientId    string
	HttpAddress string
	HttpTimeout time.Duration

	GrpcAddress string
	MQTTConfig  *shared.MqttConfig
}

func NewTransport(cfg TransportConfig, logger *zap.Logger) (*Transport, error) {
	grpc, err := NewGRPCClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("grpc client creation: %w", err)
	}

	http := NewHTTPClient(cfg)

	mqtt, err := NewMQTTClient(cfg)

	return &Transport{
		Logger: logger,
		Http:   *http,
		Grpc:   *grpc,
		Mqtt:   *mqtt,
	}, nil
}
