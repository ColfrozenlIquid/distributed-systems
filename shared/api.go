package shared

import "time"

type Position struct {
	x int
	y int
}

type Incident int

const (
	VibrationTooHigh Incident = iota
	WaterLevelAlert
	SupplyCritical
	PersonDetected
	BlockedRoute
)

type SensorInfo struct {
	Id         string   `json:"id"`
	Position   Position `json:"position"`
	SensorType string   `json:"sensor_type"`
}

type SensorRegistration struct {
	Sensor SensorInfo `json:"sensor"`
}

type SensorReading struct {
	Timestamp time.Time  `json:"timestamp"`
	Sensor    SensorInfo `json:"sensor"`
	Value     float32    `json:"value"`
}

type SensorIncident struct {
	Timestamp time.Time  `json:"timestamp"`
	Sensor    SensorInfo `json:"sensor"`
	Incident  Incident   `json:"incident"`
}
