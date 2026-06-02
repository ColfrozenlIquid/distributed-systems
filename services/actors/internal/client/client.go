package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	pb "actors/gen"
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
	grpcConn           *grpc.ClientConn
	WorldServiceClient pb.WorldServiceClient
	Server             *grpc.Server
	Listener           net.Listener
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

func NewClient(cfg ClientConfig, logger *zap.Logger) (*Client, error) {
	conn, err := grpc.NewClient(cfg.GrpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}

	srv := grpc.NewServer()

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		logger.Fatal("failed to listen for gRPC", zap.Error(err))
		return nil, err
	}

	reflection.Register(srv)

	grpc := &GRPCClient{
		grpcConn:           conn,
		WorldServiceClient: pb.NewWorldServiceClient(conn),
		Server:             srv,
		Listener:           lis,
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

func (g *GRPCClient) RegisterActorHandler(handler pb.ActorServiceServer) {
	pb.RegisterActorServiceServer(g.Server, handler)
}

func (g *GRPCClient) Serve() error {
	return g.Server.Serve(g.Listener)
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
