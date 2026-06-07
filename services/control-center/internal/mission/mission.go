package mission

import (
	"context"
	"fmt"
	"time"

	pb "control-center/gen"
	"shared"
)

type ActorMissionClient interface {
	AssignMission(ctx context.Context, actorId string, mission *pb.Mission) (<-chan Update, error)
}

type Update struct {
	MissionID string           `json:"mission_id"`
	ActorID   string           `json:"actor_id"`
	ActorType shared.ActorType `json:"actor_type"`
	State     string           `json:"state"`
	Position  shared.Position  `json:"position"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type ActorSelector interface {
	IncidentByID(id string) (shared.SensorIncident, bool)
	FirstActorByType(actorType shared.ActorType) (shared.ActorInfo, bool)
}

type Service struct {
	actors ActorMissionClient
	state  ActorSelector
}

func NewService(actors ActorMissionClient, state ActorSelector) *Service {
	return &Service{
		actors: actors,
		state:  state,
	}
}

func (s *Service) AssignIncident(ctx context.Context, incidentID string) (<-chan Update, error) {
	if s == nil || s.actors == nil {
		return nil, fmt.Errorf("actor gRPC client is not configured")
	}

	incident, ok := s.state.IncidentByID(incidentID)
	if !ok {
		return nil, fmt.Errorf("incident %q not found", incidentID)
	}

	missionType, actorType, ok := mapIncidentToMission(incident.Incident)
	if !ok {
		return nil, fmt.Errorf("incident %q has no supported mission mapping", incidentID)
	}

	actor, ok := s.state.FirstActorByType(actorType)
	if !ok {
		return nil, fmt.Errorf("no registered actor of type %q available for incident %q", actorType, incidentID)
	}

	missionID := fmt.Sprintf("%s-%d-%d", incident.Sensor.Id, int(incident.Incident), incident.Timestamp.UnixMilli())
	updates, err := s.actors.AssignMission(ctx, actor.Id, &pb.Mission{
		Id:   &pb.MissionId{Uuid: missionID},
		Type: missionType,
		Position: &pb.Position{
			X: incident.Sensor.Position.X,
			Y: incident.Sensor.Position.Y,
		},
		Priority: pb.MissionPriority_MISSION_PRIORITY_HIGH,
	})
	if err != nil {
		return nil, err
	}

	enriched := make(chan Update)
	go func() {
		defer close(enriched)
		for update := range updates {
			update.ActorID = actor.Id
			update.ActorType = actor.ActorType
			if update.MissionID == "" {
				update.MissionID = missionID
			}
			enriched <- update
		}
	}()

	return enriched, nil
}

func IncidentID(incident shared.SensorIncident) string {
	return fmt.Sprintf("%s-%d", incident.Sensor.Id, int(incident.Incident))
}

func mapIncidentToMission(incident shared.Incident) (pb.MissionType, shared.ActorType, bool) {
	switch incident {
	case shared.WaterLevelAlert:
		return pb.MissionType_MISSION_TYPE_PUMP_WATER, "", false
	case shared.VibrationTooHigh:
		return pb.MissionType_MISSION_TYPE_STABILIZE_VIBRATION, shared.VibrationVehicleActor, true
	case shared.PersonDetected:
		return pb.MissionType_MISSION_TYPE_RESCUE_PERSONS, shared.RescueVehicleActor, true
	default:
		return pb.MissionType_MISSION_TYPE_UNSPECIFIED, "", false
	}
}
