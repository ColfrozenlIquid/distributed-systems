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

// subscriberCounts is the sweep of fan-out sizes to benchmark.
var subscriberCounts = []int{1, 10, 50, 100, 500, 1000}

// fanoutNumMessages is intentionally the same constant as in your existing
// client/server so results are directly comparable.
const fanoutNumMessages = 5_000

// fanoutWirePayload mirrors the wirePayload used in your existing code.
// SentAt records the nanosecond timestamp of the *first* message published in
// a run so every subscriber shares the same start anchor.
type fanoutWirePayload struct {
	ActorID   string `json:"actor_id"`
	PosX      int32  `json:"pos_x"`
	PosY      int32  `json:"pos_y"`
	Timestamp uint64 `json:"timestamp"`
	SentAt    uint64 `json:"sent_at"` // first-message epoch ns, constant per run
}

// fanoutBenchmark coordinates a single fan-out run (one value of N).
//
// The clock starts at SentAt (set on the first published message) and stops
// when every subscriber has received all fanoutNumMessages messages. Each
// subscriber calls subscriberDone() when its counter hits the total; the
// last one to call it writes the result and closes allDone.
type fanoutBenchmark struct {
	numSubscribers int
	firstSentAt    uint64        // set by the publisher before subscriberss receive anything
	allDone        chan struct{} // closed when every subscriber has finished
	doneCount      atomic.Int64  // how many subscribers have finished
}

func newFanoutBenchmark(n int) *fanoutBenchmark {
	return &fanoutBenchmark{
		numSubscribers: n,
		allDone:        make(chan struct{}),
	}
}

// subscriberDone is called by each subscriber when it has received all
// fanoutNumMessages messages. elapsed is measured relative to firstSentAt.
// The last subscriber to call this saves the result and unblocks the runner.
func (f *fanoutBenchmark) subscriberDone(subID int, elapsed time.Duration) {
	log.Printf("[fanout n=%d] sub-%d finished in %v", f.numSubscribers, subID, elapsed)

	if f.doneCount.Add(1) == int64(f.numSubscribers) {
		tag := fmt.Sprintf("mqtt_fanout_%d_subs", f.numSubscribers)
		name := fmt.Sprintf("MQTT Fan-out (%d subscribers)", f.numSubscribers)
		log.Printf("[fanout n=%d] all subscribers done in %v (%.0f msgs/sec per subscriber)",
			f.numSubscribers, elapsed,
			float64(fanoutNumMessages)/elapsed.Seconds())
		internal.SaveResult(tag, name, fanoutNumMessages, elapsed)
		close(f.allDone)
	}
}

// spawnSubscribers creates n MQTT clients, each subscribing to the same
// wildcard topic used by your existing server. They share the benchmark
// barrier but are otherwise independent.
func spawnSubscribers(cfg *shared.MqttConfig, fb *fanoutBenchmark) {
	var wg sync.WaitGroup
	for i := 0; i < fb.numSubscribers; i++ {
		wg.Add(1)
		go func(subID int) {
			defer wg.Done()

			// Each subscriber must have a unique client ID or the broker will
			// disconnect earlier connections when a new one arrives with the
			// same ID. We embed the subscriber count in the ID so runs don't
			// accidentally share persistent sessions.
			clientID := fmt.Sprintf("fanout-sub-%d-%d", fb.numSubscribers, subID)
			client, err := shared.ConnectToMqttBroker(cfg, clientID, true, nil)
			if err != nil {
				log.Fatalf("fanout: sub-%d connect failed: %v", subID, err)
			}

			var (
				count     atomic.Int64
				startOnce sync.Once
				start     time.Time
			)

			token := client.Subscribe(shared.SubAllActorTelemetry, 0, func(_ mqtt.Client, msg mqtt.Message) {
				var p fanoutWirePayload
				if err := json.Unmarshal(msg.Payload(), &p); err != nil {
					return
				}

				// Anchor the clock to when the publisher sent the first message,
				// not when this subscriber happened to connect.
				startOnce.Do(func() {
					if p.SentAt > 0 {
						start = time.Unix(0, int64(p.SentAt))
					} else {
						start = time.Now()
					}
				})

				n := count.Add(1)
				if n == int64(fanoutNumMessages) {
					elapsed := time.Since(start)
					client.Disconnect(250)
					fb.subscriberDone(subID, elapsed)
				}
			})
			token.Wait()
			if err := token.Error(); err != nil {
				log.Fatalf("fanout: sub-%d subscribe failed: %v", subID, err)
			}
		}(i)
	}
	// We don't wait on wg here — subscribers run until they hit their message
	// count and call client.Disconnect themselves. wg is kept for clarity but
	// the fanoutBenchmark.allDone channel is the real synchronisation point.
}

// runFanoutPublisher publishes fanoutNumMessages to the broker sequentially,
// embedding the nanosecond timestamp of the *first* message in every payload
// so subscribers share a common start time regardless of when they each
// receive that first message.
func runFanoutPublisher(cfg *shared.MqttConfig, n int) {
	client, err := shared.ConnectToMqttBroker(cfg, fmt.Sprintf("fanout-pub-%d", n), true, nil)
	if err != nil {
		log.Fatalf("fanout: publisher connect failed: %v", err)
	}
	defer client.Disconnect(250)

	topic := shared.ActorTelemetryTopic("actor-1")

	// Capture the first-message timestamp before the loop so it's the same
	// for every payload in this run.
	firstSentAt := uint64(time.Now().UnixNano())

	for i := 0; i < fanoutNumMessages; i++ {
		body, _ := json.Marshal(fanoutWirePayload{
			ActorID:   "actor-1",
			PosX:      int32(i),
			PosY:      int32(i * 2),
			Timestamp: uint64(time.Now().UnixNano()),
			SentAt:    firstSentAt, // constant across the whole run
		})
		token := client.Publish(topic, 1, false, body)
		token.Wait()
	}
	log.Printf("[fanout n=%d] publisher: sent %d messages", n, fanoutNumMessages)
}

// RunMQTTFanoutBenchmark is the entry point. Call this from main() after
// your existing benchmark runs, or standalone.
//
//	RunMQTTFanoutBenchmark(mqttCfg)
func RunMQTTFanoutBenchmark(cfg *shared.MqttConfig) {
	logger, _ := zap.NewDevelopment()

	for _, n := range subscriberCounts {
		logger.Info("starting fanout run", zap.Int("subscribers", n))

		fb := newFanoutBenchmark(n)
		spawnSubscribers(cfg, fb)

		// Give every goroutine time to connect and subscribe before the
		// publisher floods the broker. Scale the grace period with N because
		// each MQTT connect is a round-trip and large subscriber counts take
		// noticeably longer to settle.
		grace := time.Duration(n)*10*time.Millisecond + 200*time.Millisecond
		time.Sleep(grace)

		runFanoutPublisher(cfg, n)

		select {
		case <-fb.allDone:
			logger.Info("fanout run complete", zap.Int("subscribers", n))
		case <-time.After(120 * time.Second):
			logger.Error("fanout run timed out — not all subscribers finished",
				zap.Int("subscribers", n),
				zap.Int64("finished", fb.doneCount.Load()))
		}

		// Let the broker fully drain before the next run so stale messages
		// from this run don't pollute the next subscriber set.
		time.Sleep(500 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------------
// Standalone entry point
//
// If you want to run just this benchmark independently (not as part of the
// main client binary), build with:
//
//	go run fanout_bench.go
//
// It reads the same .env / environment variables as the rest of the project.
// ---------------------------------------------------------------------------

func main() {
	v := viper.New()
	v.AutomaticEnv()

	mqttCfg, err := shared.LoadMqttConfig(v)
	if err != nil {
		log.Fatal("fanout: failed to load mqtt config", zap.Error(err))
	}

	RunMQTTFanoutBenchmark(mqttCfg)
}
