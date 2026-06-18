package transport

import (
	"fmt"
	"shared"
	"time"

	"go.uber.org/zap"
)

type TransportConfig struct {
	ClientId    string
	HttpAddress string
	HttpTimeout time.Duration

	GrpcAddress string
	MQTTConfig  *shared.MqttConfig
}

type Transport struct {
	Http   HTTPClient
	Grpc   GRPCClient
	Logger *zap.Logger
}

func NewClient(cfg TransportConfig, logger *zap.Logger) (*Transport, error) {
	grpc, err := NewGRPCClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("create grpc client: %w", err)
	}

	http := NewHTTPClient(cfg)

	return &Transport{
		Logger: logger,
		Http:   *http,
		Grpc:   *grpc,
	}, nil
}
