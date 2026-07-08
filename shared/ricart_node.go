package shared

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/samber/ro"
)

const (
	defaultRicartPeerTTL          = 30 * time.Second
	defaultRicartPresenceInterval = 10 * time.Second
	defaultRicartResourceID       = "charging-station"
)

type RicartNode struct {
	mu sync.Mutex

	id         string
	actorType  ActorType
	resourceID string

	clock        uint64
	state        RicartActorState
	requestClock uint64

	peers           map[string]time.Time
	pendingReplies  map[string]struct{}
	deferredReplies map[string]struct{}

	peerTTL          time.Duration
	presenceInterval time.Duration

	mqttClient    mqtt.Client
	subscriptions []ro.Subscription
	cancel        context.CancelFunc
	onEnterCS     func()
}

func NewRicartNode(id string, actorType ActorType, resourceID string, client mqtt.Client, onEnterCS func()) *RicartNode {
	if resourceID == "" {
		resourceID = defaultRicartResourceID
	}
	if onEnterCS == nil {
		onEnterCS = func() {}
	}

	return &RicartNode{
		id:               id,
		actorType:        actorType,
		resourceID:       resourceID,
		state:            RicartActorStateIdle,
		peers:            make(map[string]time.Time),
		pendingReplies:   make(map[string]struct{}),
		deferredReplies:  make(map[string]struct{}),
		peerTTL:          defaultRicartPeerTTL,
		presenceInterval: defaultRicartPresenceInterval,
		mqttClient:       client,
		onEnterCS:        onEnterCS,
	}
}

func ChargingStationResourceID(position Position) string {
	return fmt.Sprintf("charging-station:%d:%d", position.X, position.Y)
}

func (n *RicartNode) Start(ctx context.Context, streams *RicartStreams) error {
	if streams == nil {
		return fmt.Errorf("ricart streams are nil")
	}

	nodeCtx, cancel := context.WithCancel(ctx)
	n.cancel = cancel
	n.subscriptions = append(n.subscriptions,
		streams.Presence().SubscribeWithContext(nodeCtx, ro.NewObserver(
			n.handlePresence,
			func(error) {},
			func() {},
		)),
		streams.Requests().SubscribeWithContext(nodeCtx, ro.NewObserver(
			n.handleRequest,
			func(error) {},
			func() {},
		)),
		streams.Replies().SubscribeWithContext(nodeCtx, ro.NewObserver(
			n.handleReply,
			func(error) {},
			func() {},
		)),
	)

	if err := streams.Subscribe(n.mqttClient); err != nil {
		return fmt.Errorf("subscribe ricart streams: %w", err)
	}

	if err := n.publishPresence(true); err != nil {
		return err
	}

	go n.runMaintenance(nodeCtx)
	return nil
}

func (n *RicartNode) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
	for _, subscription := range n.subscriptions {
		subscription.Unsubscribe()
	}
	_ = n.publishPresence(false)
}

func (n *RicartNode) RequestCS() error {
	n.mu.Lock()
	if n.state == RicartActorStateInCS || n.state == RicartActorStateWanting {
		n.mu.Unlock()
		return nil
	}

	n.state = RicartActorStateWanting
	n.requestClock = n.tickLocked()
	n.pendingReplies = make(map[string]struct{}, len(n.peers))
	for id := range n.peers {
		n.pendingReplies[id] = struct{}{}
	}

	request := n.requestMessageLocked()
	shouldEnter := len(n.pendingReplies) == 0
	if shouldEnter {
		n.state = RicartActorStateInCS
	}
	n.mu.Unlock()

	if err := n.publishRequest(request); err != nil {
		return err
	}

	if shouldEnter {
		n.enterCS()
	}
	return nil
}

func (n *RicartNode) ReleaseCS() error {
	n.mu.Lock()
	if n.state != RicartActorStateInCS {
		n.mu.Unlock()
		return nil
	}

	deferred := make([]string, 0, len(n.deferredReplies))
	for id := range n.deferredReplies {
		deferred = append(deferred, id)
	}
	n.deferredReplies = make(map[string]struct{})
	n.state = RicartActorStateIdle

	exitEvent := n.csEventLocked()
	n.mu.Unlock()

	var wg sync.WaitGroup
	errs := make([]error, len(deferred))
	for i, id := range deferred {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			errs[i] = n.sendReply(id)
		}(i, id)
	}
	wg.Wait()

	var firstErr error
	for _, err := range errs {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if err := MqttPublish(n.mqttClient, RicartCsExitParams(exitEvent)); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (n *RicartNode) handlePresence(msg RicartPresence) {
	if msg.ActorID == n.id {
		return
	}

	var request *RicartRequest
	var shouldEnter bool

	n.mu.Lock()
	if msg.Online {
		_, known := n.peers[msg.ActorID]
		n.peers[msg.ActorID] = time.Now()
		if !known && n.state == RicartActorStateWanting {
			n.pendingReplies[msg.ActorID] = struct{}{}
			request = new(n.requestMessageLocked())
		}
	} else {
		shouldEnter = n.removePeerLocked(msg.ActorID)
	}
	n.mu.Unlock()

	if request != nil {
		_ = n.publishRequest(*request)
	}
	if shouldEnter {
		n.enterCS()
	}
}

func (n *RicartNode) handleRequest(msg RicartRequest) {
	if msg.FromID == n.id || msg.ResourceID != n.resourceID {
		return
	}

	shouldReply := false

	n.mu.Lock()
	n.updateClockLocked(msg.LogicalClock)
	n.peers[msg.FromID] = time.Now()

	shouldDefer := n.state == RicartActorStateInCS ||
		(n.state == RicartActorStateWanting && n.hasPriorityLocked(msg))
	if shouldDefer {
		n.deferredReplies[msg.FromID] = struct{}{}
	} else {
		shouldReply = true
	}
	n.mu.Unlock()

	if shouldReply {
		_ = n.sendReply(msg.FromID)
	}
}

func (n *RicartNode) handleReply(msg RicartReply) {
	if msg.ToID != n.id || msg.ResourceID != n.resourceID {
		return
	}

	var shouldEnter bool

	n.mu.Lock()
	n.updateClockLocked(msg.LogicalClock)
	delete(n.pendingReplies, msg.FromID)
	shouldEnter = n.tryEnterCSLocked()
	n.mu.Unlock()

	if shouldEnter {
		n.enterCS()
	}
}

func (n *RicartNode) runMaintenance(ctx context.Context) {
	ticker := time.NewTicker(n.presenceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = n.publishPresence(true)
			if request := n.pendingRequest(); request != nil {
				_ = n.publishRequest(*request)
			}
			if n.reapDeadPeers() {
				n.enterCS()
			}
		}
	}
}

func (n *RicartNode) pendingRequest() *RicartRequest {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != RicartActorStateWanting || len(n.pendingReplies) == 0 {
		return nil
	}

	return new(n.requestMessageLocked())
}

func (n *RicartNode) reapDeadPeers() bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	shouldEnter := false
	for id, lastSeen := range n.peers {
		if time.Since(lastSeen) > n.peerTTL {
			if n.removePeerLocked(id) {
				shouldEnter = true
			}
		}
	}
	return shouldEnter
}

func (n *RicartNode) removePeerLocked(id string) bool {
	delete(n.peers, id)
	delete(n.pendingReplies, id)
	delete(n.deferredReplies, id)
	return n.tryEnterCSLocked()
}

func (n *RicartNode) tryEnterCSLocked() bool {
	if n.state != RicartActorStateWanting || len(n.pendingReplies) != 0 {
		return false
	}
	n.state = RicartActorStateInCS
	return true
}

func (n *RicartNode) enterCS() {
	n.mu.Lock()
	event := n.csEventLocked()
	n.mu.Unlock()

	_ = MqttPublish(n.mqttClient, RicartCsEnterParams(event))
	go n.onEnterCS()
}

func (n *RicartNode) hasPriorityLocked(incoming RicartRequest) bool {
	if n.requestClock != incoming.LogicalClock {
		return n.requestClock < incoming.LogicalClock
	}
	return n.id < incoming.FromID
}

func (n *RicartNode) sendReply(toID string) error {
	n.mu.Lock()
	clock := n.tickLocked()
	reply := RicartReply{
		FromID:       n.id,
		ToID:         toID,
		ResourceID:   n.resourceID,
		LogicalClock: clock,
	}
	n.mu.Unlock()

	return MqttPublish(n.mqttClient, RicartReplyParams(toID, reply))
}

func (n *RicartNode) publishPresence(online bool) error {
	presence := RicartPresence{
		ActorID:   n.id,
		ActorType: n.actorType,
		Timestamp: time.Now(),
		Online:    online,
	}
	return MqttPublish(n.mqttClient, RicartPresenceParams(n.id, presence))
}

func (n *RicartNode) publishRequest(request RicartRequest) error {
	return MqttPublish(n.mqttClient, RicartRequestParams(request))
}

func (n *RicartNode) requestMessageLocked() RicartRequest {
	return RicartRequest{
		FromID:       n.id,
		ResourceID:   n.resourceID,
		LogicalClock: n.requestClock,
		Timestamp:    time.Now(),
	}
}

func (n *RicartNode) csEventLocked() RicartCsEvent {
	return RicartCsEvent{
		ActorID:      n.id,
		ResourceID:   n.resourceID,
		LogicalClock: n.clock,
		Timestamp:    time.Now(),
	}
}

func (n *RicartNode) tickLocked() uint64 {
	n.clock++
	return n.clock
}

func (n *RicartNode) updateClockLocked(received uint64) {
	if received >= n.clock {
		n.clock = received + 1
		return
	}
	n.clock++
}
