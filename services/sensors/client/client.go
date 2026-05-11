package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sensors/types"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "sensors/gen/proto"
)

type ClientConfig struct {
	HttpAddress string
	GrpcAddress string
}

type Client struct {
	Http     *http.Client
	grpcConn *grpc.ClientConn
	Grpc     pb.WorldServiceClient
	Logger   *zap.Logger
}

func (c *Client) Close() error {
	return c.grpcConn.Close()
}

func (c *Client) SendReading(endpoint string, reading types.Reading) error {
	data, err := json.Marshal(reading)
	if err != nil {
		return fmt.Errorf("failed to marshal reading: %w", err)
	}
	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send reading %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func NewClient(cfg ClientConfig, logger *zap.Logger) (*Client, error) {
	conn, err := grpc.NewClient(cfg.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	return &Client{
		Http:     &http.Client{},
		grpcConn: conn,
		Grpc:     pb.NewWorldServiceClient(conn),
		Logger:   logger,
	}, nil
}
