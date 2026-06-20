package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"shared"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"vshttp"

	pb "performance/gen"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type TelemetryPayload struct {
	ActorID   string `json:"actor_id"`
	PosX      int32  `json:"pos_x"`
	PosY      int32  `json:"pos_y"`
	Timestamp uint64 `json:"timestamp"`
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		logger.Fatal("failed to initialize zap logger")
	}

	router := vshttp.NewRouter()

	router.Post("/telemetry", func(c *vshttp.Context) error {
		var req TelemetryPayload
		if err := c.JSONBody(&req); err != nil {
			return err
		}
		return c.NoContent()
	})
	server, err := vshttp.Serve(":8080", router)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on :8080")

	grpcSrv, err := NewGRPCServer(logger)
	go func() {
		logger.Info("grpc server running...", zap.String("address", "0.0.0.0:50051"))
		if err := grpcSrv.Serve(); err != nil {
			logger.Fatal("gRPC serve error", zap.Error(err))
		}
	}()

	v := viper.New()
	v.AutomaticEnv()
	mqttCfg, err := shared.LoadMqttConfig(v)
	if err != nil {
		logger.Fatal("failed to load mqtt config", zap.Error(err))
	}
	startMqttThroughputCounter(logger, mqttCfg)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	logger   *zap.Logger
}

type grpcServer struct {
	pb.PerformanceServiceServer
	logger     *zap.Logger
	unaryCount uint64
	unaryStart time.Time
	unaryOnce  sync.Once
}

func NewGRPCServer(logger *zap.Logger) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		return nil, fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	srv := grpc.NewServer()
	reflection.Register(srv)

	handler := &grpcServer{
		logger: logger,
	}

	pb.RegisterPerformanceServiceServer(srv, handler)

	return &GRPCServer{
		server:   srv,
		listener: lis,
		logger:   logger,
	}, nil
}

func (g *GRPCServer) Serve() error {
	return g.server.Serve(g.listener)
}

func (s *grpcServer) ActorTelemetry(ctx context.Context, in *pb.ActorTelemetryRequest) (*pb.ActorTelemetryResponse, error) {
	s.unaryOnce.Do(func() { s.unaryStart = time.Now() })

	count := atomic.AddUint64(&s.unaryCount, 1)
	if count%1000 == 0 {
		elapsed := time.Since(s.unaryStart)
		s.logger.Debug("unary progress",
			zap.Uint64("count", count),
			zap.Float64("msgs_per_sec", float64(count)/elapsed.Seconds()),
		)
	}
	return &pb.ActorTelemetryResponse{}, nil
}

func (s *grpcServer) ActorTelemetryStream(stream grpc.ClientStreamingServer[pb.ActorTelemetryStreamRequest, pb.ActorTelemetryStreamResponse]) error {
	var count uint64
	start := time.Now()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			elapsed := time.Since(start)
			s.logger.Info("stream closed",
				zap.Uint64("messages_received", count),
				zap.Duration("elapsed", elapsed),
				zap.Float64("msgs_per_sec", float64(count)/elapsed.Seconds()),
			)
			return stream.SendAndClose(&pb.ActorTelemetryStreamResponse{})
		}
		if err != nil {
			return err
		}

		count++
		if count%1000 == 0 {
			s.logger.Debug("progress", zap.String("actor_id", msg.ActorId), zap.Uint64("count", count))
		}
	}
}

func startMqttThroughputCounter(logger *zap.Logger, cfg *shared.MqttConfig) {
	client, err := shared.ConnectToMqttBroker(cfg, "perf-server", true)
	if err != nil {
		logger.Fatal("mqtt connect failed", zap.Error(err))
	}

	var count uint64
	var start time.Time
	var once sync.Once

	token := client.Subscribe(shared.SubAllActorTelemetry, 0, func(_ mqtt.Client, msg mqtt.Message) {
		once.Do(func() { start = time.Now() })
		c := atomic.AddUint64(&count, 1)
		if c%1000 == 0 {
			elapsed := time.Since(start)
			logger.Debug("mqtt progress",
				zap.Uint64("count", c),
				zap.Float64("msgs_per_sec", float64(c)/elapsed.Seconds()),
			)
		}
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logger.Fatal("mqtt subscribe failed", zap.Error(err))
	}

	logger.Info("mqtt throughput counter subscribed", zap.String("topic", shared.SubAllActorTelemetry))
}
