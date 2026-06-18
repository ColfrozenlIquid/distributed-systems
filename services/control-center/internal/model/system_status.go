package model

import (
	"shared"
	"slices"
	"sync"
)

type SystemStatus struct {
	mu        sync.RWMutex
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []Actor                 `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
	Missions  []Mission               `json:"missions"`
}

type SystemStatusSnapshot struct {
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []Actor                 `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
	Missions  []Mission               `json:"missions"`
}

func (s *SystemStatus) AddSensor(sensor shared.SensorInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sensors = append(s.Sensors, sensor)
}

func (s *SystemStatus) AddActor(actor shared.ActorInfo) {
	actorDomain := ActorFromActorInfo(actor)

	s.mu.Lock()
	defer s.mu.Unlock()
	if index := slices.IndexFunc(s.Actors, func(existing Actor) bool {
		return existing.Id == actorDomain.Id
	}); index >= 0 {
		s.Actors[index] = actorDomain
		return
	}
	s.Actors = append(s.Actors, actorDomain)
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

func (s *SystemStatus) UpdateActorFromStatus(actorStatus shared.ActorStatus) {
	actor, index := s.getActor(actorStatus.ActorID)
	if index < 0 {
		return
	}
	s.Actors[index] = actor.UpdateWithStatus(actorStatus)
}

func (s *SystemStatus) UpdateActorFromTelemetry(actorTelemetry shared.ActorTelemetry) {
	actor, index := s.getActor(actorTelemetry.ActorID)
	if index < 0 {
		return
	}
	s.Actors[index] = actor.UpdateWithTelemetry(actorTelemetry)
}

func (s *SystemStatus) getActor(id string) (actor *Actor, index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor = nil
	index = slices.IndexFunc(s.Actors, func(existing Actor) bool {
		return existing.Id == id
	})
	if index >= 0 {
		actor = &s.Actors[index]
	}
	return actor, index
}

func (s *SystemStatus) AddMission(mission Mission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.ContainsFunc(s.Missions, func(m Mission) bool {
		return m.Id == mission.Id
	}) {
		return
	}
	s.Missions = append(s.Missions, mission)
}

func (s *SystemStatus) UpdateMission(mission shared.MissionUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(s.Missions, func(existing Mission) bool {
		return existing.Id == mission.MissionID
	})
	if index < 0 {
		return
	}

	s.Missions[index] = s.Missions[index].Update(mission)
	switch mission.State {
	case shared.MissionStateCompleted:
		s.Incidents = slices.DeleteFunc(s.Incidents, func(i shared.SensorIncident) bool {
			return i == s.Missions[index].Incident
		})
	case shared.MissionStateFailed:
		// TODO Handle failed missions?
	}
}

func (s *SystemStatus) Snapshot() SystemStatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SystemStatusSnapshot{
		Sensors:   append([]shared.SensorInfo(nil), s.Sensors...),
		Actors:    append([]Actor(nil), s.Actors...),
		Readings:  append([]shared.SensorReading(nil), s.Readings...),
		Incidents: append([]shared.SensorIncident(nil), s.Incidents...),
		Missions:  append([]Mission(nil), s.Missions...),
	}
}

func (s *SystemStatus) IncidentByID(id string) (shared.SensorIncident, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.incidentByIdLocked(id)
}

func (s *SystemStatus) incidentByIdLocked(id string) (shared.SensorIncident, bool) {
	for _, incident := range s.Incidents {
		if incident.Id == id {
			return incident, true
		}
	}
	return shared.SensorIncident{}, false
}

func (s *SystemStatus) FirstIdleActorByType(actorType shared.ActorType) (Actor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, actor := range s.Actors {
		if actor.Type == actorType && actor.State == shared.ActorStateIdle {
			return actor, true
		}
	}
	return Actor{}, false
}

func (s *SystemStatus) addIncidentLocked(incident shared.SensorIncident) {
	if slices.ContainsFunc(s.Incidents, func(i shared.SensorIncident) bool {
		return i.Sensor == incident.Sensor
	}) {
		return
	}
	s.Incidents = append(s.Incidents, incident)
}
