# Aufgabe 4
## Funktional: Ricart-Agrawala Mutual Exclusion
**Ziel:** Automatisiert nachweisen, dass mehrere Teilnehmer den Zugriff auf dieselbe kritische Ressource per Ricart-Agrawala koordinieren. Geprüft werden Request-/Reply-Fluss, Eintritt und Austritt aus der Critical Section, gegenseitiger Ausschluss, Priorisierung nach `(logical_clock, actor_id)` und die Beobachtbarkeit über `GET /ricart/status`.

**Bewertung:** `PASS`

**Versionsstand:** Commit `8b85523`

### Setup
Container, Konfiguration, Testdaten:

* Docker-Compose-Stack `vs-praktikum` mit `control-center`, `mqtt-broker` und `world-simulation`.
* Automatisierter Go-Test: `tests/aufgabe4/aufgabe4_ricart_integration_test.go`, Testfunktion `TestRicartAgrawalaMutualExclusionAndLogicalClocks`.
* Der Test erzeugt drei temporäre Ricart-Teilnehmer mit eindeutigen Actor-IDs und einer eindeutigen Ressource `charging-station:8:10:<run-id>`.
* Kommunikation erfolgt ausschließlich über die Ricart-MQTT-Topics `ricart/presence/+`, `ricart/request`, `ricart/reply/<actor-id>`, `ricart/cs/enter` und `ricart/cs/exit`.
* Beobachtung und Bewertung erfolgen über `GET /ricart/status` am Control-Center.

### Durchführung
```bash
docker run --rm --network vs-praktikum_default \
  -v "$PWD/tests/aufgabe4:/tests" \
  -w /tests \
  -e CONTROL_CENTER_URL=http://control-center:8080 \
  -e MQTT_BROKER_URL=tcp://mqtt-broker:1883 \
  golang:1.26.2-alpine \
  go test -v -run TestRicartAgrawalaMutualExclusionAndLogicalClocks .
```

Der Test wartet auf `/health`, verbindet drei MQTT-Teilnehmer, meldet sie per Presence online, startet konkurrierende Critical-Section-Requests und liest anschließend `/ricart/status`, bis alle erwarteten Events vorhanden sind.

Geprüfte Kriterien:

* Für jeden Teilnehmer existieren `request`, `cs_enter` und `cs_exit` in dieser Reihenfolge.
* Alle `request`, `reply` und `cs_enter` Events enthalten sinnvolle `logical_clock` Werte.
* Jeder `reply` enthält einen `peer_id`.
* Es gibt keine überlappenden Zeitintervalle zwischen `cs_enter` und `cs_exit` zweier Teilnehmer derselben Ressource.
* Die Reihenfolge der `cs_enter` Events folgt der Ricart-Agrawala-Priorität `(logical_clock, actor_id)`.
* Die Ressource ist nach allen `cs_exit` Events wieder `free`, hat keinen `holder_actor_id` und setzt `last_logical_clock`.

### Ergebnisse
#### Erwartetes Ergebnis
Der Go-Test endet mit `PASS`. Die Ausgabe enthält eine Funktionszusammenfassung mit drei Requests, mindestens sechs Replies, drei `cs_enter` Events und drei `cs_exit` Events für die Testressource. `/ricart/status` enthält die beteiligten Participants, die Ressource und die Eventliste mit `logical_clock`.


#### Tatsächliches Ergebnis
```text
=== RUN   TestRicartAgrawalaMutualExclusionAndLogicalClocks
    aufgabe4_ricart_integration_test.go:179: functional result: resource=charging-station:8:10:it4-1783252822913771110 actors=[it4-1783252822913771110-actor-a it4-1783252822913771110-actor-b it4-1783252822913771110-actor-c] requests=3 replies=6 cs_enter=3 cs_exit=3
    aufgabe4_ricart_integration_test.go:180: status summary: participants=6 resources=1 ricart_events=27 last_logical_clock=12
--- PASS: TestRicartAgrawalaMutualExclusionAndLogicalClocks (0.77s)
PASS
ok  	aufgabe4tests	0.771s
```

## Nicht-funktional: [Titel]
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