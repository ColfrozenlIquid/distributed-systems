package routes

import (
	"context"
	"control-center/internal/html"
	"control-center/internal/model"
	"shared"
	"vshttp"
)

func Routes(router *vshttp.Router, state *model.State) {
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
		return html.DashboardPage(html.NewDashboardPageData(state.System().Snapshot(), state.World())).Render(context.Background(), c.Writer)
	})

	router.Get("/status", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, state.System().Snapshot())
	})

	router.Get("/health", func(c *vshttp.Context) error {
		return c.Text(vshttp.OK, "control-center is healthy")
	})
}
