package html

import (
	"control-center/internal/model"
	"fmt"
	"sort"
	"strings"
	"time"

	"shared"
)

type RicartPanelData struct {
	UpdatedAt    time.Time               `json:"updated_at"`
	Overview     RicartOverview          `json:"overview"`
	Bridge       RicartResourceCard      `json:"bridge"`
	Resources    []RicartResourceCard    `json:"resources"`
	Participants []RicartParticipantCard `json:"participants"`
	Events       []RicartEventCard       `json:"events"`
}

type RicartOverview struct {
	OnlineCount        int `json:"online_count"`
	WantingCount       int `json:"wanting_count"`
	InCSCount          int `json:"in_cs_count"`
	IdleCount          int `json:"idle_count"`
	ResourceCount      int `json:"resource_count"`
	ActiveRequestCount int `json:"active_request_count"`
}

type RicartResourceCard struct {
	ResourceID     string `json:"resource_id"`
	Label          string `json:"label"`
	State          string `json:"state"`
	Tone           string `json:"tone"`
	Holder         string `json:"holder,omitempty"`
	Waiting        string `json:"waiting,omitempty"`
	Requesters     string `json:"requesters,omitempty"`
	LastChange     string `json:"last_change"`
	LastLogicalClk string `json:"last_logical_clock,omitempty"`
}

type RicartParticipantCard struct {
	ActorID      string `json:"actor_id"`
	Online       bool   `json:"online"`
	State        string `json:"state"`
	ResourceID   string `json:"resource_id,omitempty"`
	RequestClock string `json:"request_clock,omitempty"`
	Pending      string `json:"pending,omitempty"`
	Replies      string `json:"replies,omitempty"`
	LastSeen     string `json:"last_seen"`
	LastEvent    string `json:"last_event,omitempty"`
}

type RicartEventCard struct {
	Kind         string `json:"kind"`
	ActorID      string `json:"actor_id"`
	PeerID       string `json:"peer_id,omitempty"`
	ResourceID   string `json:"resource_id"`
	LogicalClock string `json:"logical_clock"`
	Timestamp    string `json:"timestamp"`
	Detail       string `json:"detail,omitempty"`
}

type RicartTileBadge struct {
	Label string
	Tone  string
	Hint  string
}

func NewRicartPanelData(snapshot model.RicartSnapshot) RicartPanelData {
	participants := ricartParticipants(snapshot.Participants)
	resources := ricartResources(snapshot.Resources)
	events := ricartEvents(snapshot.Events)

	bridge := RicartResourceCard{
		ResourceID: "bridge",
		Label:      "bridge",
		State:      "idle",
		Tone:       "normal",
	}
	if len(resources) > 0 {
		if resource, ok := bridgeResource(resources); ok {
			bridge = resource
		}
	}

	return RicartPanelData{
		UpdatedAt:    snapshot.UpdatedAt,
		Overview:     ricartOverview(snapshot),
		Bridge:       bridge,
		Resources:    resources,
		Participants: participants,
		Events:       events,
	}
}

func ricartOverview(snapshot model.RicartSnapshot) RicartOverview {
	var overview RicartOverview
	overview.ResourceCount = len(snapshot.Resources)

	for _, participant := range snapshot.Participants {
		if participant.Online {
			overview.OnlineCount++
		}
		switch participant.State {
		case shared.RicartActorStateWanting:
			overview.WantingCount++
		case shared.RicartActorStateInCS:
			overview.InCSCount++
		default:
			overview.IdleCount++
		}
		if len(participant.PendingReplies) > 0 {
			overview.ActiveRequestCount++
		}
	}

	return overview
}

func ricartResources(resources []model.RicartResourceState) []RicartResourceCard {
	cards := make([]RicartResourceCard, 0, len(resources))
	for _, resource := range resources {
		cards = append(cards, ricartResourceCard(resource))
	}
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].ResourceID < cards[j].ResourceID
	})
	return cards
}

func ricartResourceCard(resource model.RicartResourceState) RicartResourceCard {
	waiting := strings.Join(resource.WaitingActors, ", ")
	requesters := strings.Join(resource.Requesters, ", ")
	holder := ""
	if resource.HolderActorID != "" {
		holder = fallbackID(resource.HolderActorID)
	}
	lastChange := resource.LastRequestAt
	if resource.LastEnterAt.After(lastChange) {
		lastChange = resource.LastEnterAt
	}
	if resource.LastExitAt.After(lastChange) {
		lastChange = resource.LastExitAt
	}
	return RicartResourceCard{
		ResourceID:     resource.ResourceID,
		Label:          resource.Label,
		State:          string(resource.Status),
		Tone:           ricartTone(resource.Status),
		Holder:         holder,
		Waiting:        waiting,
		Requesters:     requesters,
		LastChange:     formatTimestamp(lastChange),
		LastLogicalClk: fmt.Sprintf("%d", resource.LastLogicalClk),
	}
}

func ricartParticipants(participants []model.RicartParticipantState) []RicartParticipantCard {
	cards := make([]RicartParticipantCard, 0, len(participants))
	for _, participant := range participants {
		cards = append(cards, RicartParticipantCard{
			ActorID:      fallbackID(participant.ActorID),
			Online:       participant.Online,
			State:        strings.ToLower(string(participant.State)),
			ResourceID:   ricartResourceIDLabel(participant.ResourceID),
			RequestClock: clockLabel(participant.RequestClock),
			Pending:      joinOrEmpty(participant.PendingReplies),
			Replies:      joinOrEmpty(participant.Replies),
			LastSeen:     formatTimestamp(participant.LastSeen),
			LastEvent:    participant.LastEvent,
		})
	}
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].ActorID < cards[j].ActorID
	})
	return cards
}

func ricartEvents(events []model.RicartEvent) []RicartEventCard {
	count := len(events)
	if count > 8 {
		count = 8
	}
	cards := make([]RicartEventCard, 0, count)
	for i := len(events) - 1; i >= 0 && len(cards) < count; i-- {
		event := events[i]
		peerID := ""
		if event.PeerID != "" {
			peerID = fallbackID(event.PeerID)
		}
		cards = append(cards, RicartEventCard{
			Kind:         event.Kind,
			ActorID:      fallbackID(event.ActorID),
			PeerID:       peerID,
			ResourceID:   ricartResourceIDLabel(event.ResourceID),
			LogicalClock: clockLabel(event.LogicalClock),
			Timestamp:    formatTimestamp(event.Timestamp),
			Detail:       event.Detail,
		})
	}
	return cards
}

func bridgeResource(resources []RicartResourceCard) (RicartResourceCard, bool) {
	for _, resource := range resources {
		if strings.EqualFold(resource.ResourceID, "bridge") {
			return resource, true
		}
	}
	if len(resources) > 0 {
		return resources[0], true
	}
	return RicartResourceCard{}, false
}

func RicartTileBadgeFromSnapshot(snapshot model.RicartSnapshot) RicartTileBadge {
	bridge, ok := bridgeResource(ricartResources(snapshot.Resources))
	if !ok {
		return RicartTileBadge{}
	}

	state := bridge.State
	if state == "" {
		state = "idle"
	}
	label := fmt.Sprintf("bridge %s", state)
	holder := bridge.Holder
	if holder == "" {
		holder = "none"
	}
	hint := fmt.Sprintf("holder=%s waiting=%s", holder, bridge.Waiting)
	if bridge.Waiting == "" {
		hint = fmt.Sprintf("holder=%s", holder)
	}
	return RicartTileBadge{
		Label: label,
		Tone:  bridge.Tone,
		Hint:  hint,
	}
}

func ricartTone(state model.RicartResourceStatus) string {
	switch state {
	case model.ResourceStatusOccupied:
		return "critical"
	case model.ResourceStatusWanted:
		return "warning"
	default:
		return "normal"
	}
}

func ricartResourceLabel(resourceID string) string {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "bridge"
	}
	return strings.ReplaceAll(resourceID, "_", " ")
}

func ricartResourceIDLabel(resourceID string) string {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ""
	}
	return ricartResourceLabel(resourceID)
}

func clockLabel(clock uint64) string {
	if clock == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", clock)
}

func joinOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, ", ")
}
