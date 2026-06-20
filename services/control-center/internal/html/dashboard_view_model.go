package html

import (
	"context"
	"control-center/internal/model"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	pb "control-center/gen"
	"shared"

	"github.com/a-h/templ"
)

type DashboardPageData struct {
	Title       string
	GeneratedAt time.Time
	Width       int
	Height      int
	Tiles       [][]DashboardTile
	Summary     DashboardSummary
	Sensors     []SensorListItem
	Actors      []ActorListItem
	Readings    []ReadingListItem
	Incidents   []IncidentListItem
	Legend      []LegendItem
	MapData     DashboardMapData
}

type DashboardSummary struct {
	TotalTiles       int
	LandTiles        int
	WaterTiles       int
	SensorCount      int
	ActorCount       int
	IncidentCount    int
	CriticalTiles    int
	HasLiveData      bool
	LastReadingLabel string
}

type DashboardTile struct {
	X                   int    `json:"x"`
	Y                   int    `json:"y"`
	Label               string `json:"label"`
	Terrain             string `json:"terrain"`
	Infrastructure      string `json:"infrastructure"`
	InfrastructureShort string `json:"infrastructure_short"`
	HasInfrastructure   bool   `json:"has_infrastructure"`
	HasSensor           bool   `json:"has_sensor"`
	SensorLabel         string `json:"sensor_label"`
	HasActor            bool   `json:"has_actor"`
	ActorLabel          string `json:"actor_label"`
	ActorID             string `json:"actor_id"`
	ActorState          string `json:"actor_state"`
	ReadingLabel        string `json:"reading_label"`
	ReadingTone         string `json:"reading_tone"`
	HasIncident         bool   `json:"has_incident"`
	IncidentLabel       string `json:"incident_label"`
}

type SensorListItem struct {
	ID       string
	Type     string
	Position string
}

type ActorListItem struct {
	ID       string
	Type     string
	Position string
	State    string
}

type ReadingListItem struct {
	SensorID  string
	Type      string
	Position  string
	Value     string
	Timestamp string
	Critical  bool
}

type IncidentListItem struct {
	ID           string
	Type         string
	SensorID     string
	Position     string
	Timestamp    string
	MissionID    string
	MissionState string
}

type LegendItem struct {
	Label string
	Class string
}

type DashboardMapData struct {
	Width     int             `json:"width"`
	Height    int             `json:"height"`
	Tiles     []DashboardTile `json:"tiles"`
	UpdatedAt string          `json:"updated_at"`
}

func NewDashboardPageData(status model.SystemStatusSnapshot, island *model.World) DashboardPageData {
	latestReadings := latestReadingsByPosition(status.Readings)
	incidents := incidentsByPosition(status.Incidents)
	sensorsByTile := sensorsByPosition(status.Sensors)
	actorsByTile := actorsByPosition(status.Actors)

	rows := make([][]DashboardTile, 0, len(island.Tiles))
	flatTiles := make([]DashboardTile, 0, island.Width*island.Height)
	summary := DashboardSummary{
		TotalTiles:    island.Width * island.Height,
		SensorCount:   len(status.Sensors),
		ActorCount:    len(status.Actors),
		IncidentCount: len(status.Incidents),
		HasLiveData:   len(status.Readings) > 0,
	}

	for y, row := range island.Tiles {
		tileRow := make([]DashboardTile, 0, len(row))
		for x, tile := range row {
			key := positionKey(uint32(x), uint32(y))
			reading, hasReading := latestReadings[key]
			incident, hasIncident := incidents[key]
			sensor, hasSensor := sensorsByTile[key]
			actor, hasActor := actorsByTile[key]

			tileView := NewDashboardTile(x, y, tile, sensor, hasSensor, actor, hasActor, reading, hasReading, incident, hasIncident)
			if tileView.Terrain == "land" {
				summary.LandTiles++
			} else {
				summary.WaterTiles++
			}
			if tileView.HasIncident {
				summary.CriticalTiles++
			}
			tileRow = append(tileRow, tileView)
			flatTiles = append(flatTiles, tileView)
		}
		rows = append(rows, tileRow)
	}

	if len(status.Readings) > 0 {
		last := status.Readings[len(status.Readings)-1]
		summary.LastReadingLabel = fmt.Sprintf("%s %.1f at %s", sensorTypeLabel(last.Sensor.SensorType), last.Value, formatPosition(last.Sensor.Position))
	} else {
		summary.LastReadingLabel = "No readings received"
	}

	return DashboardPageData{
		Title:       "Sturmflut Nordsee",
		GeneratedAt: time.Now(),
		Width:       island.Width,
		Height:      island.Height,
		Tiles:       rows,
		Summary:     summary,
		Sensors:     sensorListItems(status.Sensors),
		Actors:      actorListItems(status.Actors),
		Readings:    readingListItems(status.Readings),
		Incidents:   incidentListItems(status.Incidents, status.Missions),
		Legend: []LegendItem{
			{Label: "Land", Class: "legend-land"},
			{Label: "Water", Class: "legend-water"},
			{Label: "Infrastructure", Class: "legend-infrastructure"},
			{Label: "Sensor", Class: "legend-sensor"},
			{Label: "Actor", Class: "legend-actor"},
			{Label: "Critical incident", Class: "legend-incident"},
		},
		MapData: DashboardMapData{
			Width:     island.Width,
			Height:    island.Height,
			Tiles:     flatTiles,
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}
}

func NewDashboardTile(x int, y int, tile model.Tile, sensor shared.SensorInfo, hasSensor bool, actor model.Actor, hasActor bool, reading shared.SensorReading, hasReading bool, incident shared.SensorIncident, hasIncident bool) DashboardTile {
	terrain := terrainLabel(tile.Terrain)
	infra := infrastructureLabel(tile.Infrastructure)
	short := infrastructureShortLabel(tile.Infrastructure)
	label := fmt.Sprintf("Tile %02d,%02d: %s", x, y, terrain)
	if infra != "none" {
		label += ", " + infra
	}

	view := DashboardTile{
		X:                   x,
		Y:                   y,
		Label:               label,
		Terrain:             terrain,
		Infrastructure:      infra,
		InfrastructureShort: short,
		HasInfrastructure:   infra != "none",
		HasSensor:           hasSensor,
		HasActor:            hasActor,
	}

	if hasSensor {
		view.SensorLabel = sensorTypeLabel(sensor.SensorType)
	}
	if hasActor {
		view.ActorLabel = actorTypeLabel(actor.Type)
		view.ActorID = fallbackID(actor.Id)
		view.ActorState = string(actor.State)
		view.Label += ", " + view.ActorLabel
	}
	if hasReading {
		view.ReadingLabel = formatReadingValue(reading)
		view.ReadingTone = readingTone(reading)
		if view.SensorLabel == "" {
			view.SensorLabel = sensorTypeLabel(reading.Sensor.SensorType)
		}
	}
	if hasIncident {
		view.HasIncident = true
		view.IncidentLabel = incidentLabel(incident.Incident)
	}
	return view
}

func tileAttributes(tile DashboardTile) templ.Attributes {
	return templ.Attributes{
		"aria-label":          tile.Label,
		"data-testid":         fmt.Sprintf("world-tile-%02d-%02d", tile.X, tile.Y),
		"data-x":              fmt.Sprintf("%d", tile.X),
		"data-y":              fmt.Sprintf("%d", tile.Y),
		"data-terrain":        tile.Terrain,
		"data-infrastructure": tile.Infrastructure,
		"data-incident":       tile.IncidentLabel,
		"data-reading":        tile.ReadingLabel,
		"data-sensor":         tile.SensorLabel,
		"data-actor":          tile.ActorLabel,
		"data-actor-id":       tile.ActorID,
		"data-actor-state":    tile.ActorState,
		"title":               tile.Label,
	}
}

func positionKey(x uint32, y uint32) string {
	return fmt.Sprintf("%d:%d", x, y)
}

func latestReadingsByPosition(readings []shared.SensorReading) map[string]shared.SensorReading {
	latest := make(map[string]shared.SensorReading, len(readings))
	for _, reading := range readings {
		key := positionKey(reading.Sensor.Position.X, reading.Sensor.Position.Y)
		if current, ok := latest[key]; !ok || reading.Timestamp.After(current.Timestamp) {
			latest[key] = reading
		}
	}
	return latest
}

func incidentsByPosition(incidents []shared.SensorIncident) map[string]shared.SensorIncident {
	latest := make(map[string]shared.SensorIncident, len(incidents))
	for _, incident := range incidents {
		key := positionKey(incident.Sensor.Position.X, incident.Sensor.Position.Y)
		if current, ok := latest[key]; !ok || incident.Timestamp.After(current.Timestamp) {
			latest[key] = incident
		}
	}
	return latest
}

func sensorsByPosition(sensors []shared.SensorInfo) map[string]shared.SensorInfo {
	byPosition := make(map[string]shared.SensorInfo, len(sensors))
	for _, sensor := range sensors {
		byPosition[positionKey(sensor.Position.X, sensor.Position.Y)] = sensor
	}
	return byPosition
}

func actorsByPosition(actors []model.Actor) map[string]model.Actor {
	byPosition := make(map[string]model.Actor, len(actors))
	for _, actor := range actors {
		byPosition[positionKey(actor.Position.X, actor.Position.Y)] = actor
	}
	return byPosition
}

func terrainLabel(terrain pb.TerrainType) string {
	if terrain == pb.TerrainType_TERRAIN_TYPE_WATER {
		return "water"
	}
	return "land"
}

func infrastructureLabel(infrastructure pb.InfrastructureType) string {
	switch infrastructure {
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_CONTROL_CENTER:
		return "control center"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_BRIDGE:
		return "bridge"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_PORT:
		return "port"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_CHARGING_STATION:
		return "charging station"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_LANDING_STRIP:
		return "landing field"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_DEPOT:
		return "depot"
	default:
		return "none"
	}
}

func infrastructureShortLabel(infrastructure pb.InfrastructureType) string {
	switch infrastructure {
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_CONTROL_CENTER:
		return "C"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_BRIDGE:
		return "B"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_PORT:
		return "P"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_CHARGING_STATION:
		return "S"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_LANDING_STRIP:
		return "L"
	case pb.InfrastructureType_INFRASTRUCTURE_TYPE_DEPOT:
		return "D"
	default:
		return ""
	}
}

func sensorTypeLabel(sensorType shared.SensorType) string {
	return strings.ReplaceAll(string(sensorType), "_", " ")
}

func actorTypeLabel(actorType shared.ActorType) string {
	switch actorType {
	case shared.RescueVehicleActor:
		return "rescue vehicle"
	case shared.VibrationVehicleActor:
		return "vibration vehicle"
	case shared.WaterVehicleActor:
		return "water vehicle"
	default:
		return "unknown"
	}
}

func incidentLabel(incident shared.Incident) string {
	switch incident {
	case shared.VibrationTooHigh:
		return "vibration too high"
	case shared.WaterLevelAlert:
		return "water level alert"
	case shared.SupplyCritical:
		return "supply critical"
	case shared.PersonDetected:
		return "person detected"
	default:
		return "unknown incident"
	}
}

func readingTone(reading shared.SensorReading) string {
	if reading.IsCritical() {
		return "critical"
	}
	switch reading.Sensor.SensorType {
	case shared.DroneSensor:
		if reading.Value > 0.6 {
			return "warning"
		}
	default:
		if reading.Value > 65 {
			return "warning"
		}
	}
	return "normal"
}

func sensorListItems(sensors []shared.SensorInfo) []SensorListItem {
	items := make([]SensorListItem, 0, len(sensors))
	for _, sensor := range sensors {
		items = append(items, SensorListItem{
			ID:       fallbackID(sensor.Id),
			Type:     sensorTypeLabel(sensor.SensorType),
			Position: formatPosition(sensor.Position),
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func actorListItems(actors []model.Actor) []ActorListItem {
	items := make([]ActorListItem, 0, len(actors))
	for _, actor := range actors {
		items = append(items, ActorListItem{
			ID:       fallbackID(actor.Id),
			Type:     actorTypeLabel(actor.Type),
			Position: formatPosition(actor.Position),
			State:    string(actor.State),
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func readingListItems(readings []shared.SensorReading) []ReadingListItem {
	items := make([]ReadingListItem, 0, len(readings))
	for i := len(readings) - 1; i >= 0 && len(items) < 8; i-- {
		reading := readings[i]
		items = append(items, ReadingListItem{
			SensorID:  fallbackID(reading.Sensor.Id),
			Type:      sensorTypeLabel(reading.Sensor.SensorType),
			Position:  formatPosition(reading.Sensor.Position),
			Value:     formatReadingValue(reading),
			Timestamp: formatTimestamp(reading.Timestamp),
			Critical:  reading.IsCritical(),
		})
	}
	return items
}

func incidentListItems(incidents []shared.SensorIncident, missions []model.Mission) []IncidentListItem {
	items := make([]IncidentListItem, 0, len(incidents))
	// build map from incident id to mission
	missionByIncident := make(map[string]model.Mission)
	for _, m := range missions {
		missionByIncident[m.Incident.Id] = m
	}

	for i := len(incidents) - 1; i >= 0 && len(items) < 8; i-- {
		incident := incidents[i]
		if mission, ok := missionByIncident[incident.Id]; ok {
			items = append(items, IncidentListItemFromIncident(incident, &mission))
			continue
		}
		items = append(items, IncidentListItemFromIncident(incident, nil))
	}
	return items
}

func IncidentListItemFromIncident(incident shared.SensorIncident, mission *model.Mission) IncidentListItem {
	item := IncidentListItem{
		ID:        incident.Id,
		Type:      incidentLabel(incident.Incident),
		SensorID:  fallbackID(incident.Sensor.Id),
		Position:  formatPosition(incident.Sensor.Position),
		Timestamp: formatTimestamp(incident.Timestamp),
	}
	if mission != nil {
		item.MissionID = mission.Id
		item.MissionState = string(mission.State)
	}
	return item
}

func IncidentItemForMission(snapshot model.SystemStatusSnapshot, mission model.Mission) (IncidentListItem, bool) {
	for _, incident := range snapshot.Incidents {
		if incident.Id == mission.Incident.Id {
			return IncidentListItemFromIncident(incident, &mission), true
		}
	}
	return IncidentListItem{}, false
}

func WriteIncidentItemSSE(ctx context.Context, w io.Writer, snapshot model.SystemStatusSnapshot, mission model.Mission) error {
	item, ok := IncidentItemForMission(snapshot, mission)
	if !ok {
		return nil
	}

	var buf strings.Builder
	if err := IncidentItem(item).Render(ctx, &buf); err != nil {
		return err
	}

	data := strings.ReplaceAll(buf.String(), "\n", "\ndata: ")
	_, err := fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

func formatReadingValue(reading shared.SensorReading) string {
	value := math.Round(float64(reading.Value)*10) / 10
	switch reading.Sensor.SensorType {
	case shared.WaterLevelSensor:
		return fmt.Sprintf("%.1f cm", value)
	case shared.DroneSensor:
		return fmt.Sprintf("%.1f%%", value*100)
	default:
		return fmt.Sprintf("%.1f", value)
	}
}

func formatPosition(position shared.Position) string {
	return fmt.Sprintf("%02d,%02d", position.X, position.Y)
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "pending"
	}
	return ts.Format("15:04:05")
}

func fallbackID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "unregistered"
	}
	return id
}
