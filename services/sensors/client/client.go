package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "sensors/gen/proto"
)

type ClientConfig struct {
	HttpAddress string
	HttpTimeout time.Duration

	GrpcAddress string
}

type Client struct {
	Http   HTTPClient
	Grpc   GRPCClient
	Logger *zap.Logger
}

type GRPCClient struct {
	grpcConn *grpc.ClientConn
	Grpc     pb.WorldServiceClient
}

type HTTPClient struct {
	httpClient *http.Client
	BaseURL    string
	Timeout    time.Duration
}

type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func (c *GRPCClient) Close() error {
	return c.grpcConn.Close()
}

// func (c *Client) SendReading(endpoint string, reading types.Reading) error {
// 	data, err := json.Marshal(reading)
// 	if err != nil {
// 		return fmt.Errorf("failed to marshal reading: %w", err)
// 	}
// 	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(data))
// 	if err != nil {
// 		return fmt.Errorf("failed to send reading %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
// 	}
// 	return nil
// }

func NewClient(cfg ClientConfig, logger *zap.Logger) (*Client, error) {
	conn, err := grpc.NewClient(cfg.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	grpc := &GRPCClient{
		grpcConn: conn,
		Grpc:     pb.NewWorldServiceClient(conn),
	}

	http := &HTTPClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		BaseURL:    cfg.HttpAddress,
		Timeout:    cfg.HttpTimeout,
	}

	return &Client{
		Logger: logger,
		Http:   *http,
		Grpc:   *grpc,
	}, nil
}

func (c *HTTPClient) send(method string, path string, body any) (*Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Header:     resp.Header,
	}, nil
}

func (c *HTTPClient) Post(path string, body any) (*Response, error) {
	return c.send(http.MethodPost, path, body)
}
