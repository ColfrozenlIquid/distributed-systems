# Aufgabe 3
## Funktional: MQTT-Nachrichtenfluss für Sensordaten und Actor-Status
**Ziel:**
Es wird geprüft, ob die für Aufgabe 3 eingeführte MQTT-Kommunikation im deployten Docker-Compose-System funktional
arbeitet. Der Test verbindet sich als echter MQTT-Client mit Mosquitto, abonniert die Wildcard-Topics der Anwendung,
publiziert eine kritische Sensorlesung sowie Actor-Status und Actor-Telemetrie und prüft anschließend, ob die
Leitstelle diese Nachrichten in `/status` übernommen hat. Damit werden Broker-Zustellung, Topic-Schema, JSON-Payloads
und die reaktive Verarbeitung im `control-center` gemeinsam getestet.

**Bewertung:** `PASS`

**Versionsstand:** Commit `acdc92f`

### Setup
**Container:** `mqtt-broker`, `control-center`, `world-simulation`, `sensor-*`, `actor-*` aus `docker-compose.yaml`.

**Konfiguration:** Der Test nutzt `CONTROL_CENTER_URL` (Default `http://control-center:8080`) und `MQTT_BROKER_URL`
oder alternativ `MQTT_BROKER_HOST`/`MQTT_BROKER_PORT` (Default `tcp://mqtt-broker:1883`).

**Testdaten:** Der Go-Test registriert einen eindeutigen `water_level`-Sensor und einen eindeutigen `water_vehicle`
ueber die HTTP-Schnittstelle, publiziert danach eine kritische MQTT-Sensorlesung mit Wert `95.0` auf
`tiles/<x>/<y>/readings/water_level`, einen Actor-Status `on_a_mission` auf `actors/<actor-id>/status` und eine
Actor-Telemetrie auf `actors/<actor-id>/telemetry`.

**Automatisierter Test:** `tests/aufgabe3/aufgabe3_mqtt_integration_test.go`, Testfunktion
`TestMQTTSensorAndActorMessagesReachControlCenter`.

### Durchführung
Aus dem Repository-Wurzelverzeichnis:

```bash
docker compose up -d --build
docker run --rm \
  --network vs-praktikum_default \
  -v "$PWD/tests/aufgabe3:/tests" \
  -w /tests \
  -e CONTROL_CENTER_URL=http://control-center:8080 \
  -e MQTT_BROKER_URL=tcp://mqtt-broker:1883 \
  golang:1.26.2-alpine \
  go test -v -run TestMQTTSensorAndActorMessagesReachControlCenter .
```

### Ergebnisse
#### Erwartetes Ergebnis
Der Go-Test endet mit `PASS`. Die MQTT-Subscription empfängt die publizierten Nachrichten auf den erwarteten Topics.
`GET /status` enthaelt die kritische Sensorlesung, einen `WaterLevelAlert` (`incident=1`) für diesen Sensor sowie den
per MQTT aktualisierten Actor mit Status `on_a_mission` und der publizierten Position.


#### Tatsächliches Ergebnis
Der Test wurde im Docker-Netz `vs-praktikum_default` ausgefuehrt und war erfolgreich:

```text
=== RUN   TestMQTTSensorAndActorMessagesReachControlCenter
    aufgabe3_mqtt_integration_test.go:164: functional result: mqtt_broker=tcp://mqtt-broker:1883 reading_topic=tiles/6/6/readings/water_level status_topic=actors/it3-actor-1782148678786595540/status telemetry_topic=actors/it3-actor-1782148678786595540/telemetry
    aufgabe3_mqtt_integration_test.go:165: status summary: sensors=8 actors=4 readings=1061 incidents=5
    aufgabe3_mqtt_integration_test.go:166: verified mqtt control-center projection: sensor_id=it3-water-1782148678786595540 value=95.0 incident=WaterLevelAlert actor_id=it3-actor-1782148678786595540 state=on_a_mission position=(6,6)
--- PASS: TestMQTTSensorAndActorMessagesReachControlCenter (0.05s)
PASS
ok  	aufgabe3tests	0.053s
```


#### Artefakte
Automatisierter Test: `tests/aufgabe3/aufgabe3_mqtt_integration_test.go`.


## Nicht-funktional: Vergleich Durchsatzmessung der Transportschichten

**Ziel:** Es wird gemessen, wie der Durchsatz einer Nutzlast durch die eingesetzte Transportschicht beeinflusst wird. Dazu wird die Nutzlast über HTTP, gRPC (unary + stream) sowie MQTT (QoS 0 & QoS 1) versendet und der Durchsatz auf Empfängerseite gemessen. Der Test simuliert einen Knoten im Netzwerk, welcher so schnell wie möglich eine Nutzlast produziert, und diese an einen Konsumenten abgeben möchte. Das geprüfte Qualitätsmerkmal ist der Durchsatz.

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

Im Test wurden sequenziell 100 000 Nachrichtenpakete mit der angegebenen Nutzlast über die verschiedenen Transportprotokolle verschickt. Dabei wurde gemessen, wie viele Nachrichtenpakete pro Sekunde den Empfänger erreichen.

### Ergebnisse

#### Erwartetes Ergebnis

Es wird erwartet, dass der gRPC-Stream den höchsten Durchsatz erzielt. HTTP und unary gRPC sollten ähnliche Werte liefern, wobei gRPC aufgrund seines Binärformats einen Vorteil gegenüber HTTP haben dürfte. MQTT mit QoS 0 sollte deutlich höheren Durchsatz aufweisen als MQTT mit QoS 1. In einem echten Netzwerk mit Latenz > 0 ms sollten alle Protokolle deutlich reduzierten Durchsatz aufweisen. gRPC streaming und MQTT QoS 0 sollten weiterhin beim Durchsatz dominieren. Bei allen Protokollen mit Request / Reply sollte bei Latenzen > 0 die Latenz den Durchsatz extrem stark beeinflussen.

#### Tatsächliches Ergebnis

Das Ergebnis ist in den folgenden Grafiken visualisiert. Die erste Grafik zeigt den Durchsatz auf einem lokalen Rechner, wo Produzent und Konsument auf derselben Maschine laufen. Die zweite Grafik zeigt den Durchsatz bei 2 Rechnern, die über ein echtes Netzwerk mit einer durchschnittlichen RTT von rund 20ms verbunden sind. Es ist klar zu erkennen, dass die asynchronen Verfahren ein Vielfaches vom Durchsatz aufweisen. Es ist auch zu erkennen das in echten NEtzwerken der Durchsatz physikalisch auf `Durchsatz [Nachrichten / Sekunde] = 1 Sekunde / RTT` beschraenkt ist.

![](./aufgabe3/benchmark_chart_local.png)

![](./aufgabe3/benchmark_chart_remote.png)
