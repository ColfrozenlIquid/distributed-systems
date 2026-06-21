# Aufgabe X
## Funktional: [Titel]
**Ziel:**

**Bewertung:** `PASS`, `FAIL` oder `BLOCKED`

**Versionsstand:** Commit `#`

### Setup
Container, Konfiguration, Testdaten

### Durchführung


### Ergebnisse
#### Erwartetes Ergebnis


#### Tatsächliches Ergebnis


#### Artefakte
Logs, Screenshots oder Reports


## Nicht-funktional: Vergleich Durchsatzmessung der Transportschichten

**Ziel:** Es wird gemessen, wie der Durchsatz einer Nutzlast durch die eingesetzte Transportschicht beeinflusst wird. Dazu wird die Nutzlast über HTTP, gRPC (unary + stream) sowie MQTT (QoS 0 & QoS 1) versendet und der Durchsatz auf Empfängerseite gemessen. Der Test simuliert einen Knoten im Netzwerk, welcher so schnell wie moeglich eine Nutzlast produziert, und diese an einen Konsumenten abgeben moechte. Hierbei wird getest Das geprüfte Qualitätsmerkmal ist der Durchsatz.

**Bewertung:** `PASS`

**Versionsstand:** Commit `2760b922`

### Setup

**Container:** MQTT Runtime

**Konfiguration:** Keine

**Testdaten:** Über alle Transportschichten hinweg wurden folgende Nutzdaten versendet und auf Empfängerseite deserialisiert.

```go
type wirePayload struct {
	ActorID   string `json:"actor_id"`
	PosX      int32  `json:"pos_x"`
	PosY      int32  `json:"pos_y"`
	Timestamp uint64 `json:"timestamp"`
	SentAt    uint64 `json:"sent_at"`
}
```

### Durchführung

Im Test wurden sequentiell 100.000 Nachrichtenpakete mit der angegebenen Nutzlast über die verschiedenen Transportprotokolle verschickt. Dabei wurde gemessen, wie viele Nachrichtenpakete pro Sekunde den Empfänger erreichen.

### Ergebnisse

#### Erwartetes Ergebnis

Es wird erwartet, dass der gRPC-Stream den höchsten Durchsatz erzielt. HTTP und unary gRPC sollten ähnliche Werte liefern, wobei gRPC aufgrund seines Binärformats einen Vorteil gegenüber HTTP haben dürfte. MQTT mit QoS 0 sollte deutlich höheren Durchsatz aufweisen als MQTT mit QoS 1. In einem echten Netzwerk mit Latenz > 0 ms sollten alle Protokolle deutlich reduzierten Durchsatz aufweisen. gRPC streaming und MQTT QoS 0 sollten weiterhin beim Durchsatz dominieren. Bei allen Protokolle mit Request / Reply sollten bei Latenzen > 0 die Latenz den Durchsatz extrem stark beeinflussen.

#### Tatsächliches Ergebnis

Das Ergebnis ist in den folgenden Grafiken visualisiert. Die erste Grafik zeigt den Durchsatz auf einem lokalen Rechner, wo Produzen und Konsument auf der selben Maschine laufen. Die zweite Grafik zeigt den Durchsatz bei 2 Rechnern die ueber ein echtes Netzwerk mit einem durchschnitts RTT von rund 20ms verbunden sind. Es ist klar zu erkennen das die asynchronen Verfahren ein vielfaches vom Durchsatz aufweisen. Es ist auch zu erkennen das in echten NEtzwerken der Durchsatz physikalisch auf `Durchsatz [Nachrichten / Sekunde] = 1 Sekunde / RTT` beschraenkt ist.

![](./aufgabe3/benchmark_chart_local.png)

![](./aufgabe3/benchmark_chart_remote.png)

#### Artefakte

Logs, Screenshots oder Reports
