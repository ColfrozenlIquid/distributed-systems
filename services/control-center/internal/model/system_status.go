package model

import (
	"shared"
	"sync"
)

type SystemStatus struct {
	mu        sync.RWMutex
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
}

type SystemStatusSnapshot struct {
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
}

func (s *SystemStatus) AddSensor(sensor shared.SensorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sensors = append(s.Sensors, sensor)
}

func (s *SystemStatus) AddReading(reading shared.SensorReading) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Readings = append(s.Readings, reading)
	if reading.IsCritical() {
		s.addIncidentLocked(reading.ToIncident())
	}
}

func (s *SystemStatus) AddIncident(incident shared.SensorIncident) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addIncidentLocked(incident)
}

func (s *SystemStatus) Snapshot() SystemStatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SystemStatusSnapshot{
		Sensors:   append([]shared.SensorInfo(nil), s.Sensors...),
		Readings:  append([]shared.SensorReading(nil), s.Readings...),
		Incidents: append([]shared.SensorIncident(nil), s.Incidents...),
	}
}

func (s *SystemStatus) addIncidentLocked(incident shared.SensorIncident) {
	s.Incidents = append(s.Incidents, incident)
}
