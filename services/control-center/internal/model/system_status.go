package model

import (
	"fmt"
	"shared"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type SystemStatus struct {
	mu          sync.RWMutex
	Sensors     []shared.SensorInfo `json:"sensors"`
	Actors      []shared.ActorInfo  `json:"actors"`
	ActorsNew   []Actor
	Readings    []shared.SensorReading  `json:"readings"`
	Incidents   []shared.SensorIncident `json:"incidents"`
	Missions    []Update                `json:"missions"`
	MissionsNew []Mission
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
	if index := slices.IndexFunc(s.ActorsNew, func(existing Actor) bool {
		return existing.Id == actorDomain.Id
	}); index >= 0 {
		s.ActorsNew[index] = actorDomain
		return
	}
	s.ActorsNew = append(s.ActorsNew, actorDomain)
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

func (s *SystemStatus) UpdateActorFromStatus(actorStatus shared.ActorStatus) {
	actor, index := s.getActor(actorStatus.ActorID)
	if index < 0 {
		return
	}
	s.ActorsNew[index] = actor.UpdateWithStatus(actorStatus)
}

func (s *SystemStatus) UpdateActorFromTelemetry(actorTelemetry shared.ActorTelemetry) {
	actor, index := s.getActor(actorTelemetry.ActorID)
	if index < 0 {
		return
	}
	s.ActorsNew[index] = actor.UpdateWithTelemetry(actorTelemetry)
}

func (s *SystemStatus) getActor(id string) (actor *Actor, index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor = nil
	index = slices.IndexFunc(s.ActorsNew, func(existing Actor) bool {
		return existing.Id == id
	})
	if index >= 0 {
		actor = &s.ActorsNew[index]
	}
	return actor, index
}

func (s *SystemStatus) UpdateMissionNew(mission shared.MissionUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := slices.IndexFunc(s.MissionsNew, func(existing Mission) bool {
		return existing.Id == mission.MissionID
	})
	if index < 0 {
		return
	}

	s.MissionsNew[index] = s.MissionsNew[index].Update(mission)
	switch mission.State {
	case shared.MissionStateCompleted:
		s.Incidents = slices.DeleteFunc(s.Incidents, func(i shared.SensorIncident) bool {
			return i == s.MissionsNew[index].Incident
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
		Actors:    append([]Actor(nil), s.ActorsNew...),
		Readings:  append([]shared.SensorReading(nil), s.Readings...),
		Incidents: append([]shared.SensorIncident(nil), s.Incidents...),
		Missions:  append([]Mission(nil), s.MissionsNew...),
	}
}

func (s *SystemStatus) IncidentByID(id string) (shared.SensorIncident, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.incidentByIdLocked(id)
}

func (s *SystemStatus) incidentByIdLocked(id string) (shared.SensorIncident, bool) {
	for _, incident := range s.Incidents {
		if IncidentID(incident) == id {
			return incident, true
		}
	}
	return shared.SensorIncident{}, false
}

func (s *SystemStatus) FirstActorByType(actorType shared.ActorType) (Actor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, actor := range s.ActorsNew {
		if actor.Type == actorType {
			return actor, true
		}
	}
	return Actor{}, false
}

func (s *SystemStatus) UpdateMission(update Update) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if update.MissionID == "" {
		update.MissionID = fmt.Sprintf("%s-%s", update.ActorID, update.UpdatedAt.Format("150405.000"))
	}

	if index := slices.IndexFunc(s.Missions, func(existing Update) bool {
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
			return
		}
		incident, _ := s.incidentByIdLocked(update.MissionID[:index])
		s.Incidents = slices.DeleteFunc(s.Incidents, func(i shared.SensorIncident) bool {
			return i.Sensor == incident.Sensor
		})
	}
}

func (s *SystemStatus) addIncidentLocked(incident shared.SensorIncident) {
	if incident.Id == "" {
		id, err := uuid.NewV7()
		if err != nil {
			panic(err)
		}
		incident.Id = id.String()
	}
	if slices.ContainsFunc(s.Incidents, func(i shared.SensorIncident) bool {
		return i.Sensor == incident.Sensor
	}) {
		return
	}
	s.Incidents = append(s.Incidents, incident)
}
