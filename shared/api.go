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
)

type SensorType string

const (
	VibrationSensor        SensorType = "vibration"
	WaterLevelSensor       SensorType = "water_level"
	SupplyLevelSensor      SensorType = "supply_level"
	PersonConfidenceSensor SensorType = "person_confidence"
)

type SensorInfo struct {
	Id         string     `json:"id"`
	Position   Position   `json:"position"`
	SensorType SensorType `json:"sensor_type"`
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
