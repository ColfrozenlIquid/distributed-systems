package model

import (
	"shared"
	"slices"
	"sync"
)

type SystemStatus struct {
	mu        sync.RWMutex
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []shared.ActorInfo      `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
}

type SystemStatusSnapshot struct {
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []shared.ActorInfo      `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
}

func (s *SystemStatus) AddSensor(sensor shared.SensorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sensors = append(s.Sensors, sensor)
}

func (s *SystemStatus) AddActor(actor shared.ActorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Actors = append(s.Actors, actor)
}

func (s *SystemStatus) AddReading(reading shared.SensorReading) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Readings = append(s.Readings, reading)
	s.updateSensorInfo(reading)
	if reading.IsCritical() {
		s.addIncidentLocked(reading.ToIncident())
	}
}

func (s *SystemStatus) updateSensorInfo(reading shared.SensorReading) {
	if index := slices.IndexFunc(s.Sensors, func(sensor shared.SensorInfo) bool {
		return sensor.Id == reading.Sensor.Id
	}); index >= 0 {
		slices.Replace(s.Sensors, index, index+1, reading.Sensor)
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
		Actors:    append([]shared.ActorInfo(nil), s.Actors...),
		Readings:  append([]shared.SensorReading(nil), s.Readings...),
		Incidents: append([]shared.SensorIncident(nil), s.Incidents...),
	}
}

func (s *SystemStatus) addIncidentLocked(incident shared.SensorIncident) {
	if slices.ContainsFunc(s.Incidents, func(i shared.SensorIncident) bool {
		return i.Sensor == incident.Sensor
	}) {
		return
	}
	s.Incidents = append(s.Incidents, incident)
}
