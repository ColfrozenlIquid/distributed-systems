package aufgabe4tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	ricartStateIdle    = "IDLE"
	ricartStateWanting = "WANTING"
	ricartStateInCS    = "IN_CS"
)

type ricartPresence struct {
	ActorID   string    `json:"actor_id"`
	ActorType string    `json:"actor_type"`
	Timestamp time.Time `json:"timestamp"`
	Online    bool      `json:"online"`
}

type ricartRequest struct {
	FromID       string    `json:"from_id"`
	ResourceID   string    `json:"resource_id"`
	LogicalClock uint64    `json:"logical_clock"`
	Timestamp    time.Time `json:"timestamp"`
}

type ricartReply struct {
	FromID       string `json:"from_id"`
	ToID         string `json:"to_id"`
	ResourceID   string `json:"resource_id"`
	LogicalClock uint64 `json:"logical_clock"`
}

type ricartCSEvent struct {
	ActorID      string    `json:"actor_id"`
	ResourceID   string    `json:"resource_id"`
	LogicalClock uint64    `json:"logical_clock"`
	Timestamp    time.Time `json:"timestamp"`
}

type ricartSnapshot struct {
	UpdatedAt    time.Time                   `json:"updated_at"`
	Participants []ricartParticipantState    `json:"participants"`
	Resources    []ricartResourceState       `json:"resources"`
	Events       []ricartObservedStatusEvent `json:"events"`
}

type ricartParticipantState struct {
	ActorID        string    `json:"actor_id"`
	Online         bool      `json:"online"`
	State          string    `json:"state"`
	ResourceID     string    `json:"resource_id,omitempty"`
	RequestClock   uint64    `json:"request_clock,omitempty"`
	RequestedAt    time.Time `json:"requested_at,omitempty"`
	PendingReplies []string  `json:"pending_replies,omitempty"`
	Replies        []string  `json:"replies,omitempty"`
	LastSeen       time.Time `json:"last_seen"`
	EnteredAt      time.Time `json:"entered_at,omitempty"`
	ExitedAt       time.Time `json:"exited_at,omitempty"`
	LastEvent      string    `json:"last_event,omitempty"`
}

type ricartResourceState struct {
	ResourceID       string    `json:"resource_id"`
	Label            string    `json:"label"`
	Status           string    `json:"status"`
	HolderActorID    string    `json:"holder_actor_id,omitempty"`
	WaitingActors    []string  `json:"waiting_actors,omitempty"`
	Requesters       []string  `json:"requesters,omitempty"`
	LastRequestAt    time.Time `json:"last_request_at,omitempty"`
	LastEnterAt      time.Time `json:"last_enter_at,omitempty"`
	LastExitAt       time.Time `json:"last_exit_at,omitempty"`
	LastLogicalClock uint64    `json:"last_logical_clock,omitempty"`
	LastEvent        string    `json:"last_event,omitempty"`
}

type ricartObservedStatusEvent struct {
	Kind         string    `json:"kind"`
	ActorID      string    `json:"actor_id"`
	PeerID       string    `json:"peer_id,omitempty"`
	ResourceID   string    `json:"resource_id"`
	LogicalClock uint64    `json:"logical_clock"`
	Timestamp    time.Time `json:"timestamp"`
	Detail       string    `json:"detail,omitempty"`
}

type testRicartNode struct {
	t          *testing.T
	id         string
	runPrefix  string
	resourceID string
	client     mqtt.Client

	mu              sync.Mutex
	clock           uint64
	state           string
	requestClock    uint64
	peers           map[string]struct{}
	pendingReplies  map[string]struct{}
	deferredReplies map[string]struct{}
	entered         chan struct{}
}

func TestRicartAgrawalaMutualExclusionAndLogicalClocks(t *testing.T) {
	httpClient := &http.Client{Timeout: 10 * time.Second}
	baseURL := controlCenterURL()
	waitForHealth(t, httpClient, baseURL)

	brokerURL := mqttBrokerURL()
	runID := fmt.Sprintf("it4-%d", time.Now().UnixNano())
	resourceID := "charging-station:8:10:" + runID
	actorIDs := []string{runID + "-actor-a", runID + "-actor-b", runID + "-actor-c"}

	nodes := make([]*testRicartNode, 0, len(actorIDs))
	for _, actorID := range actorIDs {
		node := newTestRicartNode(t, brokerURL, runID, actorID, resourceID)
		nodes = append(nodes, node)
		defer node.disconnect()
		defer node.publishPresence(false)
	}

	waitForStatus(t, httpClient, baseURL, 10*time.Second, func(snapshot ricartSnapshot) bool {
		return allParticipantsOnline(snapshot, actorIDs)
	})

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node *testRicartNode) {
			defer wg.Done()
			node.requestCS()
			node.waitEntered(10 * time.Second)
			time.Sleep(250 * time.Millisecond)
			node.releaseCS()
		}(node)
	}
	wg.Wait()

	finalSnapshot := waitForStatus(t, httpClient, baseURL, 10*time.Second, func(snapshot ricartSnapshot) bool {
		events := eventsForResource(snapshot.Events, resourceID)
		return countEvents(events, "request") == len(actorIDs) &&
			countEvents(events, "cs_enter") == len(actorIDs) &&
			countEvents(events, "cs_exit") == len(actorIDs) &&
			countEvents(events, "reply") >= len(actorIDs)*(len(actorIDs)-1)
	})

	events := eventsForResource(finalSnapshot.Events, resourceID)
	assertRicartEventFlow(t, events, actorIDs)
	assertMutualExclusion(t, events, actorIDs)
	assertRicartPriorityOrder(t, events)

	resource, ok := findResource(finalSnapshot.Resources, resourceID)
	if !ok {
		t.Fatalf("expected resource %q in /ricart/status", resourceID)
	}
	if resource.Status != "free" {
		t.Fatalf("expected resource %q to be free after all cs_exit events, got %q", resourceID, resource.Status)
	}
	if resource.HolderActorID != "" {
		t.Fatalf("expected no resource holder after all cs_exit events, got %q", resource.HolderActorID)
	}
	if resource.LastLogicalClock == 0 {
		t.Fatalf("expected resource %q to expose last_logical_clock > 0", resourceID)
	}

	t.Logf("functional result: resource=%s actors=%v requests=%d replies=%d cs_enter=%d cs_exit=%d", resourceID, actorIDs, countEvents(events, "request"), countEvents(events, "reply"), countEvents(events, "cs_enter"), countEvents(events, "cs_exit"))
	t.Logf("status summary: participants=%d resources=%d ricart_events=%d last_logical_clock=%d", len(finalSnapshot.Participants), len(finalSnapshot.Resources), len(finalSnapshot.Events), resource.LastLogicalClock)
}

func newTestRicartNode(t *testing.T, brokerURL, runPrefix, actorID, resourceID string) *testRicartNode {
	t.Helper()

	node := &testRicartNode{
		t:               t,
		id:              actorID,
		runPrefix:       runPrefix,
		resourceID:      resourceID,
		state:           ricartStateIdle,
		peers:           make(map[string]struct{}),
		pendingReplies:  make(map[string]struct{}),
		deferredReplies: make(map[string]struct{}),
		entered:         make(chan struct{}, 1),
	}

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(actorID).
		SetCleanSession(true).
		SetConnectTimeout(5 * time.Second).
		SetOrderMatters(false).
		SetDefaultPublishHandler(node.handleMessage)

	node.client = mqtt.NewClient(opts)
	token := node.client.Connect()
	if !token.WaitTimeout(10 * time.Second) {
		t.Fatalf("connect MQTT client %q to %s timed out", actorID, brokerURL)
	}
	if err := token.Error(); err != nil {
		t.Fatalf("connect MQTT client %q to %s: %v", actorID, brokerURL, err)
	}

	node.subscribe(map[string]byte{
		"ricart/presence/+":       1,
		"ricart/request":          2,
		"ricart/reply/" + actorID: 2,
	})
	node.publishPresence(true)
	return node
}

func (n *testRicartNode) subscribe(filters map[string]byte) {
	n.t.Helper()

	token := n.client.SubscribeMultiple(filters, nil)
	if !token.WaitTimeout(5 * time.Second) {
		n.t.Fatalf("subscribe MQTT client %q to %v timed out", n.id, filters)
	}
	if err := token.Error(); err != nil {
		n.t.Fatalf("subscribe MQTT client %q to %v: %v", n.id, filters, err)
	}
}

func (n *testRicartNode) handleMessage(_ mqtt.Client, message mqtt.Message) {
	switch {
	case strings.HasPrefix(message.Topic(), "ricart/presence/"):
		var presence ricartPresence
		if json.Unmarshal(message.Payload(), &presence) == nil {
			n.handlePresence(presence)
		}
	case message.Topic() == "ricart/request":
		var request ricartRequest
		if json.Unmarshal(message.Payload(), &request) == nil {
			n.handleRequest(request)
		}
	case message.Topic() == "ricart/reply/"+n.id:
		var reply ricartReply
		if json.Unmarshal(message.Payload(), &reply) == nil {
			n.handleReply(reply)
		}
	}
}

func (n *testRicartNode) handlePresence(presence ricartPresence) {
	if presence.ActorID == n.id || !strings.HasPrefix(presence.ActorID, n.runPrefix+"-") {
		return
	}

	var shouldEnter bool
	n.mu.Lock()
	if presence.Online {
		n.peers[presence.ActorID] = struct{}{}
	} else {
		delete(n.peers, presence.ActorID)
		delete(n.pendingReplies, presence.ActorID)
		delete(n.deferredReplies, presence.ActorID)
		shouldEnter = n.tryEnterCSLocked()
	}
	n.mu.Unlock()

	if shouldEnter {
		n.enterCS()
	}
}

func (n *testRicartNode) handleRequest(request ricartRequest) {
	if request.FromID == n.id || request.ResourceID != n.resourceID || !strings.HasPrefix(request.FromID, n.runPrefix+"-") {
		return
	}

	shouldReply := false
	n.mu.Lock()
	n.updateClockLocked(request.LogicalClock)
	n.peers[request.FromID] = struct{}{}
	shouldDefer := n.state == ricartStateInCS ||
		(n.state == ricartStateWanting && n.hasPriorityLocked(request))
	if shouldDefer {
		n.deferredReplies[request.FromID] = struct{}{}
	} else {
		shouldReply = true
	}
	n.mu.Unlock()

	if shouldReply {
		n.sendReply(request.FromID)
	}
}

func (n *testRicartNode) handleReply(reply ricartReply) {
	if reply.ToID != n.id || reply.ResourceID != n.resourceID {
		return
	}

	var shouldEnter bool
	n.mu.Lock()
	n.updateClockLocked(reply.LogicalClock)
	delete(n.pendingReplies, reply.FromID)
	shouldEnter = n.tryEnterCSLocked()
	n.mu.Unlock()

	if shouldEnter {
		n.enterCS()
	}
}

func (n *testRicartNode) requestCS() {
	n.mu.Lock()
	if n.state == ricartStateInCS || n.state == ricartStateWanting {
		n.mu.Unlock()
		return
	}
	n.state = ricartStateWanting
	n.requestClock = n.tickLocked()
	n.pendingReplies = make(map[string]struct{}, len(n.peers))
	for id := range n.peers {
		n.pendingReplies[id] = struct{}{}
	}
	request := ricartRequest{
		FromID:       n.id,
		ResourceID:   n.resourceID,
		LogicalClock: n.requestClock,
		Timestamp:    time.Now().UTC(),
	}
	shouldEnter := len(n.pendingReplies) == 0
	if shouldEnter {
		n.state = ricartStateInCS
	}
	n.mu.Unlock()

	n.publish("ricart/request", 2, false, request)
	if shouldEnter {
		n.enterCS()
	}
}

func (n *testRicartNode) releaseCS() {
	n.mu.Lock()
	if n.state != ricartStateInCS {
		n.mu.Unlock()
		return
	}
	deferred := make([]string, 0, len(n.deferredReplies))
	for id := range n.deferredReplies {
		deferred = append(deferred, id)
	}
	n.deferredReplies = make(map[string]struct{})
	n.state = ricartStateIdle
	exitEvent := ricartCSEvent{
		ActorID:      n.id,
		ResourceID:   n.resourceID,
		LogicalClock: n.clock,
		Timestamp:    time.Now().UTC(),
	}
	n.mu.Unlock()

	sort.Strings(deferred)
	for _, id := range deferred {
		n.sendReply(id)
	}
	n.publish("ricart/cs/exit", 1, false, exitEvent)
}

func (n *testRicartNode) waitEntered(timeout time.Duration) {
	n.t.Helper()

	select {
	case <-n.entered:
	case <-time.After(timeout):
		n.t.Fatalf("node %q did not enter critical section within %s", n.id, timeout)
	}
}

func (n *testRicartNode) enterCS() {
	n.mu.Lock()
	event := ricartCSEvent{
		ActorID:      n.id,
		ResourceID:   n.resourceID,
		LogicalClock: n.clock,
		Timestamp:    time.Now().UTC(),
	}
	n.mu.Unlock()

	n.publish("ricart/cs/enter", 1, false, event)
	select {
	case n.entered <- struct{}{}:
	default:
	}
}

func (n *testRicartNode) sendReply(toID string) {
	n.mu.Lock()
	reply := ricartReply{
		FromID:       n.id,
		ToID:         toID,
		ResourceID:   n.resourceID,
		LogicalClock: n.tickLocked(),
	}
	n.mu.Unlock()

	n.publish("ricart/reply/"+toID, 2, false, reply)
}

func (n *testRicartNode) publishPresence(online bool) {
	n.publish("ricart/presence/"+n.id, 1, true, ricartPresence{
		ActorID:   n.id,
		ActorType: "integration_test_actor",
		Timestamp: time.Now().UTC(),
		Online:    online,
	})
}

func (n *testRicartNode) publish(topic string, qos byte, retained bool, payload any) {
	n.t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		n.t.Fatalf("marshal MQTT payload for %s: %v", topic, err)
	}
	token := n.client.Publish(topic, qos, retained, body)
	if !token.WaitTimeout(5 * time.Second) {
		n.t.Fatalf("publish MQTT message to %s timed out", topic)
	}
	if err := token.Error(); err != nil {
		n.t.Fatalf("publish MQTT message to %s: %v", topic, err)
	}
}

func (n *testRicartNode) disconnect() {
	if n.client != nil && n.client.IsConnected() {
		n.client.Disconnect(250)
	}
}

func (n *testRicartNode) hasPriorityLocked(incoming ricartRequest) bool {
	if n.requestClock != incoming.LogicalClock {
		return n.requestClock < incoming.LogicalClock
	}
	return n.id < incoming.FromID
}

func (n *testRicartNode) tryEnterCSLocked() bool {
	if n.state != ricartStateWanting || len(n.pendingReplies) != 0 {
		return false
	}
	n.state = ricartStateInCS
	return true
}

func (n *testRicartNode) tickLocked() uint64 {
	n.clock++
	return n.clock
}

func (n *testRicartNode) updateClockLocked(received uint64) {
	if received >= n.clock {
		n.clock = received + 1
		return
	}
	n.clock++
}

func assertRicartEventFlow(t *testing.T, events []ricartObservedStatusEvent, actorIDs []string) {
	t.Helper()

	for _, actorID := range actorIDs {
		requestIndex := eventIndex(events, actorID, "request")
		enterIndex := eventIndex(events, actorID, "cs_enter")
		exitIndex := eventIndex(events, actorID, "cs_exit")
		if requestIndex == -1 || enterIndex == -1 || exitIndex == -1 {
			t.Fatalf("expected request, cs_enter, and cs_exit for actor %q; events=%v", actorID, eventSummary(events))
		}
		if !(requestIndex < enterIndex && enterIndex < exitIndex) {
			t.Fatalf("expected actor %q event order request < cs_enter < cs_exit, got request=%d enter=%d exit=%d", actorID, requestIndex, enterIndex, exitIndex)
		}
		if events[requestIndex].LogicalClock == 0 {
			t.Fatalf("expected request logical_clock > 0 for actor %q", actorID)
		}
		if events[enterIndex].LogicalClock < events[requestIndex].LogicalClock {
			t.Fatalf("expected cs_enter logical_clock >= request clock for actor %q, got request=%d enter=%d", actorID, events[requestIndex].LogicalClock, events[enterIndex].LogicalClock)
		}
	}

	for _, event := range events {
		if event.Kind == "reply" {
			if event.PeerID == "" {
				t.Fatalf("expected reply event to contain peer_id: %+v", event)
			}
			if event.LogicalClock == 0 {
				t.Fatalf("expected reply logical_clock > 0: %+v", event)
			}
		}
	}
}

func assertMutualExclusion(t *testing.T, events []ricartObservedStatusEvent, actorIDs []string) {
	t.Helper()

	type interval struct {
		actorID string
		enter   time.Time
		exit    time.Time
	}

	intervals := make([]interval, 0, len(actorIDs))
	for _, actorID := range actorIDs {
		enter := eventByActorAndKind(events, actorID, "cs_enter")
		exit := eventByActorAndKind(events, actorID, "cs_exit")
		if !enter.Timestamp.Before(exit.Timestamp) && !enter.Timestamp.Equal(exit.Timestamp) {
			t.Fatalf("expected cs_enter timestamp <= cs_exit timestamp for actor %q, got enter=%s exit=%s", actorID, enter.Timestamp, exit.Timestamp)
		}
		intervals = append(intervals, interval{actorID: actorID, enter: enter.Timestamp, exit: exit.Timestamp})
	}

	for i := 0; i < len(intervals); i++ {
		for j := i + 1; j < len(intervals); j++ {
			a := intervals[i]
			b := intervals[j]
			overlaps := a.enter.Before(b.exit) && b.enter.Before(a.exit)
			if overlaps {
				t.Fatalf("critical sections overlap for %q and %q: %s-%s vs %s-%s", a.actorID, b.actorID, a.enter, a.exit, b.enter, b.exit)
			}
		}
	}
}

func assertRicartPriorityOrder(t *testing.T, events []ricartObservedStatusEvent) {
	t.Helper()

	requests := make([]ricartObservedStatusEvent, 0)
	enters := make([]ricartObservedStatusEvent, 0)
	for _, event := range events {
		switch event.Kind {
		case "request":
			requests = append(requests, event)
		case "cs_enter":
			enters = append(enters, event)
		}
	}

	sort.Slice(requests, func(i, j int) bool {
		if requests[i].LogicalClock != requests[j].LogicalClock {
			return requests[i].LogicalClock < requests[j].LogicalClock
		}
		return requests[i].ActorID < requests[j].ActorID
	})
	sort.Slice(enters, func(i, j int) bool {
		return enters[i].Timestamp.Before(enters[j].Timestamp)
	})

	if len(requests) != len(enters) {
		t.Fatalf("cannot compare priority order: requests=%d cs_enter=%d", len(requests), len(enters))
	}
	for i := range requests {
		if requests[i].ActorID != enters[i].ActorID {
			t.Fatalf("expected cs_enter order to follow Ricart priority (logical_clock, actor_id); request_order=%v enter_order=%v", actorOrder(requests), actorOrder(enters))
		}
	}
}

func controlCenterURL() string {
	if value := os.Getenv("CONTROL_CENTER_URL"); value != "" {
		return value
	}
	return "http://control-center:8080"
}

func mqttBrokerURL() string {
	if value := os.Getenv("MQTT_BROKER_URL"); value != "" {
		return value
	}
	host := os.Getenv("MQTT_BROKER_HOST")
	if host == "" {
		host = "mqtt-broker"
	}
	port := os.Getenv("MQTT_BROKER_PORT")
	if port == "" {
		port = "1883"
	}
	return fmt.Sprintf("tcp://%s:%s", host, port)
}

func waitForHealth(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
		if err != nil {
			t.Fatalf("create health request: %v", err)
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if ctx.Err() != nil {
			t.Fatalf("control center did not become healthy at %s within 30s", baseURL)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func getRicartStatus(t *testing.T, client *http.Client, baseURL string) ricartSnapshot {
	t.Helper()

	resp, err := client.Get(baseURL + "/ricart/status")
	if err != nil {
		t.Fatalf("GET /ricart/status failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read /ricart/status body: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /ricart/status returned HTTP %d, want %d, body=%q", resp.StatusCode, http.StatusOK, string(body))
	}

	var snapshot ricartSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		t.Fatalf("decode /ricart/status JSON: %v; body=%q", err, string(body))
	}
	return snapshot
}

func waitForStatus(t *testing.T, client *http.Client, baseURL string, timeout time.Duration, predicate func(ricartSnapshot) bool) ricartSnapshot {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var last ricartSnapshot
	for time.Now().Before(deadline) {
		last = getRicartStatus(t, client, baseURL)
		if predicate(last) {
			return last
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("ricart status did not match expectation within %s; participants=%d resources=%d events=%d", timeout, len(last.Participants), len(last.Resources), len(last.Events))
	return ricartSnapshot{}
}

func allParticipantsOnline(snapshot ricartSnapshot, actorIDs []string) bool {
	for _, actorID := range actorIDs {
		idx := slices.IndexFunc(snapshot.Participants, func(participant ricartParticipantState) bool {
			return participant.ActorID == actorID && participant.Online
		})
		if idx == -1 {
			return false
		}
	}
	return true
}

func eventsForResource(events []ricartObservedStatusEvent, resourceID string) []ricartObservedStatusEvent {
	filtered := make([]ricartObservedStatusEvent, 0, len(events))
	for _, event := range events {
		if event.ResourceID == resourceID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func countEvents(events []ricartObservedStatusEvent, kind string) int {
	count := 0
	for _, event := range events {
		if event.Kind == kind {
			count++
		}
	}
	return count
}

func findResource(resources []ricartResourceState, resourceID string) (ricartResourceState, bool) {
	for _, resource := range resources {
		if resource.ResourceID == resourceID {
			return resource, true
		}
	}
	return ricartResourceState{}, false
}

func eventIndex(events []ricartObservedStatusEvent, actorID, kind string) int {
	for idx, event := range events {
		if event.ActorID == actorID && event.Kind == kind {
			return idx
		}
	}
	return -1
}

func eventByActorAndKind(events []ricartObservedStatusEvent, actorID, kind string) ricartObservedStatusEvent {
	for _, event := range events {
		if event.ActorID == actorID && event.Kind == kind {
			return event
		}
	}
	return ricartObservedStatusEvent{}
}

func actorOrder(events []ricartObservedStatusEvent) []string {
	actors := make([]string, 0, len(events))
	for _, event := range events {
		actors = append(actors, event.ActorID)
	}
	return actors
}

func eventSummary(events []ricartObservedStatusEvent) []string {
	summary := make([]string, 0, len(events))
	for _, event := range events {
		summary = append(summary, fmt.Sprintf("%s:%s:%d", event.Kind, event.ActorID, event.LogicalClock))
	}
	return summary
}
