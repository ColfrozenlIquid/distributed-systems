package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "simulation/gen"
)

func main() {
	addr := "localhost:50051"
	totalRequests := 100000
	concurrency := 1000

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewWorldServiceClient(conn)

	var wg sync.WaitGroup
	requestCh := make(chan int, totalRequests)

	for i := 0; i < totalRequests; i++ {
		requestCh <- i
	}
	close(requestCh)

	sensorTypes := []pb.SensorType{
		pb.SensorType_SENSOR_TYPE_WATER_LEVEL,
		pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE,
		pb.SensorType_SENSOR_TYPE_VIBRATION,
		pb.SensorType_SENSOR_TYPE_SUPPLY_LEVEL,
	}

	start := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range requestCh {
				x := rand.Intn(20)
				y := rand.Intn(20)

				randomType := sensorTypes[rand.Intn(len(sensorTypes))]

				req := &pb.TileReadingRequest{
					XPos:       int32(x),
					YPos:       int32(y),
					SensorType: randomType,
				}

				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				res, err := client.TileReading(ctx, req)
				cancel()

				if err != nil {
					log.Printf("error: %v", err)
				} else {
					fmt.Printf("Sensor type: %s, reading: %f", res.SensorType.String(), res.SensorReading)
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	fmt.Printf("Finished %d requests in %s\n", totalRequests, elapsed)
	fmt.Printf("Throughput: %.2f req/sec\n", float64(totalRequests)/elapsed.Seconds())
}
