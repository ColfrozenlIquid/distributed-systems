# Aufgabe 1

## Funktional: Registrierung, kritischer Messwert und Dashboard-Status

**Ziel:**
Es wird geprüft, ob die Leitstelle im deployten Docker-Compose-System über die selbst implementierte
HTTP/REST-Schnittstelle einen Sensor und einen Actor registriert, einen kritischen Wasserstand annimmt und daraus im
verteilten Systemzustand einen `WaterLevelAlert` ableitet. Zusätzlich wird geprüft, ob `/map` weiterhin renderbares HTML
für das Dashboard liefert.

**Bewertung:** `PASS`

**Versionsstand:** Commit `9763cd9`

### Setup

**Container:** `world-simulation`, `control-center`, `sensor-*`, `actor-*` aus `docker-compose.yaml`.

**Testdaten:** Der Go-Test erzeugt eindeutige IDs für einen zusätzlichen Sensor vom Typ `water_level`, einen
zusätzlichen
Actor vom Typ `rescue_vehicle` und sendet einen Messwert `95.0` an Position `(3,3)`. Der Schwellwert für `water_level`
liegt bei `> 80`, daher muss ein Vorfall entstehen.

**Automatisierter Test:** `tests/aufgabe1/aufgabe1_integration_test.go`, Testfunktion
`TestRegistrationCriticalReadingAndDashboard`.

### Durchführung

Aus dem Repository-Wurzelverzeichnis:

```bash
docker compose up -d --build
docker run --rm \
  --network vs-praktikum_default \
  -v "$PWD/tests/aufgabe1:/tests" \
  -w /tests \
  -e CONTROL_CENTER_URL=http://control-center:8080 \
  golang:1.26.2-alpine \
  go test -v -run TestRegistrationCriticalReadingAndDashboard .
```

### Ergebnisse
#### Tatsächliches Ergebnis

Der Test wurde im Docker-Netz `vs-praktikum_default` ausgeführt und war erfolgreich:

```text
=== RUN   TestRegistrationCriticalReadingAndDashboard
    aufgabe1_integration_test.go:97: functional result: registered_sensor=it-water-1780954705276180327 registered_actor=it-actor-1780954705276181827 critical_value=95.0 incident=WaterLevelAlert
    aufgabe1_integration_test.go:98: status summary: sensors=8 actors=4 readings=9830 incidents=7
    aufgabe1_integration_test.go:99: dashboard result: content_type="text/html; charset=utf-8" bytes=292948
--- PASS: TestRegistrationCriticalReadingAndDashboard (0.13s)
PASS
ok      aufgabe1tests   0.128s
```

## Nicht-funktional: Verfügbarkeit und Antwortzeit des Status-Endpunkts

**Ziel:** Es wird gemessen, ob die Leitstelle ihren Gesamtzustand unter wiederholten Client-/Dashboard-Abfragen stabil
und schnell
ausliefert. Die geprüften Qualitätsmerkmale sind Verfügbarkeit und Antwortzeit.

**Bewertung:** `PASS`

**Versionsstand:** Commit `9763cd9`

### Setup

**Container:** `world-simulation`, `control-center`, `sensor-*`, `actor-*` aus `docker-compose.yaml`.

**Messkriterien:** 200 sequenzielle `GET /status`-Anfragen, 0 HTTP-/Transportfehler, alle Antworten HTTP `200`,
p95-Antwortzeit unter `200 ms`.

**Automatisierter Test:** `tests/aufgabe1/aufgabe1_integration_test.go`, Testfunktion
`TestStatusEndpointAvailabilityAndLatency`.

### Durchführung

Aus dem Repository-Wurzelverzeichnis:

```bash
docker compose up -d --build
docker run --rm \
  --network vs-praktikum_default \
  -v "$PWD/tests/aufgabe1:/tests" \
  -w /tests \
  -e CONTROL_CENTER_URL=http://control-center:8080 \
  golang:1.26.2-alpine \
  go test -v -run TestStatusEndpointAvailabilityAndLatency .
```

### Ergebnisse

#### Erwartetes Ergebnis

Der Go-Test endet mit `PASS`. Die Ausgabe enthält `requests=200`, `status_counts=map[200:200]` und Latenzstatistiken mit
`p95_ms < 200.00`.

#### Tatsächliches Ergebnis

Der Test wurde im Docker-Netz `vs-praktikum_default` ausgeführt und war erfolgreich:

```text
=== RUN   TestStatusEndpointAvailabilityAndLatency
    aufgabe1_integration_test.go:144: non-functional result: requests=200 status_counts=map[200:200]
    aufgabe1_integration_test.go:145: latency statistics: min_ms=2.78 avg_ms=3.13 p50_ms=2.99 p95_ms=3.95 max_ms=5.17
--- PASS: TestStatusEndpointAvailabilityAndLatency (0.64s)
PASS
ok      aufgabe1tests   0.637s
```