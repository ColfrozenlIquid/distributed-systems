package model

import (
	"control-center/internal/mission"
	"fmt"
	"shared"
	"slices"
	"strings"
	"sync"
)

type SystemStatus struct {
	mu        sync.RWMutex
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []shared.ActorInfo      `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
	Missions  []mission.Update        `json:"missions"`
}

type SystemStatusSnapshot struct {
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []shared.ActorInfo      `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
	Missions  []mission.Update        `json:"missions"`
}

func (s *SystemStatus) AddSensor(sensor shared.SensorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sensors = append(s.Sensors, sensor)
}

func (s *SystemStatus) AddActor(actor shared.ActorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index := slices.IndexFunc(s.Actors, func(existing shared.ActorInfo) bool {
		return existing.Id == actor.Id
	}); index >= 0 {
		s.Actors[index] = actor
		return
	}
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
		Missions:  append([]mission.Update(nil), s.Missions...),
	}
}

func (s *SystemStatus) IncidentByID(id string) (shared.SensorIncident, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, incident := range s.Incidents {
		if mission.IncidentID(incident) == id {
			return incident, true
		}
	}
	return shared.SensorIncident{}, false
}

func (s *SystemStatus) FirstActorByType(actorType shared.ActorType) (shared.ActorInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, actor := range s.Actors {
		if actor.ActorType == actorType {
			return actor, true
		}
	}
	return shared.ActorInfo{}, false
}

func (s *SystemStatus) UpdateMission(update mission.Update) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if update.MissionID == "" {
		update.MissionID = fmt.Sprintf("%s-%s", update.ActorID, update.UpdatedAt.Format("150405.000"))
	}

	if index := slices.IndexFunc(s.Missions, func(existing mission.Update) bool {
		return existing.MissionID == update.MissionID && existing.ActorID == update.ActorID
	}); index >= 0 {
		s.Missions[index] = update
	} else {
		s.Missions = append(s.Missions, update)
	}

	if index := slices.IndexFunc(s.Actors, func(actor shared.ActorInfo) bool {
		return actor.Id == update.ActorID
	}); index >= 0 {
		s.Actors[index].Position = update.Position
	}

	if update.State == "done" {
		index := strings.LastIndexAny(update.MissionID, "-")
		if index < 0 {
			fmt.Printf("invalid mission ID %s (no '-' found at index %d )", update.MissionID, index)
		}
		incident, _ := s.IncidentByID(update.MissionID[:index])
		s.Incidents = slices.DeleteFunc(s.Incidents, func(i shared.SensorIncident) bool {
			return i.Sensor == incident.Sensor
		})
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
