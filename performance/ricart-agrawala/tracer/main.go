// Command mqtt-tracer is a passive observability client for the Ricart-Agrawala
// MQTT traffic. It subscribes to every ricart/* topic as a separate client,
// decodes each message using the existing `shared` types, and writes:
//
//  1. a JSONL trace file (one line per message seen) for offline analysis
//     (message counts by type, sizes, ordering, etc.)
//  2. a live "handover summary" line every time a node enters the critical
//     section: how long the resource was free, and how many messages were
//     exchanged to negotiate that handover.
//
// This does NOT modify your nodes at all — it's a second client sitting on
// the broker, so it sees exactly what the broker fans out.
//
// Lives at performance/ricart-agrawala/tracer/main.go, built the same way
// as your other services via `docker build --build-arg BUILD_TARGET=tracer`.
//
// Local dev, without docker:
//
//	go run ./performance/ricart-agrawala/tracer -broker tcp://localhost:1883 -scenario baseline
//
// In docker-compose it's driven by env vars instead (MQTT_BROKER_URL,
// SCENARIO, TRACE_OUTPUT_DIR) — see envOr() below.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	// Adjust this import path to match your module.
	shared "shared"
)

// TraceEvent is one line of the JSONL trace. Fields are left empty (omitted)
// when they don't apply to a given message type.
type TraceEvent struct {
	Seq          uint64    `json:"seq"`
	ArrivalTime  time.Time `json:"arrival_time"`
	Scenario     string    `json:"scenario,omitempty"`
	Topic        string    `json:"topic"`
	MsgType      string    `json:"msg_type"`
	SizeBytes    int       `json:"size_bytes"`
	ActorID      string    `json:"actor_id,omitempty"`
	FromID       string    `json:"from_id,omitempty"`
	ToID         string    `json:"to_id,omitempty"`
	ResourceID   string    `json:"resource_id,omitempty"`
	LogicalClock uint64    `json:"logical_clock,omitempty"`
	Online       *bool     `json:"online,omitempty"`
	// OriginTimestamp is the timestamp embedded in the payload itself
	// (only present on Presence/Request/CsEvent). Comparing this to
	// ArrivalTime gives you one-way transit time for that message,
	// PROVIDED node clocks are synced (e.g. NTP) — otherwise treat it
	// as a rough indicator only and rely on ArrivalTime deltas, which
	// are all measured on this single tracer's clock and therefore
	// immune to cross-node clock skew.
	OriginTimestamp *time.Time `json:"origin_timestamp,omitempty"`
}

// HandoverSummary is emitted every time a node enters the CS. It answers
// your core question directly: how long was the resource free, and what
// did it cost (in messages) to hand it over.
type HandoverSummary struct {
	ResourceID       string        `json:"resource_id"`
	EnteredBy        string        `json:"entered_by"`
	Scenario         string        `json:"scenario,omitempty"`
	EnteredAt        time.Time     `json:"entered_at"`
	FreeDuration     time.Duration `json:"free_duration_ns"`
	MessagesSinceEnd int           `json:"messages_since_last_release"`
	RequestsSeen     int           `json:"requests_seen"`
	RepliesSeen      int           `json:"replies_seen"`
}

type tracer struct {
	mu       sync.Mutex
	w        *bufio.Writer
	f        *os.File
	seq      uint64
	scenario string

	// per-resource windowing state, reset on every CsEnter
	lastReleaseAt map[string]time.Time
	msgSinceEnd   map[string]int
	reqSinceEnd   map[string]int
	replySinceEnd map[string]int

	// aggregate counters for the end-of-run summary
	totalByType map[string]int
	totalBytes  int64
}

func newTracer(path, scenario string) (*tracer, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &tracer{
		w:             bufio.NewWriterSize(f, 64*1024),
		f:             f,
		scenario:      scenario,
		lastReleaseAt: make(map[string]time.Time),
		msgSinceEnd:   make(map[string]int),
		reqSinceEnd:   make(map[string]int),
		replySinceEnd: make(map[string]int),
		totalByType:   make(map[string]int),
	}, nil
}

func (t *tracer) writeLocked(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Warn("failed to marshal trace event", "err", err)
		return
	}
	t.w.Write(b)
	t.w.WriteByte('\n')
}

func (t *tracer) record(ev TraceEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.seq++
	ev.Seq = t.seq
	ev.Scenario = t.scenario
	t.writeLocked(ev)

	t.totalByType[ev.MsgType]++
	t.totalBytes += int64(ev.SizeBytes)

	if ev.ResourceID == "" {
		return
	}
	switch ev.MsgType {
	case "request":
		t.msgSinceEnd[ev.ResourceID]++
		t.reqSinceEnd[ev.ResourceID]++
	case "reply":
		t.msgSinceEnd[ev.ResourceID]++
		t.replySinceEnd[ev.ResourceID]++
	}
}

func (t *tracer) onCsExit(ev TraceEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastReleaseAt[ev.ResourceID] = ev.ArrivalTime
	// Reset window counters — anything from here counts toward the
	// *next* handover.
	t.msgSinceEnd[ev.ResourceID] = 0
	t.reqSinceEnd[ev.ResourceID] = 0
	t.replySinceEnd[ev.ResourceID] = 0
}

func (t *tracer) onCsEnter(ev TraceEvent) {
	t.mu.Lock()
	last, known := t.lastReleaseAt[ev.ResourceID]
	summary := HandoverSummary{
		ResourceID:       ev.ResourceID,
		EnteredBy:        ev.ActorID,
		Scenario:         t.scenario,
		EnteredAt:        ev.ArrivalTime,
		MessagesSinceEnd: t.msgSinceEnd[ev.ResourceID],
		RequestsSeen:     t.reqSinceEnd[ev.ResourceID],
		RepliesSeen:      t.replySinceEnd[ev.ResourceID],
	}
	if known {
		summary.FreeDuration = ev.ArrivalTime.Sub(last)
	}
	t.writeLocked(map[string]any{"handover": summary})
	t.mu.Unlock()

	slog.Info("cs handover",
		"resource", ev.ResourceID,
		"entered_by", ev.ActorID,
		"free_for", summary.FreeDuration,
		"messages", summary.MessagesSinceEnd,
		"requests", summary.RequestsSeen,
		"replies", summary.RepliesSeen,
	)
}

func (t *tracer) flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.w.Flush()
}

func (t *tracer) printFinalSummary() {
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Println("\n--- trace summary ---")
	fmt.Printf("scenario:    %s\n", t.scenario)
	fmt.Printf("total msgs:  %d\n", t.seq)
	fmt.Printf("total bytes: %d\n", t.totalBytes)
	for msgType, count := range t.totalByType {
		fmt.Printf("  %-10s %d\n", msgType, count)
	}
}

func (t *tracer) close() {
	t.flush()
	t.f.Close()
}

// envOr returns the env var if set, otherwise falls back to def. Lets the
// same binary be driven by flags locally and by env vars in docker-compose,
// matching how your other services (collector, proxy-setup) are configured.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	broker := flag.String("broker", envOr("MQTT_BROKER_URL", "tcp://localhost:1883"), "MQTT broker URL")
	scenario := flag.String("scenario", envOr("SCENARIO", "default"), "label for this test run, e.g. a toxiproxy condition name")
	outDir := flag.String("out", envOr("TRACE_OUTPUT_DIR", "."), "directory to write the trace file into")
	flag.Parse()

	outPath := fmt.Sprintf("%s/mqtt_trace_%s_%d.jsonl", *outDir, *scenario, time.Now().Unix())
	tr, err := newTracer(outPath, *scenario)
	if err != nil {
		slog.Error("failed to open trace file", "err", err)
		os.Exit(1)
	}
	defer tr.close()

	opts := mqtt.NewClientOptions().
		AddBroker(*broker).
		SetClientID("ricart-tracer-" + fmt.Sprint(time.Now().UnixNano())).
		SetCleanSession(true).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("failed to connect", "err", token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)

	client.AddRoute(shared.SubAllRicartPresence, func(_ mqtt.Client, msg mqtt.Message) {
		var p shared.RicartPresence
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			slog.Warn("malformed presence payload", "err", err)
			return
		}
		online := p.Online
		tr.record(TraceEvent{
			ArrivalTime:     time.Now(),
			Topic:           msg.Topic(),
			MsgType:         "presence",
			SizeBytes:       len(msg.Payload()),
			ActorID:         p.ActorID,
			Online:          &online,
			OriginTimestamp: &p.Timestamp,
		})
	})

	client.AddRoute(shared.SubAllRicartRequest, func(_ mqtt.Client, msg mqtt.Message) {
		var r shared.RicartRequest
		if err := json.Unmarshal(msg.Payload(), &r); err != nil {
			slog.Warn("malformed request payload", "err", err)
			return
		}
		tr.record(TraceEvent{
			ArrivalTime:     time.Now(),
			Topic:           msg.Topic(),
			MsgType:         "request",
			SizeBytes:       len(msg.Payload()),
			FromID:          r.FromID,
			ResourceID:      r.ResourceID,
			LogicalClock:    r.LogicalClock,
			OriginTimestamp: &r.Timestamp,
		})
	})

	client.AddRoute(shared.SubAllRicartReply, func(_ mqtt.Client, msg mqtt.Message) {
		var r shared.RicartReply
		if err := json.Unmarshal(msg.Payload(), &r); err != nil {
			slog.Warn("malformed reply payload", "err", err)
			return
		}
		tr.record(TraceEvent{
			ArrivalTime:  time.Now(),
			Topic:        msg.Topic(),
			MsgType:      "reply",
			SizeBytes:    len(msg.Payload()),
			FromID:       r.FromID,
			ToID:         r.ToID,
			ResourceID:   r.ResourceID,
			LogicalClock: r.LogicalClock,
		})
	})

	client.AddRoute(shared.SubAllRicartCsEnter, func(_ mqtt.Client, msg mqtt.Message) {
		var e shared.RicartCsEvent
		if err := json.Unmarshal(msg.Payload(), &e); err != nil {
			slog.Warn("malformed cs-enter payload", "err", err)
			return
		}
		ev := TraceEvent{
			ArrivalTime:     time.Now(),
			Topic:           msg.Topic(),
			MsgType:         "cs_enter",
			SizeBytes:       len(msg.Payload()),
			ActorID:         e.ActorID,
			ResourceID:      e.ResourceID,
			LogicalClock:    e.LogicalClock,
			OriginTimestamp: &e.Timestamp,
		}
		tr.record(ev)
		tr.onCsEnter(ev)
	})

	client.AddRoute(shared.SubAllRicartCsExit, func(_ mqtt.Client, msg mqtt.Message) {
		var e shared.RicartCsEvent
		if err := json.Unmarshal(msg.Payload(), &e); err != nil {
			slog.Warn("malformed cs-exit payload", "err", err)
			return
		}
		ev := TraceEvent{
			ArrivalTime:     time.Now(),
			Topic:           msg.Topic(),
			MsgType:         "cs_exit",
			SizeBytes:       len(msg.Payload()),
			ActorID:         e.ActorID,
			ResourceID:      e.ResourceID,
			LogicalClock:    e.LogicalClock,
			OriginTimestamp: &e.Timestamp,
		}
		tr.record(ev)
		tr.onCsExit(ev)
	})

	// Subscribe at the SAME QoS as the publishers use, so the broker
	// never has to downgrade delivery to this subscriber and you don't
	// lose visibility into messages your nodes actually received at a
	// higher QoS.
	// SubscribeMultiple wants map[string]byte, but the shared package
	// keys its QoS map by the distinct TopicSubscription type, so we
	// convert (same thing shared.Streams.Subscribe does internally via
	// lo.MapKeys).
	filters := make(map[string]byte)
	for topic, qos := range shared.RicartSubscriptionsWithQoSLevels() {
		filters[topic.String()] = qos
	}
	if token := client.SubscribeMultiple(filters, nil); token.Wait() && token.Error() != nil {
		slog.Error("failed to subscribe", "err", token.Error())
		os.Exit(1)
	}

	slog.Info("tracer running", "broker", *broker, "scenario", *scenario, "out", outPath)

	// Periodic flush so you can tail -f the file mid-run.
	flushTicker := time.NewTicker(2 * time.Second)
	defer flushTicker.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-flushTicker.C:
			tr.flush()
		case <-sigCh:
			tr.printFinalSummary()
			return
		}
	}
}
