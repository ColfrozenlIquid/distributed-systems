package shared

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Node struct {
	mu sync.Mutex

	id         string
	resourceID string

	// Lamport-Uhr
	clock uint64

	// Algorithmus-Zustand
	state        RicartActorState
	requestClock uint64 // Uhrwert beim eigenen Request

	// Membership
	peers   map[string]time.Time // actor_id -> lastSeen
	peerTTL time.Duration        // z.B. 30s; danach gilt Peer als tot

	// Deferred Replies: IDs, auf deren REPLY wir noch warten
	pendingReplies map[string]struct{}

	// Deferred Requests: Requests die wir zurückgehalten haben
	deferredReplies []string // actor_ids, denen wir noch antworten müssen

	mqttClient mqtt.Client
	onEnterCS  func() // Callback wenn CS betreten werden kann
}

func (n *Node) tick() uint64 {
	n.clock++
	return n.clock
}

// Bei eingehender Nachricht mit fremdem Timestamp
func (n *Node) updateClock(received uint64) {
	if received >= n.clock {
		n.clock = received + 1
	} else {
		n.clock++
	}
}

// Aufgerufen wenn eine RicartPresence eingeht (ricart/presence/+)
func (n *Node) handlePresence(msg RicartPresence) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if msg.ActorID == n.id {
		return // eigene Nachricht ignorieren
	}

	if msg.Online {
		_, wasUnknown := n.peers[msg.ActorID]
		n.peers[msg.ActorID] = time.Now()

		if wasUnknown {
			// Neuer Peer: falls wir gerade WANTING sind,
			// müssen wir auch von ihm ein Reply bekommen.
			// Das geht nur, wenn er unseren Request noch nicht gesehen hat —
			// daher: wir senden nochmal einen Request ODER
			// akzeptieren, dass er unseren Request noch erhalten wird
			// (da wir retain=false nutzen, ist ein Re-Request nötig).
			// Einfachste Lösung: Pending-Reply hinzufügen und warten.
			if n.state == RicartActorStateWanting {
				n.pendingReplies[msg.ActorID] = struct{}{}
				// Der neue Peer wird unseren Request erhalten,
				// da wir ihn gleich nach Request-Broadcast senden.
				// Aber: Timing-Problem! Lösung: Re-Broadcast bei neuem Peer.
				go n.broadcastRequest() // nochmal senden
			}
		}
	} else {
		// Graceful leave: Peer aus Membership entfernen
		n.removePeer(msg.ActorID)
	}
}

// Periodisch aufgerufen um tote Peers zu entfernen
func (n *Node) reapDeadPeers() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for id, lastSeen := range n.peers {
		if time.Since(lastSeen) > n.peerTTL {
			n.removePeer(id)
		}
	}
}

// Muss unter n.mu aufgerufen werden
func (n *Node) removePeer(id string) {
	delete(n.peers, id)
	// Falls wir auf diesen Peer warten: Reply als erhalten werten
	// (Crash-Annahme: toter Peer gibt CS frei)
	delete(n.pendingReplies, id)
	n.tryEnterCS()
}

func (n *Node) peerCount() int {
	return len(n.peers)
}

func (n *Node) RequestCS() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.state = RicartActorStateWanting
	n.requestClock = n.tick() // tick() MUSS vor broadcastRequest() stehen,
	// damit requestClock den Sendezeitpunkt widerspiegelt

	n.pendingReplies = make(map[string]struct{})
	for id := range n.peers {
		n.pendingReplies[id] = struct{}{}
	}

	n.broadcastRequest()
}

// broadcastRequest muss requestClock verwenden, nicht tick() nochmal aufrufen.
// Das war bereits korrekt — aber beim Re-Broadcast bei Late-Joiner (handlePresence)
// darf die Uhr NICHT nochmal getickt werden, sonst ändert sich der requestClock:
func (n *Node) broadcastRequest() {
	// requestClock wurde einmalig bei RequestCS() eingefroren und ändert sich nicht.
	// Auch beim Re-Broadcast bei neuem Peer bleibt er gleich —
	// der Request hat für alle denselben logischen Timestamp.
	msg := RicartRequest{
		FromID:       n.id,
		LogicalClock: n.requestClock, // eingefroren, nie n.clock direkt
		ResourceID:   n.resourceID,
		Timestamp:    time.Now(), // nur Logging, nie für Ordnung
	}
	payload, _ := json.Marshal(msg)
	n.mqttClient.Publish("ricart/request", 2, false, payload)
}

func (n *Node) handleRequest(msg RicartRequest) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.updateClock(msg.LogicalClock)

	// Aktualisiere lastSeen des Senders (er ist offensichtlich am Leben)
	if _, ok := n.peers[msg.FromID]; ok {
		n.peers[msg.FromID] = time.Now()
	}

	shouldDefer := n.state == RicartActorStateInCS ||
		(n.state == RicartActorStateWanting && n.hasPriority(msg))

	if shouldDefer {
		n.deferredReplies = append(n.deferredReplies, msg.FromID)
	} else {
		n.sendReply(msg.FromID)
	}
}

// Gibt true zurück wenn WIR Priorität haben (d.h. wir dürfen zuerst)
// Priorität: kleinerer Lamport-Timestamp gewinnt; bei Gleichstand kleinere ID
func (n *Node) hasPriority(incoming RicartRequest) bool {
	if n.requestClock != incoming.LogicalClock {
		return n.requestClock < incoming.LogicalClock
	}
	return n.id < incoming.FromID // lexikografischer Tiebreaker
}

// Vor dem Senden JEDER ausgehenden Nachricht die Uhr inkrementieren.
func (n *Node) sendReply(toID string) {
	// Uhr ticken bevor wir den Timestamp in die Nachricht schreiben
	clock := n.tick()

	msg := RicartReply{
		FromID:       n.id,
		ToID:         toID,
		ResourceID:   n.resourceID,
		LogicalClock: clock, // war vorher n.clock ohne tick — Bug
	}
	payload, _ := json.Marshal(msg)
	topic := fmt.Sprintf("ricart/reply/%s", toID)
	n.mqttClient.Publish(topic, 2, false, payload)
}

func (n *Node) handleReply(msg RicartReply) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.updateClock(msg.LogicalClock)
	delete(n.pendingReplies, msg.FromID)
	n.tryEnterCS()
}

// Muss unter n.mu aufgerufen werden
func (n *Node) tryEnterCS() {
	if n.state == RicartActorStateWanting && len(n.pendingReplies) == 0 {
		n.state = RicartActorStateInCS
		go n.onEnterCS() // non-blocking, Callback für die eigentliche CS-Arbeit
	}
}

func (n *Node) ReleaseCS() {
	n.mu.Lock()
	deferred := n.deferredReplies
	n.deferredReplies = nil
	n.state = RicartActorStateIdle
	n.mu.Unlock()

	// Alle aufgeschobenen Replies jetzt senden
	for _, id := range deferred {
		n.sendReply(id) // sendReply ist threadsafe da nur mqttClient.Publish aufruft
	}

	// Leitstelle informieren
	_ = MqttPublish(n.mqttClient, RicartCsExitParams(nil))
}
