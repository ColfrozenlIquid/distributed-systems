## Das Membership-Problem

Ricart-Agrawala erfordert, dass jeder Teilnehmer weiß, von wie vielen anderen er ein REPLY braucht. Ohne Leitstelle gibt es drei realistische Lösungen:

**Option A — Statische Konfiguration:** Anzahl der Peers fest per Env-Variable. Simpelst, aber unflexibel bei Containerstart-Reihenfolge.

**Option B — MQTT-basiertes Presence-Protokoll (empfohlen):** Jeder Aktor publiziert seinen Join auf einem Presence-Topic mit `retain=true`. Alle anderen sehen durch Wildcard-Subscription das aktuelle Peer-Set. Membership ist damit dynamisch und selbstheilend.

**Option C — Gossip/Flood:** Zu komplex für dieses Projekt.

**→ Option B** passt perfekt zu eurer bestehenden MQTT-Architektur und erfüllt die Anforderung, dass die Leitstelle nur *beobachtet*.

---

## Topic-Struktur

```
ricart/presence/{actor_id}        QoS 1, retain=true   — Heartbeat / Join-Announce
ricart/request                      QoS 2, retain=false  — CS-Request an alle
ricart/reply/{target_actor_id}    QoS 2, retain=false  — Unicast-Reply
ricart/cs/enter                     QoS 1, retain=false  — Observability für Leitstelle
ricart/cs/exit                      QoS 1, retain=false  — Observability für Leitstelle
```

**Designentscheidungen:**
- `ricart/request` ist ein Broadcast-Topic — alle subscriben darauf
- `ricart/reply/{id}` ist ein Unicast-Topic — jeder subscribt nur auf seine eigene ID
- `retain=true` auf presence ermöglicht Late-Joiners, den aktuellen Peer-Set zu sehen
- QoS 2 auf REQUEST/REPLY ist zwingend — verlorene oder doppelte Nachrichten brechen die Korrektheit des Algorithmus
- Die CS-Topics dienen ausschließlich der Leitstellen-Observability (Aufgabenanforderung)

---

## Go-Structs mit JSON-Annotationen

```go
// ricart/types.go

package ricart

// ActorState repräsentiert den Zustand eines Aktors im Algorithmus
type ActorState string

const (
    StateIdle      ActorState = "IDLE"
    StateWanting   ActorState = "WANTING"   // will CS betreten, sammelt Replies
    StateInCS      ActorState = "IN_CS"     // in kritischem Abschnitt
)

// PresenceMsg — publiziert auf ricart/presence/{actor_id}
// QoS 1, retain=true
// Beim sauberen Shutdown: leere Payload (retained message löschen)
type PresenceMsg struct {
    ActorID string `json:"actor_id"`
    Role      string `json:"role"`       // "drone", "rover", etc.
    Timestamp int64  `json:"timestamp"`  // Unix ms, für lastSeen-Timeout
    Online    bool   `json:"online"`     // false = graceful leave
}

// RequestMsg — publiziert auf ricart/request
// QoS 2, retain=false
type RequestMsg struct {
    FromID        string `json:"from_id"`
    LogicalClock  uint64 `json:"logical_clock"`
    ResourceID    string `json:"resource_id"`    // z.B. "ladestation_nord"
    Timestamp     int64  `json:"timestamp"`      // Wall-clock, nur für Logging
}

// ReplyMsg — publiziert auf ricart/reply/{target_id}
// QoS 2, retain=false
type ReplyMsg struct {
    FromID       string `json:"from_id"`
    ToID         string `json:"to_id"`
    ResourceID   string `json:"resource_id"`
    LogicalClock uint64 `json:"logical_clock"`
}

// CSEventMsg — publiziert auf ricart/cs/enter und ricart/cs/exit
// QoS 1, retain=false — nur zur Observability durch Leitstelle
type CSEventMsg struct {
    ActorID    string `json:"actor_id"`
    ResourceID   string `json:"resource_id"`
    LogicalClock uint64 `json:"logical_clock"`
    Timestamp    int64  `json:"timestamp"`
}
```

---

## Kernlogik-Snippets

### Node-State und Membership-Tracking

```go
// ricart/node.go

type Node struct {
    mu sync.Mutex

    id         string
    resourceID string

    // Lamport-Uhr
    clock uint64

    // Algorithmus-Zustand
    state        ActorState
    requestClock uint64 // Uhrwert beim eigenen Request

    // Membership
    peers     map[string]time.Time // actor_id -> lastSeen
    peerTTL   time.Duration        // z.B. 30s; danach gilt Peer als tot

    // Deferred Replies: IDs, auf deren REPLY wir noch warten
    pendingReplies map[string]struct{}

    // Deferred Requests: Requests die wir zurückgehalten haben
    deferredReplies []string // actor_ids, denen wir noch antworten müssen

    mqttClient mqtt.Client
    onEnterCS  func() // Callback wenn CS betreten werden kann
}
```

### Lamport-Uhr

```go
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
```

### Membership-Management via MQTT Presence

```go
// Aufgerufen wenn eine PresenceMsg eingeht (ricart/presence/+)
func (n *Node) handlePresence(msg PresenceMsg) {
    n.mu.Lock()
    defer n.mu.Unlock()

    if msg.ActorID == n.id {
        return // eigene Nachricht ignorieren
    }

    if msg.Online {
        wasUnknown := _, ok := n.peers[msg.ActorID]; !ok
        n.peers[msg.ActorID] = time.Now()

        if wasUnknown {
            // Neuer Peer: falls wir gerade WANTING sind,
            // müssen wir auch von ihm ein Reply bekommen.
            // Das geht nur, wenn er unseren Request noch nicht gesehen hat —
            // daher: wir senden nochmal einen Request ODER
            // akzeptieren, dass er unseren Request noch erhalten wird
            // (da wir retain=false nutzen, ist ein Re-Request nötig).
            // Einfachste Lösung: Pending-Reply hinzufügen und warten.
            if n.state == StateWanting {
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
```

### Request senden

```go
func (n *Node) RequestCS() {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.state = StateWanting
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
    msg := RequestMsg{
        FromID:       n.id,
        LogicalClock: n.requestClock, // eingefroren, nie n.clock direkt
        ResourceID:   n.resourceID,
        Timestamp:    time.Now().UnixMilli(), // nur Logging, nie für Ordnung
    }
    payload, _ := json.Marshal(msg)
    n.mqttClient.Publish("ricart/request", 2, false, payload)
}
```

### Eingehenden Request behandeln

```go
func (n *Node) handleRequest(msg RequestMsg) {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.updateClock(msg.LogicalClock)

    // Aktualisiere lastSeen des Senders (er ist offensichtlich am Leben)
    if _, ok := n.peers[msg.FromID]; ok {
        n.peers[msg.FromID] = time.Now()
    }

    shouldDefer := n.state == StateInCS ||
        (n.state == StateWanting && n.hasPriority(msg))

    if shouldDefer {
        n.deferredReplies = append(n.deferredReplies, msg.FromID)
    } else {
        n.sendReply(msg.FromID)
    }
}

// Gibt true zurück wenn WIR Priorität haben (d.h. wir dürfen zuerst)
// Priorität: kleinerer Lamport-Timestamp gewinnt; bei Gleichstand kleinere ID
func (n *Node) hasPriority(incoming RequestMsg) bool {
    if n.requestClock != incoming.LogicalClock {
        return n.requestClock < incoming.LogicalClock
    }
    return n.id < incoming.FromID // lexikografischer Tiebreaker
}

// Vor dem Senden JEDER ausgehenden Nachricht die Uhr inkrementieren.
func (n *Node) sendReply(toID string) {
    // Uhr ticken bevor wir den Timestamp in die Nachricht schreiben
    clock := n.tick()

    msg := ReplyMsg{
        FromID:       n.id,
        ToID:         toID,
        ResourceID:   n.resourceID,
        LogicalClock: clock, // war vorher n.clock ohne tick — Bug
    }
    payload, _ := json.Marshal(msg)
    topic := fmt.Sprintf("ricart/reply/%s", toID)
    n.mqttClient.Publish(topic, 2, false, payload)
}
```

### Eingehendes Reply behandeln

```go
func (n *Node) handleReply(msg ReplyMsg) {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.updateClock(msg.LogicalClock)
    delete(n.pendingReplies, msg.FromID)
    n.tryEnterCS()
}

// Muss unter n.mu aufgerufen werden
func (n *Node) tryEnterCS() {
    if n.state == StateWanting && len(n.pendingReplies) == 0 {
        n.state = StateInCS
        go n.onEnterCS() // non-blocking, Callback für die eigentliche CS-Arbeit
    }
}
```

### CS verlassen

```go
func (n *Node) ReleaseCS() {
    n.mu.Lock()
    deferred := n.deferredReplies
    n.deferredReplies = nil
    n.state = StateIdle
    n.mu.Unlock()

    // Alle aufgeschobenen Replies jetzt senden
    for _, id := range deferred {
        n.sendReply(id) // sendReply ist threadsafe da nur mqttClient.Publish aufruft
    }

    // Leitstelle informieren
    n.publishCSEvent("ricart/cs/exit")
}
```

---

## Das Late-Joiner / Timing-Problem

Es gibt eine Racecondition: Node A sendet einen Request, danach joint Node B. B hat den Request nie gesehen, A wartet aber auf B's Reply.

**Lösung im obigen Code:** Beim Presence-Join eines neuen Peers während `StateWanting` sendet A den Request nochmal (`broadcastRequest()`). B empfängt den Request und antwortet. Da B gerade nicht in der CS ist, antwortet er sofort — kein Problem.

**Alternativer Ansatz** (robuster für das Praktikum): Beim Joinen einer Gruppe publiziert jeder neue Peer explizit eine `JoinAck`-Nachricht an alle, die gerade `StateWanting` sind. Das erfordert aber mehr State. Der Re-Broadcast ist einfacher und korrekt.

---

## Subscriptions beim Start

```go
// Beim Initialisieren
client.Subscribe("ricart/presence/+",  1, node.handlePresenceMQTT)
client.Subscribe("ricart/request",     2, node.handleRequestMQTT)
client.Subscribe(fmt.Sprintf("ricart/reply/%s", node.id), 2, node.handleReplyMQTT)

// Eigene Presence ankündigen (retained!)
presenceMsg := PresenceMsg{ActorID: node.id, Role: role, Online: true, Timestamp: time.Now().UnixMilli()}
payload, _ := json.Marshal(presenceMsg)
client.Publish(fmt.Sprintf("ricart/presence/%s", node.id), 1, true, payload)

// LWT für ungraceful disconnect (vor Connect setzen)
// LWT-Payload: PresenceMsg{Online: false}
```
---
## Liveness nachweisen

Liveness bedeutet: **jeder, der die CS betreten will, tut das irgendwann.** Du musst das nicht formal beweisen, sondern experimentell zeigen — also Logs produzieren, die den Fortschritt sichtbar machen.

### Was du loggen musst

Jeder Zustandsübergang braucht einen strukturierten Log-Eintrag mit Lamport-Zeit und Wall-Clock:

```go
type LivenessEvent struct {
    WallTime     time.Time    `json:"wall_time"`
    LogicalClock uint64       `json:"logical_clock"`
    VehicleID    string       `json:"vehicle_id"`
    Event        string       `json:"event"`        // REQUEST_SENT, REPLY_RECEIVED, CS_ENTER, CS_EXIT, REPLY_SENT_DEFERRED
    Detail       string       `json:"detail,omitempty"` // z.B. "waiting_for=[drone-2,drone-3]" oder "deferred_to=drone-1"
}
```

Ein Liveness-Nachweis sieht dann so aus — drei Fahrzeuge, alle wollen gleichzeitig die CS:

```
t=1  drone-1  REQUEST_SENT        logical_clock=5  waiting_for=[drone-2,drone-3]
t=1  drone-2  REQUEST_SENT        logical_clock=5  waiting_for=[drone-1,drone-3]
t=1  drone-3  REQUEST_SENT        logical_clock=3  waiting_for=[drone-1,drone-2]

t=2  drone-1  REPLY_SENT          to=drone-3   (drone-3 hat kleineren clock, gibt nach)
t=2  drone-2  REPLY_SENT          to=drone-3
t=2  drone-3  REPLY_RECEIVED      from=drone-1  pending=1
t=2  drone-3  REPLY_RECEIVED      from=drone-2  pending=0
t=2  drone-3  CS_ENTER            logical_clock=4

t=3  drone-3  CS_EXIT             logical_clock=7
t=3  drone-3  REPLY_SENT_DEFERRED to=drone-1
t=3  drone-3  REPLY_SENT_DEFERRED to=drone-2

t=4  drone-1  REPLY_RECEIVED      from=drone-3  pending=1
t=4  drone-2  REPLY_RECEIVED      from=drone-3  pending=1
...
```

Aus diesem Log kannst du in der Abnahme zeigen: Kein Fahrzeug verhungert, jeder CS_ENTER folgt auf einen CS_EXIT eines anderen.

### Automatisierter Liveness-Check

Zusätzlich zum Log kannst du einen einfachen Watchdog einbauen:

```go
func (n *Node) startLivenessWatchdog(timeout time.Duration) {
    go func() {
        for {
            time.Sleep(timeout / 2)
            n.mu.Lock()
            if n.state == StateWanting {
                waited := time.Since(n.wantingSince) // Timestamp beim Eintritt in WANTING
                if waited > timeout {
                    log.Printf("[LIVENESS_WARN] vehicle=%s waited=%v pending=%v — possible deadlock",
                        n.id, waited, maps.Keys(n.pendingReplies))
                }
            }
            n.mu.Unlock()
        }
    }()
}
```

Das gibt dir im Testprotokoll einen messbaren, nicht-funktionalen Nachweis: *"Kein Fahrzeug wartete länger als X ms ohne Fortschritt."*

---

## Fehlerszenarien

### 1. Nachrichtenausfall

MQTT QoS 2 eliminiert Nachrichtenverlust auf der Broker-Ebene praktisch. Relevanter für euch ist der Fall, dass der **Broker selbst kurz ausfällt** während ein QoS-2-Handshake läuft.

**Was passiert:** Der Paho-Client puffert unbestätigte QoS-2-Nachrichten in der persistenten Session und sendet sie nach Reconnect nach. Solange die Session nicht als Clean gestartet wird, gehen keine Nachrichten verloren.

**Was du zeigen kannst:** Broker für 5 Sekunden stoppen während drone-1 einen REQUEST gesendet hat aber noch nicht alle REPLYs hat. Nach Broker-Restart: drone-1 betritt die CS korrekt.

```go
// Sicherstellen dass persistente Session konfiguriert ist:
opts.SetCleanSession(false)
opts.SetClientID(n.id) // stabile ID für Session-Wiedererkennung
```

**Für das Testprotokoll:** Zeige den Log-Ausschnitt mit der Reconnect-Sequenz und dem späteren CS_ENTER.

---

### 2. Prozessabsturz

Das ist der kritischste Fall — ein Fahrzeug stirbt während andere auf sein REPLY warten.

**Problem:** drone-2 hat einen REQUEST von drone-1 empfangen und würde normalerweise antworten — aber erst nach seinem eigenen CS_EXIT. drone-2 crasht. drone-1 wartet ewig → **Deadlock**.

**Deine Lösung liegt bereits im Membership-System:** Der `peerTTL`-Timeout in `reapDeadPeers()` behandelt genau das:

```go
func (n *Node) reapDeadPeers() {
    n.mu.Lock()
    defer n.mu.Unlock()

    for id, lastSeen := range n.peers {
        if time.Since(lastSeen) > n.peerTTL {
            log.Printf("[PEER_DEAD] vehicle=%s last_seen=%v — removing from pending",
                n.id, lastSeen)
            n.removePeer(id) // löscht auch aus pendingReplies + ruft tryEnterCS()
        }
    }
}
```

`removePeer` wertet das ausstehende Reply des toten Peers implizit als erteilt. Das ist die **Crash-Assumption**: Ein toter Prozess hält keine Ressource mehr.

**Wichtig:** Der LWT des abgestürzten Fahrzeugs (`PresenceMsg{Online: false}`) triggert `handlePresence` → `removePeer` sofort, ohne auf den TTL-Timeout warten zu müssen — das ist der schnelle Pfad. Der TTL-Reaper ist der Fallback für den Fall, dass auch der LWT verloren geht (z.B. Broker-Neustart genau im falschen Moment).

**Was du zeigen kannst:**

```
Szenario: drone-1 wartet auf Reply von drone-2. drone-2 wird gekillt (docker kill).

t=0   drone-1  REQUEST_SENT      pending=[drone-2, drone-3]
t=0   drone-3  REPLY_SENT        to=drone-1
t=1   drone-1  REPLY_RECEIVED    from=drone-3  pending=[drone-2]
t=1   drone-2  LWT published     (broker sendet LWT nach disconnect)
t=1   drone-1  PEER_DEAD         vehicle=drone-2
t=1   drone-1  CS_ENTER          (pending leer nach removePeer)
```

**Trade-off für die Abnahme ansprechen:** Diese Lösung ist sicher solange der Absturz-Peer die CS nicht gerade hält. Falls er *in* der CS abstirbt und der LWT nicht rechtzeitig kommt, kann ein anderes Fahrzeug die CS betreten bevor die Ressource wirklich frei ist. Für das Praktikum ist das akzeptabel — vollständige Crash-Recovery (z.B. mit Epoch-Nummern) wäre Forschungsgebiet.

---

### 3. Nicht-FIFO-Kommunikation

MQTT über TCP ist de facto FIFO pro Connection — TCP garantiert Reihenfolge. Nicht-FIFO ist also kein Praxisproblem bei euch, aber ihr müsst zeigen, dass ihr die Konsequenz verstanden habt.

**Was passiert theoretisch:** drone-1 sendet REQUEST (clock=5), dann später REQUEST (clock=7, Re-Broadcast wegen Late-Joiner). drone-2 empfängt sie in Reihenfolge 7, dann 5. drone-2 hat bereits auf clock=7 geantwortet — antwortet nun nochmal auf clock=5. drone-1 bekommt zwei REPLYs von drone-2.

**Wie du das absicherst:**

```go
// Pro Peer den zuletzt gesehenen Request-Clock merken
type peerState struct {
    lastSeen    time.Time
    lastReqClock uint64 // höchster bisher gesehener RequestClock von diesem Peer
}

func (n *Node) handleRequest(msg RequestMsg) {
    n.mu.Lock()
    defer n.mu.Unlock()

    n.updateClock(msg.LogicalClock)

    // Veralteten/duplizierten Request erkennen und verwerfen
    ps := n.peerStates[msg.FromID]
    if msg.LogicalClock <= ps.lastReqClock {
        log.Printf("[DUPLICATE_REQUEST] from=%s clock=%d last_seen=%d — ignoring",
            msg.FromID, msg.LogicalClock, ps.lastReqClock)
        return
    }
    ps.lastReqClock = msg.LogicalClock
    ps.lastSeen = time.Now()
    n.peerStates[msg.FromID] = ps

    // ... rest wie gehabt
}
```

Auf der Reply-Seite: mehrfache REPLYs vom selben Peer für denselben Request sind harmlos — `delete(n.pendingReplies, msg.FromID)` ist idempotent.

**Was du für die Abnahme zeigen kannst:** Einen Unit-Test, der `handleRequest` zweimal mit demselben `LogicalClock`-Wert aufruft und zeigt, dass nur eine Reply gesendet wird.

---

## Zusammenfassung für das Testprotokoll

| Szenario           | Nachweis                                     | Erwartetes Verhalten                           |
|--------------------|----------------------------------------------|------------------------------------------------|
| Liveness (3 Peers) | Strukturierter Log aller Zustandsübergänge   | Jeder betritt CS genau einmal, kein Verhungern |
| Broker-Ausfall     | `docker stop broker` während WANTING         | Nach Reconnect korrekte CS_ENTER               |
| Prozessabsturz     | `docker kill drone-2` während drone-1 wartet | LWT → removePeer → CS_ENTER für drone-1        |
| Nicht-FIFO         | Unit-Test mit dupliziertem Request           | Duplikat wird verworfen, kein doppeltes Reply  |