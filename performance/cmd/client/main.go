package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shared"
	"time"

	pb "performance/gen"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// numMessages must match the server constant so the receiver knows when it's done.
const numMessages = 5_000

// progressEvery controls how often (in messages sent) each run logs a
// progress update. Set to 0 to disable progress logging entirely.
const progressEvery = 500

// logProgress prints a periodic "N/total" update for a named run. i is the
// 0-based loop index, so we report on i+1 (i.e. after the 500th, 1000th, ...
// message has actually been sent), plus a final update on the very last
// message in case numMessages isn't a multiple of progressEvery.
func logProgress(name string, i, total int) {
	if progressEvery <= 0 {
		return
	}
	sent := i + 1
	if sent%progressEvery == 0 || sent == total {
		log.Printf("%s: progress %d/%d", name, sent, total)
	}
}

// clientConfig holds every address/URL the client dials or posts to. All
// fields are overridable via environment variables (and a .env file in the
// working directory) by name, e.g. GRPC_TARGET, HTTP_URL.
type clientConfig struct {
	GRPCTarget string `mapstructure:"GRPC_TARGET"`
	HTTPURL    string `mapstructure:"HTTP_URL"`
}

// loadClientConfig reads ./.env (if present) plus the real environment,
// falling back to sane defaults when a key isn't set either place.
func loadClientConfig(v *viper.Viper) (*clientConfig, error) {
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")

	v.SetDefault("GRPC_TARGET", "localhost:50051")
	v.SetDefault("HTTP_URL", "http://localhost:8080/telemetry")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading .env: %w", err)
		}
		// no .env file present — that's fine, fall back to env vars/defaults
	}

	cfg := &clientConfig{
		GRPCTarget: v.GetString("GRPC_TARGET"),
		HTTPURL:    v.GetString("HTTP_URL"),
	}
	return cfg, nil
}

// wirePayload is the JSON envelope used by HTTP and MQTT.
type wirePayload struct {
	ActorID   string `json:"actor_id"`
	PosX      int32  `json:"pos_x"`
	PosY      int32  `json:"pos_y"`
	Timestamp uint64 `json:"timestamp"`
	SentAt    uint64 `json:"sent_at"`
}

func main() {
	v := viper.New()
	v.AutomaticEnv()

	cfg, err := loadClientConfig(v)
	if err != nil {
		log.Fatalf("failed to load client config: %v", err)
	}

	mqttCfg, err := shared.LoadMqttConfig(v)
	if err != nil {
		log.Fatal("failed to load mqtt config", zap.Error(err))
	}

	conn, err := grpc.NewClient(cfg.GRPCTarget,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	grpcClient := pb.NewPerformanceServiceClient(conn)

	runStreamLoad(grpcClient)
	time.Sleep(500 * time.Millisecond) // let the server flush its result before the next run

	runUnaryLoad(grpcClient)
	time.Sleep(500 * time.Millisecond)

	runHTTPLoad(cfg.HTTPURL)
	time.Sleep(500 * time.Millisecond)

	runMQTTLoad(mqttCfg)
}

// runStreamLoad opens a single client-streaming RPC and pushes all messages
// down it sequentially. The stream provides its own framing so there is no
// per-message round-trip cost — this is the natural operating mode of gRPC
// streaming.
func runStreamLoad(client pb.PerformanceServiceClient) {
	stream, err := client.ActorTelemetryStream(context.Background())
	if err != nil {
		log.Fatalf("stream: failed to open: %v", err)
	}
	for i := 0; i < numMessages; i++ {
		if err := stream.Send(&pb.ActorTelemetryStreamRequest{
			ActorId:   "actor-1",
			PosX:      int32(i),
			PosY:      int32(i * 2),
			Timestamp: uint64(time.Now().UnixNano()),
		}); err != nil {
			log.Fatalf("stream: send failed at %d: %v", i, err)
		}
		logProgress("stream", i, numMessages)
	}
	// CloseAndRecv blocks until the server has processed every message and
	// replied, so we know the benchmark is fully complete before returning.
	if _, err := stream.CloseAndRecv(); err != nil {
		log.Fatalf("stream: close failed: %v", err)
	}
	log.Printf("stream: sent %d messages", numMessages)
}

// runUnaryLoad sends each message as a separate RPC and waits for the
// response before sending the next. No goroutines, no pipelining — this
// exposes the true per-message round-trip cost of unary gRPC.
func runUnaryLoad(client pb.PerformanceServiceClient) {
	for i := 0; i < numMessages; i++ {
		if _, err := client.ActorTelemetry(context.Background(), &pb.ActorTelemetryRequest{
			ActorId:   "actor-1",
			PosX:      int32(i),
			PosY:      int32(i * 2),
			Timestamp: uint64(time.Now().UnixNano()),
		}); err != nil {
			log.Fatalf("unary: call failed at %d: %v", i, err)
		}
		logProgress("unary", i, numMessages)
	}
	log.Printf("unary: sent %d messages", numMessages)
}

// runHTTPLoad posts each message and waits for the 204 before continuing.
// Same discipline as unary gRPC — one round-trip at a time.
func runHTTPLoad(url string) {
	httpClient := &http.Client{}
	for i := 0; i < numMessages; i++ {
		body, _ := json.Marshal(wirePayload{
			ActorID:   "actor-1",
			PosX:      int32(i),
			PosY:      int32(i * 2),
			Timestamp: uint64(time.Now().UnixNano()),
		})
		resp, err := httpClient.Post(
			"http://localhost:8080/telemetry",
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			log.Fatalf("http: post failed at %d: %v", i, err)
		}
		resp.Body.Close()
		logProgress("http", i, numMessages)
	}
	log.Printf("http: sent %d messages", numMessages)
}

// runMQTTLoad publishes messages sequentially. token.Wait() ensures each
// publish has been handed to the network stack before moving on — without it,
// the loop races far ahead of the broker and the comparison becomes meaningless.
func runMQTTLoad(cfg *shared.MqttConfig) {
	mqttClient, err := shared.ConnectToMqttBroker(cfg, "perf-client", true)
	if err != nil {
		log.Fatalf("mqtt: connect failed: %v", err)
	}
	defer mqttClient.Disconnect(250)

	topic := shared.ActorTelemetryTopic("actor-1")
	for i := 0; i < numMessages; i++ {
		body, _ := json.Marshal(wirePayload{
			ActorID:   "actor-1",
			PosX:      int32(i),
			PosY:      int32(i * 2),
			Timestamp: uint64(time.Now().UnixNano()),
			SentAt:    uint64(time.Now().UnixNano()),
		})
		token := mqttClient.Publish(topic, 1, false, body)
		token.Wait()
		logProgress("mqtt", i, numMessages)
	}
	log.Printf("mqtt: sent %d messages", numMessages)
}
