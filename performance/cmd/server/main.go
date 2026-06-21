package main

import (
	"context"
	"encoding/json"
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
	"performance/internal"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// numMessages must match the client constant.
const numMessages = 10_000

// serverConfig holds every address/port the server binds to or advertises.
// All fields are overridable via environment variables (and a .env file in
// the working directory) by name, e.g. GRPC_ADDR, HTTP_ADDR.
type serverConfig struct {
	GRPCAddr string `mapstructure:"GRPC_ADDR"`
	HTTPAddr string `mapstructure:"HTTP_ADDR"`
}

// loadServerConfig reads ./.env (if present) plus the real environment,
// falling back to sane defaults when a key isn't set either place.
func loadServerConfig(v *viper.Viper) (*serverConfig, error) {
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	v.SetDefault("GRPC_ADDR", "0.0.0.0:50051")
	v.SetDefault("HTTP_ADDR", ":8080")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading .env: %w", err)
		}
		// no .env file present — that's fine, fall back to env vars/defaults
	}

	cfg := &serverConfig{
		GRPCAddr: v.GetString("GRPC_ADDR"),
		HTTPAddr: v.GetString("HTTP_ADDR"),
	}
	return cfg, nil
}

// Telemetry is the canonical in-memory representation of a telemetry event.
// Every protocol handler decodes its own wire format into this struct before
// handing it to the receiver — that decoding cost is included in the timing.
type Telemetry struct {
	ActorID   string
	PosX      int32
	PosY      int32
	Timestamp uint64
}

// wirePayload is the JSON envelope shared by HTTP and MQTT.
type wirePayload struct {
	ActorID   string `json:"actor_id"`
	PosX      int32  `json:"pos_x"`
	PosY      int32  `json:"pos_y"`
	Timestamp uint64 `json:"timestamp"`
	SentAt    uint64 `json:"sent_at"`
}

// receiver measures throughput for one protocol.
//
// The clock starts on the first decoded message and stops when the expected
// total arrives. Timing lives entirely on the receiver side so producer-side
// quirks (TCP batching, MQTT broker buffering, etc.) don't distort results.
type receiver struct {
	tag   string
	name  string
	total int64

	count     atomic.Int64
	startOnce sync.Once
	start     time.Time
}

func newReceiver(tag, name string) *receiver {
	return &receiver{tag: tag, name: name, total: numMessages}
}

// record is called once per fully-decoded message. It is safe for concurrent
// callers (the HTTP server dispatches handlers in goroutines).
func (r *receiver) record(msg Telemetry) {
	r.startOnce.Do(func() { r.start = time.Now() })

	n := r.count.Add(1)
	if n > r.total {
		return // stray message after benchmark completed, ignore
	}
	if n == r.total {
		elapsed := time.Since(r.start)
		log.Printf("[%s] received %d msgs in %v (%.0f msgs/sec)",
			r.name, r.total, elapsed, float64(r.total)/elapsed.Seconds())
		internal.SaveResult(r.tag, r.name, int(r.total), elapsed)
	}

	// msg is fully decoded and available here for any downstream processing.
	_ = msg
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("failed to initialize logger")
	}

	// --- config (.env + real environment) ---
	v := viper.New()
	v.AutomaticEnv()
	cfg, err := loadServerConfig(v)
	if err != nil {
		logger.Fatal("failed to load server config", zap.Error(err))
	}

	// --- gRPC (stream + unary share one server, separate receivers) ---
	streamRx := newReceiver("stream_grpc", "gRPC Stream")
	unaryRx := newReceiver("unary_grpc", "gRPC Unary")
	grpcSrv, err := newGRPCServer(cfg.GRPCAddr, streamRx, unaryRx)
	if err != nil {
		log.Fatalf("failed to create gRPC server: %v", err)
	}
	go func() {
		logger.Info("gRPC listening", zap.String("addr", cfg.GRPCAddr))
		if err := grpcSrv.serve(); err != nil {
			logger.Fatal("gRPC error", zap.Error(err))
		}
	}()

	// --- HTTP ---
	httpRx := newReceiver("http", "HTTP REST")
	router := vshttp.NewRouter()
	router.Post("/telemetry", func(c *vshttp.Context) error {
		var p wirePayload
		if err := c.JSONBody(&p); err != nil {
			return err
		}
		httpRx.record(Telemetry{
			ActorID: p.ActorID, PosX: p.PosX, PosY: p.PosY, Timestamp: p.Timestamp,
		})
		return c.NoContent()
	})
	httpServer, err := vshttp.Serve(cfg.HTTPAddr, router)
	if err != nil {
		log.Fatalf("failed to start HTTP server: %v", err)
	}
	defer httpServer.Close()
	logger.Info("HTTP listening", zap.String("addr", cfg.HTTPAddr))

	// --- MQTT ---
	mqttCfg, err := shared.LoadMqttConfig(v)
	if err != nil {
		logger.Fatal("failed to load MQTT config", zap.Error(err))
	}
	mqttRx := newReceiver("mqtt", "MQTT")
	startMQTTReceiver(logger, mqttCfg, mqttRx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logger.Info("shutting down")
}

// =============================================================================
// gRPC
// =============================================================================

type grpcHandler struct {
	pb.UnimplementedPerformanceServiceServer
	streamRx *receiver
	unaryRx  *receiver
}

type grpcServer struct {
	lis net.Listener
	srv *grpc.Server
}

func newGRPCServer(addr string, streamRx, unaryRx *receiver) (*grpcServer, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	srv := grpc.NewServer()
	reflection.Register(srv)
	pb.RegisterPerformanceServiceServer(srv, &grpcHandler{
		streamRx: streamRx,
		unaryRx:  unaryRx,
	})
	return &grpcServer{lis: lis, srv: srv}, nil
}

func (g *grpcServer) serve() error { return g.srv.Serve(g.lis) }

// ActorTelemetry handles unary RPCs. The protobuf layer has already decoded
// the wire bytes by the time we get here; we just reshape into Telemetry.
func (h *grpcHandler) ActorTelemetry(_ context.Context, in *pb.ActorTelemetryRequest) (*pb.ActorTelemetryResponse, error) {
	h.unaryRx.record(Telemetry{
		ActorID: in.ActorId, PosX: in.PosX, PosY: in.PosY, Timestamp: in.Timestamp,
	})
	return &pb.ActorTelemetryResponse{}, nil
}

// ActorTelemetryStream handles the client-streaming RPC. Messages arrive in
// order; we record each one as soon as it's decoded off the wire.
func (h *grpcHandler) ActorTelemetryStream(
	stream grpc.ClientStreamingServer[pb.ActorTelemetryStreamRequest, pb.ActorTelemetryStreamResponse],
) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			// All messages received. Unblock the client so it can proceed to
			// the next benchmark.
			return stream.SendAndClose(&pb.ActorTelemetryStreamResponse{})
		}
		if err != nil {
			return err
		}
		h.streamRx.record(Telemetry{
			ActorID: msg.ActorId, PosX: msg.PosX, PosY: msg.PosY, Timestamp: msg.Timestamp,
		})
	}
}

// =============================================================================
// MQTT
// =============================================================================

func startMQTTReceiver(logger *zap.Logger, cfg *shared.MqttConfig, rx *receiver) {
	client, err := shared.ConnectToMqttBroker(cfg, "perf-server", true)
	if err != nil {
		logger.Fatal("mqtt: connect failed", zap.Error(err))
	}

	token := client.Subscribe(shared.SubAllActorTelemetry, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p wirePayload
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			logger.Error("mqtt: unmarshal failed", zap.Error(err))
			return
		}

		rx.recordWithSentAt(Telemetry{
			ActorID:   p.ActorID,
			PosX:      p.PosX,
			PosY:      p.PosY,
			Timestamp: p.Timestamp,
		}, p.SentAt)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logger.Fatal("mqtt: subscribe failed", zap.Error(err))
	}
	logger.Info("MQTT subscribed", zap.String("topic", shared.SubAllActorTelemetry))
}

func (r *receiver) recordWithSentAt(msg Telemetry, sentAt uint64) {
	r.startOnce.Do(func() { r.start = time.Unix(0, int64(sentAt)) })

	n := r.count.Add(1)
	if n > r.total {
		return
	}
	if n == r.total {
		elapsed := time.Since(r.start)
		log.Printf("[%s] received %d msgs in %v (%.0f msgs/sec)",
			r.name, r.total, elapsed, float64(r.total)/elapsed.Seconds())
		internal.SaveResult(r.tag, r.name, int(r.total), elapsed)
	}

	_ = msg
}
