package model

import (
	"context"
	"fmt"
	"time"

	"shared"
)

type ActorMissionClient interface {
	AssignMission(ctx context.Context, mission Mission) error
}

type ActorAndMissionManager interface {
	IncidentByID(id string) (shared.SensorIncident, bool)
	FirstIdleActorByType(actorType shared.ActorType) (Actor, bool)
	AddMission(mission Mission)
}

type MissionService struct {
	actors ActorMissionClient
	state  ActorAndMissionManager
}

func NewService(actors ActorMissionClient, state ActorAndMissionManager) *MissionService {
	return &MissionService{
		actors: actors,
		state:  state,
	}
}

func (s *MissionService) AssignIncident(ctx context.Context, incidentID string) (string, error) {
	incident, ok := s.state.IncidentByID(incidentID)
	if !ok {
		return "", fmt.Errorf("incident %q not found", incidentID)
	}

	actorType := incident.Incident.ToActorType()
	actor, ok := s.state.FirstIdleActorByType(actorType)
	if !ok {
		return "", fmt.Errorf("no registered actor of type %q available for incident %q", actorType, incidentID)
	}

	mission := Mission{
		Id:        fmt.Sprintf("%s/%s", actorType, incident.Id),
		ActorId:   actor.Id,
		Incident:  incident,
		State:     shared.MissionStateAssigned,
		UpdatedAt: time.Now(),
	}
	s.state.AddMission(mission)

	return mission.Id, s.actors.AssignMission(ctx, mission)
}
