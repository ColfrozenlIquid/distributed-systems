package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"shared"
	"sync"
	"time"

	pb "performance/gen"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TelemetryPayload struct {
	ActorID   string `json:"actor_id"`
	PosX      int32  `json:"pos_x"`
	PosY      int32  `json:"pos_y"`
	Timestamp uint64 `json:"timestamp"`
}

type Client struct {
	client pb.PerformanceServiceClient
}

func main() {
	v := viper.New()
	v.AutomaticEnv()
	mqttCfg, err := shared.LoadMqttConfig(v)
	if err != nil {
		log.Fatal("failed to load mqtt config", zap.Error(err))
	}

	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := Client{
		client: pb.NewPerformanceServiceClient(conn),
	}

	// client.runStreamLoad()
	// client.runUnaryLoad()
	// client.runHTTPLoad()
	client.runMqttLoad(mqttCfg)
}

func (c *Client) runStreamLoad() {
	stream, err := c.client.ActorTelemetryStream(context.Background())
	if err != nil {
		log.Fatalf("failed to open stream: %v", err)
	}

	const numMessages = 100_000
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		msg := &pb.ActorTelemetryStreamRequest{
			ActorId:   "actor-1",
			PosX:      int32(i),
			PosY:      int32(i * 2),
			Timestamp: uint64(time.Now().UnixNano()),
		}
		if err := stream.Send(msg); err != nil {
			log.Fatalf("failed to send: %v", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	_ = resp
	if err != nil {
		log.Fatalf("failed to close stream: %v", err)
	}

	log.Printf("sent %d messages in %v (%.0f msgs/sec)",
		numMessages, time.Since(start), float64(numMessages)/time.Since(start).Seconds())
}

func (c *Client) runUnaryLoad() {
	const numRequests = 100_000
	const concurrency = 50

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	start := time.Now()
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			_, err := c.client.ActorTelemetry(context.Background(), &pb.ActorTelemetryRequest{
				ActorId:   "actor-1",
				PosX:      int32(i),
				PosY:      int32(i * 2),
				Timestamp: uint64(time.Now().UnixNano()),
			})
			if err != nil {
				log.Printf("call failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	elapsed := time.Since(start)
	log.Printf("sent %d unary calls in %v (%.0f req/sec)", numRequests, elapsed, float64(numRequests)/elapsed.Seconds())
}

func (c *Client) runHTTPLoad() {
	const numRequests = 100_000
	const concurrency = 50
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	httpClient := &http.Client{}
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			payload := TelemetryPayload{
				ActorID:   "actor-1",
				PosX:      int32(i),
				PosY:      int32(i * 2),
				Timestamp: uint64(time.Now().UnixNano()),
			}
			body, _ := json.Marshal(payload)

			resp, err := httpClient.Post(
				"http://localhost:8080/telemetry",
				"application/json",
				bytes.NewReader(body),
			)
			if err != nil {
				log.Printf("call failed: %v", err)
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("sent %d HTTP calls in %v (%.0f req/sec)", numRequests, elapsed, float64(numRequests)/elapsed.Seconds())
}

func (c *Client) runMqttLoad(cfg *shared.MqttConfig) {
	client, err := shared.ConnectToMqttBroker(cfg, "perf-client", true)
	if err != nil {
		log.Fatalf("mqtt connect failed: %v", err)
	}
	defer client.Disconnect(250)

	const numMessages = 100_000
	topic := shared.ActorTelemetryTopic("actor-1")

	start := time.Now()
	for i := 0; i < numMessages; i++ {
		payload := map[string]any{
			"actor_id":  "actor-1",
			"pos_x":     int32(i),
			"pos_y":     int32(i * 2),
			"timestamp": uint64(time.Now().UnixNano()),
		}
		body, _ := json.Marshal(payload)

		token := client.Publish(topic, 0, false, body)
		// don't Wait() here if you want max publish throughput —
		// see note below on what Wait() actually measures
		_ = token
	}
	elapsed := time.Since(start)
	log.Printf("published %d mqtt messages in %v (%.0f msgs/sec)",
		numMessages, elapsed, float64(numMessages)/elapsed.Seconds())
}
