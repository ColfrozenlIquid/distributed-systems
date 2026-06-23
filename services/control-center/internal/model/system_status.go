package model

import (
	"shared"
	"slices"
	"sync"
	"time"

	"github.com/samber/ro"
)

type SystemStatus struct {
	mu             sync.RWMutex
	Sensors        []shared.SensorInfo     `json:"sensors"`
	Actors         []Actor                 `json:"actors"`
	Readings       []shared.SensorReading  `json:"readings"`
	Incidents      []shared.SensorIncident `json:"incidents"`
	Missions       []Mission               `json:"missions"`
	missionUpdates ro.Subject[Mission]
}

type SystemStatusSnapshot struct {
	Sensors   []shared.SensorInfo     `json:"sensors"`
	Actors    []Actor                 `json:"actors"`
	Readings  []shared.SensorReading  `json:"readings"`
	Incidents []shared.SensorIncident `json:"incidents"`
	Missions  []Mission               `json:"missions"`
}

func NewSystemStatus() SystemStatus {
	return SystemStatus{
		missionUpdates: ro.NewPublishSubject[Mission](),
	}
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
		s.Sensors[index] = reading.Sensor
	}
}

func (s *SystemStatus) UpdateActorFromStatus(actorStatus shared.ActorStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(s.Actors, func(existing Actor) bool {
		return existing.Id == actorStatus.ActorID
	})
	if index < 0 {
		return
	}
	actor := s.Actors[index]
	if actor.State == shared.ActorStateOnAMission &&
		(actorStatus.State == shared.ActorStateError || actorStatus.State == shared.ActorStateOffline) {
		s.failMissionOfActorLocked(actor.Id)
	}
	s.Actors[index] = s.Actors[index].UpdateWithStatus(actorStatus)
}

func (s *SystemStatus) UpdateActorFromTelemetry(actorTelemetry shared.ActorTelemetry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(s.Actors, func(existing Actor) bool {
		return existing.Id == actorTelemetry.ActorID
	})
	if index < 0 {
		return
	}
	s.Actors[index] = s.Actors[index].UpdateWithTelemetry(actorTelemetry)
}

func (s *SystemStatus) failMissionOfActorLocked(actorId string) {
	index := slices.IndexFunc(s.Missions, func(m Mission) bool {
		return m.ActorId == actorId
	})
	if index < 0 {
		return
	}
	s.updateMissionLocked(index, shared.MissionUpdate{
		MissionID: s.Missions[index].Id,
		ActorID:   s.Missions[index].ActorId,
		State:     shared.MissionStateFailed,
		Timestamp: time.Now(),
	})
}

func (s *SystemStatus) AddMission(mission Mission) {
	s.mu.Lock()
	if slices.ContainsFunc(s.Missions, func(m Mission) bool {
		return m.Id == mission.Id
	}) {
		s.mu.Unlock()
		return
	}
	s.Missions = append(s.Missions, mission)
	s.mu.Unlock()
	s.publishMissionUpdate(mission)
}

func (s *SystemStatus) UpdateMission(mission shared.MissionUpdate) {
	s.mu.Lock()
	index := slices.IndexFunc(s.Missions, func(existing Mission) bool {
		return existing.Id == mission.MissionID
	})
	if index < 0 {
		s.mu.Unlock()
		return
	}
	s.updateMissionLocked(index, mission)
	s.mu.Unlock()
}

func (s *SystemStatus) updateMissionLocked(index int, mission shared.MissionUpdate) {
	updated := s.Missions[index].Update(mission)
	s.Missions[index] = updated

	if mission.State == shared.MissionStateCompleted {
		s.Incidents = slices.DeleteFunc(s.Incidents, func(i shared.SensorIncident) bool {
			return i == updated.Incident
		})
	}
	s.publishMissionUpdate(updated)
}

func (s *SystemStatus) MissionUpdates() ro.Observable[Mission] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.missionUpdates == nil {
		s.missionUpdates = ro.NewPublishSubject[Mission]()
	}
	return s.missionUpdates
}

func (s *SystemStatus) publishMissionUpdate(mission Mission) {
	if s.missionUpdates == nil {
		return
	}
	s.missionUpdates.Next(mission)
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
