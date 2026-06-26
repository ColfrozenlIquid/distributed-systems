package model

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"shared"

	"github.com/samber/ro"
)

type RicartStatus struct {
	mu        sync.RWMutex
	actors    map[string]*RicartParticipant
	resources map[string]*RicartResource
	events    []RicartEvent
	updates   ro.Subject[RicartSnapshot]
}

type RicartParticipant struct {
	ActorID        string
	Online         bool
	State          shared.RicartActorState
	ResourceID     string
	RequestClock   uint64
	RequestedAt    time.Time
	PendingReplies map[string]struct{}
	Replies        map[string]struct{}
	LastSeen       time.Time
	EnteredAt      time.Time
	ExitedAt       time.Time
	LastEvent      string
}

type RicartResourceStatus string

const (
	ResourceStatusFree     = "free"
	ResourceStatusWanted   = "wanted"
	ResourceStatusOccupied = "occupied"
)

type RicartResource struct {
	ResourceID     string
	HolderActorID  string
	WaitingActors  map[string]struct{}
	Requesters     map[string]struct{}
	Status         RicartResourceStatus
	LastRequestAt  time.Time
	LastEnterAt    time.Time
	LastExitAt     time.Time
	LastLogicalClk uint64
	LastEvent      string
}

type RicartEvent struct {
	Kind         string    `json:"kind"`
	ActorID      string    `json:"actor_id"`
	PeerID       string    `json:"peer_id,omitempty"`
	ResourceID   string    `json:"resource_id"`
	LogicalClock uint64    `json:"logical_clock"`
	Timestamp    time.Time `json:"timestamp"`
	Detail       string    `json:"detail,omitempty"`
}

type RicartSnapshot struct {
	UpdatedAt    time.Time                `json:"updated_at"`
	Participants []RicartParticipantState `json:"participants"`
	Resources    []RicartResourceState    `json:"resources"`
	Events       []RicartEvent            `json:"events"`
}

type RicartParticipantState struct {
	ActorID        string                  `json:"actor_id"`
	Online         bool                    `json:"online"`
	State          shared.RicartActorState `json:"state"`
	ResourceID     string                  `json:"resource_id,omitempty"`
	RequestClock   uint64                  `json:"request_clock,omitempty"`
	RequestedAt    time.Time               `json:"requested_at,omitempty"`
	PendingReplies []string                `json:"pending_replies,omitempty"`
	Replies        []string                `json:"replies,omitempty"`
	LastSeen       time.Time               `json:"last_seen"`
	EnteredAt      time.Time               `json:"entered_at,omitempty"`
	ExitedAt       time.Time               `json:"exited_at,omitempty"`
	LastEvent      string                  `json:"last_event,omitempty"`
}

type RicartResourceState struct {
	ResourceID     string               `json:"resource_id"`
	Label          string               `json:"label"`
	Status         RicartResourceStatus `json:"status"`
	HolderActorID  string               `json:"holder_actor_id,omitempty"`
	WaitingActors  []string             `json:"waiting_actors,omitempty"`
	Requesters     []string             `json:"requesters,omitempty"`
	LastRequestAt  time.Time            `json:"last_request_at,omitempty"`
	LastEnterAt    time.Time            `json:"last_enter_at,omitempty"`
	LastExitAt     time.Time            `json:"last_exit_at,omitempty"`
	LastLogicalClk uint64               `json:"last_logical_clock,omitempty"`
	LastEvent      string               `json:"last_event,omitempty"`
}

func NewRicartStatus() RicartStatus {
	return RicartStatus{
		actors:    make(map[string]*RicartParticipant),
		resources: make(map[string]*RicartResource),
		events:    make([]RicartEvent, 0, 32),
		updates:   ro.NewPublishSubject[RicartSnapshot](),
	}
}

func (s *RicartStatus) Updates() ro.Observable[RicartSnapshot] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.updates == nil {
		s.updates = ro.NewPublishSubject[RicartSnapshot]()
	}
	return s.updates
}

func (s *RicartStatus) HandlePresence(presence shared.RicartPresence) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.ensureParticipantLocked(presence.ActorID)
	previousResourceID := participant.ResourceID
	participant.Online = presence.Online
	participant.LastSeen = presence.Timestamp
	participant.LastEvent = presenceEventLabel(presence.Online)

	if presence.Online {
		s.recordEventLocked(RicartEvent{
			Kind:      "presence_online",
			ActorID:   presence.ActorID,
			Timestamp: presence.Timestamp,
		})
		s.publishSnapshotLocked()
		return
	}

	// only executed if not online
	if participant.State != shared.RicartActorStateIdle {
		s.resetParticipantRequestLocked(participant)
		participant.State = shared.RicartActorStateIdle
		participant.EnteredAt = time.Time{}
		participant.ExitedAt = presence.Timestamp
	}
	if resource := s.resourceLocked(previousResourceID); resource != nil && resource.HolderActorID == participant.ActorID {
		resource.HolderActorID = ""
		resource.LastExitAt = presence.Timestamp
		resource.LastEvent = "presence_offline"
		resource.refreshResourceStateLocked()
	}
	s.recordEventLocked(RicartEvent{
		Kind:      "presence_offline",
		ActorID:   presence.ActorID,
		Timestamp: presence.Timestamp,
	})
	s.publishSnapshotLocked()
}

func (s *RicartStatus) HandleRequest(request shared.RicartRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.ensureParticipantLocked(request.FromID)
	participant.Online = true
	participant.State = shared.RicartActorStateWanting
	participant.ResourceID = request.ResourceID
	participant.RequestClock = request.LogicalClock
	participant.RequestedAt = request.Timestamp
	participant.LastSeen = request.Timestamp
	participant.LastEvent = "request"
	participant.PendingReplies = make(map[string]struct{})
	participant.Replies = make(map[string]struct{})

	for id, other := range s.actors {
		if id == request.FromID || !other.Online {
			continue
		}
		participant.PendingReplies[id] = struct{}{}
	}

	resource := s.ensureResourceLocked(request.ResourceID)
	resource.LastRequestAt = request.Timestamp
	resource.LastLogicalClk = request.LogicalClock
	resource.LastEvent = "request"
	resource.WaitingActors[request.FromID] = struct{}{}
	resource.Requesters[request.FromID] = struct{}{}
	resource.refreshResourceStateLocked()

	s.recordEventLocked(RicartEvent{
		Kind:         "request",
		ActorID:      request.FromID,
		ResourceID:   request.ResourceID,
		LogicalClock: request.LogicalClock,
		Timestamp:    request.Timestamp,
		Detail:       fmt.Sprintf("pending=%d", len(participant.PendingReplies)),
	})
	s.publishSnapshotLocked()
}

func (s *RicartStatus) HandleReply(reply shared.RicartReply) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.ensureParticipantLocked(reply.ToID)
	participant.Online = true
	participant.LastSeen = time.Now()
	participant.LastEvent = "reply_received"
	if participant.PendingReplies != nil {
		delete(participant.PendingReplies, reply.FromID)
	}
	if participant.Replies == nil {
		participant.Replies = make(map[string]struct{})
	}
	participant.Replies[reply.FromID] = struct{}{}

	if resource := s.resourceLocked(reply.ResourceID); resource != nil {
		delete(resource.Requesters, reply.ToID)
		resource.LastLogicalClk = reply.LogicalClock
		resource.LastEvent = "reply"
		resource.refreshResourceStateLocked()
	}

	s.recordEventLocked(RicartEvent{
		Kind:         "reply",
		ActorID:      reply.ToID,
		PeerID:       reply.FromID,
		ResourceID:   reply.ResourceID,
		LogicalClock: reply.LogicalClock,
		Timestamp:    time.Now(),
	})
	s.publishSnapshotLocked()
}

func (s *RicartStatus) HandleCSEnter(event shared.RicartCsEvent) {
	s.handleCSStateLocked(event, shared.RicartActorStateInCS, "cs_enter")
}

func (s *RicartStatus) HandleCSExit(event shared.RicartCsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.ensureParticipantLocked(event.ActorID)
	participant.Online = true
	participant.State = shared.RicartActorStateIdle
	participant.ResourceID = ""
	participant.RequestClock = 0
	participant.RequestedAt = time.Time{}
	participant.PendingReplies = nil
	participant.Replies = nil
	participant.ExitedAt = event.Timestamp
	participant.LastSeen = event.Timestamp
	participant.LastEvent = "cs_exit"

	resource := s.ensureResourceLocked(event.ResourceID)
	if resource.HolderActorID == event.ActorID {
		resource.HolderActorID = ""
	}
	delete(resource.WaitingActors, event.ActorID)
	delete(resource.Requesters, event.ActorID)
	resource.LastExitAt = event.Timestamp
	resource.LastLogicalClk = event.LogicalClock
	resource.LastEvent = "cs_exit"
	resource.refreshResourceStateLocked()

	s.recordEventLocked(RicartEvent{
		Kind:         "cs_exit",
		ActorID:      event.ActorID,
		ResourceID:   event.ResourceID,
		LogicalClock: event.LogicalClock,
		Timestamp:    event.Timestamp,
	})
	s.publishSnapshotLocked()
}

func (s *RicartStatus) handleCSStateLocked(event shared.RicartCsEvent, state shared.RicartActorState, kind string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	participant := s.ensureParticipantLocked(event.ActorID)
	participant.Online = true
	participant.State = state
	participant.ResourceID = event.ResourceID
	participant.LastSeen = event.Timestamp
	participant.LastEvent = kind
	if state == shared.RicartActorStateInCS {
		participant.EnteredAt = event.Timestamp
	}

	resource := s.ensureResourceLocked(event.ResourceID)
	resource.HolderActorID = event.ActorID
	resource.LastLogicalClk = event.LogicalClock
	resource.LastEvent = kind
	resource.LastEnterAt = event.Timestamp
	delete(resource.WaitingActors, event.ActorID)
	delete(resource.Requesters, event.ActorID)
	resource.refreshResourceStateLocked()

	s.recordEventLocked(RicartEvent{
		Kind:         kind,
		ActorID:      event.ActorID,
		ResourceID:   event.ResourceID,
		LogicalClock: event.LogicalClock,
		Timestamp:    event.Timestamp,
	})
	s.publishSnapshotLocked()
}

func (s *RicartStatus) Snapshot() RicartSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	participants := make([]RicartParticipantState, 0, len(s.actors))
	for _, participant := range s.actors {
		participants = append(participants, participant.snapshot())
	}
	sort.Slice(participants, func(i, j int) bool {
		return participants[i].ActorID < participants[j].ActorID
	})

	resources := make([]RicartResourceState, 0, len(s.resources))
	for _, resource := range s.resources {
		resources = append(resources, resource.snapshot())
	}
	sort.Slice(resources, func(i, j int) bool {
		return resources[i].ResourceID < resources[j].ResourceID
	})

	events := append([]RicartEvent(nil), s.events...)
	return RicartSnapshot{
		UpdatedAt:    time.Now(),
		Participants: participants,
		Resources:    resources,
		Events:       events,
	}
}

func (s *RicartStatus) ensureParticipantLocked(actorID string) *RicartParticipant {
	if participant, ok := s.actors[actorID]; ok {
		return participant
	}
	participant := &RicartParticipant{ActorID: actorID}
	s.actors[actorID] = participant
	return participant
}

func (s *RicartStatus) ensureResourceLocked(resourceID string) *RicartResource {
	resource := s.resourceLocked(resourceID)
	if resource != nil {
		return resource
	}
	resource = &RicartResource{
		ResourceID:    resourceID,
		WaitingActors: make(map[string]struct{}),
		Requesters:    make(map[string]struct{}),
		Status:        ResourceStatusFree,
	}
	s.resources[resourceID] = resource
	return resource
}

func (s *RicartStatus) resourceLocked(resourceID string) *RicartResource {
	if resourceID == "" {
		resourceID = "bridge"
	}
	resource, ok := s.resources[resourceID]
	if !ok {
		return nil
	}
	return resource
}

func (s *RicartStatus) recordEventLocked(event RicartEvent) {
	event.Timestamp = event.Timestamp.UTC()
	s.events = append(s.events, event)
	if len(s.events) > 50 {
		s.events = append([]RicartEvent(nil), s.events[len(s.events)-50:]...)
	}
	if resource := s.resourceLocked(event.ResourceID); resource != nil {
		resource.LastEvent = event.Kind
	}
}

func (s *RicartStatus) publishSnapshotLocked() {
	if s.updates == nil {
		return
	}
	s.updates.Next(s.Snapshot())
}

func (r *RicartResource) refreshResourceStateLocked() {
	if r == nil {
		return
	}
	if r.HolderActorID != "" {
		r.Status = ResourceStatusOccupied
		return
	}
	if len(r.WaitingActors) > 0 || len(r.Requesters) > 0 {
		r.Status = ResourceStatusWanted
		return
	}
	r.Status = ResourceStatusFree
}

func (p *RicartParticipant) snapshot() RicartParticipantState {
	pending := make([]string, 0, len(p.PendingReplies))
	for id := range p.PendingReplies {
		pending = append(pending, id)
	}
	sort.Strings(pending)

	replies := make([]string, 0, len(p.Replies))
	for id := range p.Replies {
		replies = append(replies, id)
	}
	sort.Strings(replies)

	return RicartParticipantState{
		ActorID:        p.ActorID,
		Online:         p.Online,
		State:          p.State,
		ResourceID:     p.ResourceID,
		RequestClock:   p.RequestClock,
		RequestedAt:    p.RequestedAt,
		PendingReplies: pending,
		Replies:        replies,
		LastSeen:       p.LastSeen,
		EnteredAt:      p.EnteredAt,
		ExitedAt:       p.ExitedAt,
		LastEvent:      p.LastEvent,
	}
}

func (r *RicartResource) snapshot() RicartResourceState {
	waiting := make([]string, 0, len(r.WaitingActors))
	for id := range r.WaitingActors {
		waiting = append(waiting, id)
	}
	sort.Strings(waiting)

	requesters := make([]string, 0, len(r.Requesters))
	for id := range r.Requesters {
		requesters = append(requesters, id)
	}
	sort.Strings(requesters)

	return RicartResourceState{
		ResourceID:     r.ResourceID,
		Label:          resourceLabel(r.ResourceID),
		Status:         r.Status,
		HolderActorID:  r.HolderActorID,
		WaitingActors:  waiting,
		Requesters:     requesters,
		LastRequestAt:  r.LastRequestAt,
		LastEnterAt:    r.LastEnterAt,
		LastExitAt:     r.LastExitAt,
		LastLogicalClk: r.LastLogicalClk,
		LastEvent:      r.LastEvent,
	}
}

func (s *RicartStatus) resetParticipantRequestLocked(participant *RicartParticipant) {
	if participant == nil {
		return
	}
	participant.PendingReplies = nil
	participant.Replies = nil
	participant.RequestClock = 0
	participant.RequestedAt = time.Time{}
	participant.ResourceID = ""
}

func presenceEventLabel(online bool) string {
	if online {
		return "presence_online"
	}
	return "presence_offline"
}

func resourceLabel(resourceID string) string {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "bridge"
	}
	return strings.ReplaceAll(resourceID, "_", " ")
}
