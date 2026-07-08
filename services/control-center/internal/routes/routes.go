package routes

import (
	"context"
	"control-center/internal/html"
	"control-center/internal/model"
	"fmt"
	"shared"
	"slices"
	"vshttp"

	"github.com/samber/ro"
)

func Routes(router *vshttp.Router, state *model.State, actors model.ActorMissionClient) {
	missions := model.NewService(actors, state.System())

	router.Post("/reading", func(c *vshttp.Context) error {
		var request shared.SensorReading
		if err := c.JSONBody(&request); err != nil {
			return err
		}
		state.System().AddReading(request)
		return c.NoContent()
	})

	router.Post("/register/sensor", func(c *vshttp.Context) error {
		var request shared.SensorRegistration
		if err := c.JSONBody(&request); err != nil {
			return err
		}
		state.System().AddSensor(request.Sensor)
		return c.NoContent()
	})

	router.Post("/register/actor", func(c *vshttp.Context) error {
		var request shared.ActorRegistration
		if err := c.JSONBody(&request); err != nil {
			return err
		}
		state.System().AddActor(request.Actor)
		return c.NoContent()
	})

	router.Get("/map", func(c *vshttp.Context) error {
		c.Writer.Header().Replace("Content-Type", "text/html; charset=utf-8")
		c.Writer.WriteHeader(vshttp.OK)
		return html.DashboardPage(html.NewDashboardPageData(state.System().Snapshot(), state.World(), state.Ricart().Snapshot())).Render(context.Background(), c.Writer)
	})

	router.Get("/status", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, state.System().Snapshot())
	})

	router.Get("/ricart/status", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, state.Ricart().Snapshot())
	})

	router.Get("/status/incidents", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, state.System().Snapshot().Incidents)
	})

	// Assign a mission to an incident. If the request comes from HTMX (hx-request), return an HTML fragment (incident item) that HTMX can swap in-place.
	router.Post("/incidents/:incidentID/mission", func(c *vshttp.Context) error {
		incidentID := c.Param("incidentID")
		missionID, err := missions.AssignIncident(context.Background(), incidentID)
		if err != nil {
			return vshttp.Error(vshttp.InternalServerError, err.Error())
		}

		// If HTMX requested an HTML response, render the updated incident item
		if c.Header("HX-Request") == "true" || c.Header("Accept") == "text/html" {
			c.Writer.Header().Replace("Content-Type", "text/html; charset=utf-8")
			c.Writer.WriteHeader(vshttp.OK)

			// render current snapshot and extract the incident item to return
			snapshot := state.System().Snapshot()
			page := html.NewDashboardPageData(snapshot, state.World(), state.Ricart().Snapshot())

			if idx := slices.IndexFunc(page.Incidents, func(i html.IncidentListItem) bool { return i.ID == incidentID }); idx >= 0 {
				return html.IncidentItem(page.Incidents[idx]).Render(context.Background(), c.Writer)
			}
			return vshttp.Error(vshttp.NotFound, fmt.Sprintf("incident %q not found", incidentID))
		}

		return c.Text(vshttp.OK, missionID)
	})

	// JSON endpoint returning mission(s) for an incident
	router.Get("/incidents/:incidentID/mission", func(c *vshttp.Context) error {
		incidentID := c.Param("incidentID")
		snapshot := state.System().Snapshot()

		if idx := slices.IndexFunc(snapshot.Missions, func(m model.Mission) bool { return m.Incident.Id == incidentID }); idx >= 0 {
			return c.JSON(vshttp.OK, snapshot.Missions[idx])
		}
		return vshttp.Error(vshttp.NotFound, "mission not found")
	})

	// SSE stream for mission updates (renders templ fragments and sends as SSE data)
	router.Get("/events/missions", func(c *vshttp.Context) error {
		c.Writer.Header().Replace("Content-Type", "text/event-stream; charset=utf-8")
		c.Writer.Header().Replace("Cache-Control", "no-cache")
		c.Writer.WriteHeader(vshttp.OK)

		ctx := context.Background()
		sub := state.System().MissionUpdates().SubscribeWithContext(ctx, ro.NewObserverWithContext(
			func(ctx context.Context, mission model.Mission) {
				_ = html.WriteIncidentItemSSE(ctx, c.Writer, state.System().Snapshot(), mission)
				if flusher, ok := c.Writer.(interface{ Flush() error }); ok {
					_ = flusher.Flush()
				}
			},
			func(ctx context.Context, err error) {},
			func(ctx context.Context) {},
		))
		defer sub.Unsubscribe()

		<-ctx.Done()
		return nil
	})

	router.Get("/health", func(c *vshttp.Context) error {
		return c.Text(vshttp.OK, "control-center is healthy")
	})
}
