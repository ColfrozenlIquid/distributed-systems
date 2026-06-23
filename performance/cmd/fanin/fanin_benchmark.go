package main

import (
	"encoding/json"
	"fmt"
	"log"
	"performance/internal"
	"shared"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// producerCounts is the sweep of fan-in sizes to benchmark.
var producerCounts = []int{1, 10, 50, 100, 500, 1000}

// faninNumMessages is the number of messages each producer sends per run.
// Total messages received by the subscriber = numProducers * faninNumMessages.
const faninNumMessages = 2_000

// faninWirePayload mirrors fanoutWirePayload. ProducerID identifies which
// producer sent the message so the subscriber can track per-producer progress
// if needed later.
type faninWirePayload struct {
	ActorID    string `json:"actor_id"`
	ProducerID int    `json:"producer_id"`
	PosX       int32  `json:"pos_x"`
	PosY       int32  `json:"pos_y"`
	Timestamp  uint64 `json:"timestamp"`
	SentAt     uint64 `json:"sent_at"` // first-message epoch ns for this producer
}

// faninBenchmark coordinates a single fan-in run (one value of N producers).
//
// The clock starts when the subscriber receives its very first message across
// all producers, and stops when totalExpected messages have been received.
// This measures end-to-end broker ingestion + delivery throughput under
// concurrent publish load.
type faninBenchmark struct {
	numProducers  int
	totalExpected int64 // numProducers * faninNumMessages

	received      atomic.Int64 // total messages received by the subscriber
	firstReceived sync.Once
	start         time.Time

	allDone chan struct{} // closed when totalExpected messages are received

	// per-producer received counts, useful for diagnosing imbalanced delivery
	perProducer []atomic.Int64
}

func newFaninBenchmark(n int) *faninBenchmark {
	return &faninBenchmark{
		numProducers:  n,
		totalExpected: int64(n) * faninNumMessages,
		allDone:       make(chan struct{}),
		perProducer:   make([]atomic.Int64, n),
	}
}

// spawnSubscriber creates a single MQTT client that subscribes to the
// wildcard topic covering all actor telemetry. It counts every message
// received and stops the clock when totalExpected have arrived.
func spawnSubscriber(cfg *shared.MqttConfig, fb *faninBenchmark) {
	clientID := fmt.Sprintf("fanin-sub-%d", fb.numProducers)
	client, err := shared.ConnectToMqttBroker(cfg, clientID, true, nil)
	if err != nil {
		log.Fatalf("fanin: subscriber connect failed: %v", err)
	}

	token := client.Subscribe(shared.SubAllActorTelemetry, 0, func(_ mqtt.Client, msg mqtt.Message) {
		var p faninWirePayload
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			return
		}

		// Start the clock on the very first message from any producer.
		fb.firstReceived.Do(func() {
			if p.SentAt > 0 {
				fb.start = time.Unix(0, int64(p.SentAt))
			} else {
				fb.start = time.Now()
			}
		})

		// Track per-producer counts for post-run diagnostics.
		if p.ProducerID >= 0 && p.ProducerID < fb.numProducers {
			fb.perProducer[p.ProducerID].Add(1)
		}

		n := fb.received.Add(1)
		if n == fb.totalExpected {
			elapsed := time.Since(fb.start)
			client.Disconnect(250)

			totalMsgs := int(fb.totalExpected)
			tag := fmt.Sprintf("mqtt_fanin_%d_producers", fb.numProducers)
			name := fmt.Sprintf("MQTT Fan-in (%d producers)", fb.numProducers)

			log.Printf("[fanin n=%d] subscriber received all %d messages in %v (%.0f msgs/sec aggregate, %.0f msgs/sec per producer)",
				fb.numProducers,
				totalMsgs,
				elapsed,
				float64(totalMsgs)/elapsed.Seconds(),
				float64(faninNumMessages)/elapsed.Seconds(),
			)

			// Log any producers that delivered fewer messages than expected —
			// this catches broker drops or subscriber back-pressure silently
			// discarding messages.
			for i, c := range fb.perProducer {
				got := c.Load()
				if got != faninNumMessages {
					log.Printf("[fanin n=%d] WARNING: producer-%d delivered %d/%d messages",
						fb.numProducers, i, got, faninNumMessages)
				}
			}

			internal.SaveResult(tag, name, totalMsgs, elapsed)
			close(fb.allDone)
		}
	})
	token.Wait()
	if err := token.Error(); err != nil {
		log.Fatalf("fanin: subscriber subscribe failed: %v", err)
	}
}

// spawnProducers launches n goroutines, each with its own MQTT client,
// each publishing faninNumMessages messages to its own actor topic.
// All producers start simultaneously via a shared startGate so the broker
// sees concurrent inbound load rather than a sequential ramp.
func spawnProducers(cfg *shared.MqttConfig, fb *faninBenchmark) {
	var wg sync.WaitGroup
	startGate := make(chan struct{})

	for i := 0; i < fb.numProducers; i++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()

			clientID := fmt.Sprintf("fanin-pub-%d-%d", fb.numProducers, producerID)
			client, err := shared.ConnectToMqttBroker(cfg, clientID, true, nil)
			if err != nil {
				log.Fatalf("fanin: producer-%d connect failed: %v", producerID, err)
			}
			defer client.Disconnect(250)

			// Each producer publishes to its own unique actor topic so the
			// broker's topic routing is exercised, and all land under the
			// wildcard the subscriber uses.
			topic := shared.ActorTelemetryTopic(fmt.Sprintf("actor-%d", producerID))

			// Wait at the gate so all producers start publishing at the
			// same instant, maximising concurrent inbound pressure.
			<-startGate

			firstSentAt := uint64(time.Now().UnixNano())
			for j := 0; j < faninNumMessages; j++ {
				body, _ := json.Marshal(faninWirePayload{
					ActorID:    fmt.Sprintf("actor-%d", producerID),
					ProducerID: producerID,
					PosX:       int32(j),
					PosY:       int32(j * 2),
					Timestamp:  uint64(time.Now().UnixNano()),
					SentAt:     firstSentAt,
				})
				token := client.Publish(topic, 1, false, body)
				token.Wait()
			}
			log.Printf("[fanin n=%d] producer-%d: sent %d messages",
				fb.numProducers, producerID, faninNumMessages)
		}(i)
	}

	// Give all goroutines time to connect before opening the gate.
	// Scale grace period with N for the same reason as in fan-out.
	grace := time.Duration(fb.numProducers)*10*time.Millisecond + 200*time.Millisecond
	time.Sleep(grace)

	log.Printf("[fanin n=%d] opening start gate — all producers publishing now", fb.numProducers)
	close(startGate)

	wg.Wait()
	log.Printf("[fanin n=%d] all producers finished sending", fb.numProducers)
}

// RunMQTTFaninBenchmark is the entry point. Call this from main() alongside
// RunMQTTFanoutBenchmark, or run this file standalone.
//
//	RunMQTTFaninBenchmark(mqttCfg)
func RunMQTTFaninBenchmark(cfg *shared.MqttConfig) {
	logger, _ := zap.NewDevelopment()

	for _, n := range producerCounts {
		logger.Info("starting fan-in run", zap.Int("producers", n))

		fb := newFaninBenchmark(n)

		// Subscriber must be ready before any producer sends a message,
		// otherwise early messages are lost and the run never completes.
		spawnSubscriber(cfg, fb)

		// Brief pause to ensure the subscribe ACK has arrived and the broker
		// has registered the subscription before producers flood it.
		time.Sleep(200 * time.Millisecond)

		// Producers connect, wait at the gate, then all publish concurrently.
		// spawnProducers returns only after every producer has sent all its
		// messages, so the timeout below is purely a safety net.
		go spawnProducers(cfg, fb)

		timeout := time.Duration(n)*50*time.Millisecond + 120*time.Second
		select {
		case <-fb.allDone:
			logger.Info("fan-in run complete", zap.Int("producers", n))
		case <-time.After(timeout):
			logger.Error("fan-in run timed out — subscriber did not receive all messages",
				zap.Int("producers", n),
				zap.Int64("received", fb.received.Load()),
				zap.Int64("expected", fb.totalExpected),
			)
		}

		// Drain period between runs so retained QoS messages or in-flight
		// retries from this run don't skew the next.
		time.Sleep(500 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// Standalone entry point
//
// Build and run with:
//
//	go run fanin_bench.go
//
// Reads the same environment variables / .env as the rest of the project.
// ---------------------------------------------------------------------------
func main() {
	v := viper.New()
	v.AutomaticEnv()

	mqttCfg, err := shared.LoadMqttConfig(v)
	if err != nil {
		log.Fatal("fanin: failed to load mqtt config", zap.Error(err))
	}

	RunMQTTFaninBenchmark(mqttCfg)
}
