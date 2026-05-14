package main

import (
	"context"
	"shared"
	"simulation/world"
	"vshttp"
)

func Routes(router *vshttp.Router) {
	systemStatus := SystemStatus{}
	island := world.NewWorld(20, 20)

	router.Post("/reading", func(c *vshttp.Context) error {
		var request shared.SensorReading
		if err := c.JSONBody(&request); err != nil {
			return err
		}
		systemStatus.AddReading(request)
		return c.NoContent()
	})

	router.Post("/register/sensor", func(c *vshttp.Context) error {
		var request shared.SensorRegistration
		if err := c.JSONBody(&request); err != nil {
			return err
		}
		systemStatus.AddSensor(request.Sensor)
		return c.NoContent()
	})

	router.Get("/map", func(c *vshttp.Context) error {
		c.Writer.Header().Replace("Content-Type", "text/html; charset=utf-8")
		c.Writer.WriteHeader(vshttp.OK)
		return DashboardPage(NewDashboardPageData(systemStatus.Snapshot(), &island)).Render(context.Background(), c.Writer)
	})

	router.Get("/status", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, systemStatus.Snapshot())
	})
}
