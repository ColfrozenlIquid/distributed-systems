package client

import (
	pb "control-center/gen"
	"control-center/internal/config"
	"fmt"
	"shared"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	WorldClient *WorldClient
	ActorClient *ActorClient
	MqttClient  mqtt.Client
	Logger      *zap.Logger
}

type WorldClient struct {
	grpcConn           *grpc.ClientConn
	WorldServiceClient pb.WorldServiceClient
}

func (c *WorldClient) Close() error {
	return c.grpcConn.Close()
}

func NewClient(cfg config.ControlCenterConfig, logger *zap.Logger) (*Client, error) {
	conn, err := grpc.NewClient(cfg.WorldSimulationHostAndPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	worldClient := &WorldClient{
		grpcConn:           conn,
		WorldServiceClient: pb.NewWorldServiceClient(conn),
	}

	mqttClient, err := shared.ConnectToMqttBroker(cfg, "control-center", false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	return &Client{
		WorldClient: worldClient,
		ActorClient: NewActorClient(),
		MqttClient:  mqttClient,
		Logger:      logger,
	}, err
}
