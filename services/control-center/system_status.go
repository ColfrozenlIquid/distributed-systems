package main

import "shared"

type SystemStatus struct {
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
}

func (s *SystemStatus) AddSensor(sensor shared.SensorInfo) {
	s.Sensors = append(s.Sensors, sensor)
}

func (s *SystemStatus) AddReading(reading shared.SensorReading) {
	s.Readings = append(s.Readings, reading)
	if reading.IsCritical() {
		s.AddIncident(reading.ToIncident())
	}
}

func (s *SystemStatus) AddIncident(incident shared.SensorIncident) {
	s.Incidents = append(s.Incidents, incident)
}
