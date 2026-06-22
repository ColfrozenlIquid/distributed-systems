package shared

import "time"

type Position struct {
	X uint32 `json:"x"`
	Y uint32 `json:"y"`
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
	VibrationSensor  SensorType = "vibration"
	WaterLevelSensor SensorType = "water_level"
	DroneSensor      SensorType = "person_confidence"
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
	Id        string     `json:"id"`
	Timestamp time.Time  `json:"timestamp"`
	Sensor    SensorInfo `json:"sensor"`
	Incident  Incident   `json:"incident"`
}

func (r *SensorReading) IsCritical() bool {
	switch r.Sensor.SensorType {
	case VibrationSensor:
		return r.Value > 4
	case WaterLevelSensor:
		return r.Value > 80
	case DroneSensor:
		return r.Value > 0.8
	default:
		return false
	}
}

func (r *SensorReading) ToIncident() SensorIncident {
	return SensorIncident{
		Sensor:    r.Sensor,
		Timestamp: r.Timestamp,
		Incident:  r.Sensor.SensorType.IncidentForSensorType(),
	}
}

func (r SensorType) IncidentForSensorType() Incident {
	switch r {
	case VibrationSensor:
		return VibrationTooHigh
	case WaterLevelSensor:
		return WaterLevelAlert
	case DroneSensor:
		return PersonDetected
	default:
		return -1
	}
}

type ActorType string

const (
	RescueVehicleActor    ActorType = "rescue_vehicle"
	VibrationVehicleActor ActorType = "vibration_vehicle"
	WaterVehicleActor     ActorType = "water_vehicle"
)

type ActorInfo struct {
	Id        string    `json:"id"`
	Position  Position  `json:"position"`
	ActorType ActorType `json:"actor_type"`
}

type ActorRegistration struct {
	Actor ActorInfo `json:"actor"`
}
