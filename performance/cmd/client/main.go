package main

import (
	"context"
	"log"
	"sync"
	"time"

	pb "performance/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPerformanceServiceClient(conn)

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

			_, err := client.ActorTelemetry(context.Background(), &pb.ActorTelemetryRequest{
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

// func main() {
// 	conn, err := grpc.NewClient("localhost:50051",
// 		grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		log.Fatalf("did not connect: %v", err)
// 	}
// 	defer conn.Close()

// 	client := pb.NewPerformanceServiceClient(conn)

// 	stream, err := client.ActorTelemetry(context.Background())
// 	if err != nil {
// 		log.Fatalf("failed to open stream: %v", err)
// 	}

// 	const numMessages = 100_000
// 	start := time.Now()

// 	for i := 0; i < numMessages; i++ {
// 		msg := &pb.ActorTelemetryRequest{
// 			ActorId:   "actor-1",
// 			PosX:      int32(i),
// 			PosY:      int32(i * 2),
// 			Timestamp: uint64(time.Now().UnixNano()),
// 		}
// 		if err := stream.Send(msg); err != nil {
// 			log.Fatalf("failed to send: %v", err)
// 		}
// 	}

// 	resp, err := stream.CloseAndRecv()
// 	_ = resp // ActorTelemetryResponse{} is empty in your proto
// 	if err != nil {
// 		log.Fatalf("failed to close stream: %v", err)
// 	}

// 	log.Printf("sent %d messages in %v (%.0f msgs/sec)",
// 		numMessages, time.Since(start), float64(numMessages)/time.Since(start).Seconds())
// }
