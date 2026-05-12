package main

import (
	"shared"
	"vshttp"
)

func Routes(router *vshttp.Router) {
	systemStatus := SystemStatus{}

	router.Post("/reading", func(c *vshttp.Context) error {
		var request *shared.SensorReading
		if err := c.JSONBody(request); err != nil {
			return err
		}
		systemStatus.AddReading(*request)
		return c.NoContent()
	})

	router.Post("/register/sensor", func(c *vshttp.Context) error {
		var request *shared.SensorRegistration
		if err := c.JSONBody(request); err != nil {
			return err
		}
		systemStatus.AddSensor(request.Sensor)
		return c.NoContent()
	})

	router.Get("/map", func(c *vshttp.Context) error {
		return c.NoContent()
	})

	router.Get("/status", func(c *vshttp.Context) error {
		return c.JSON(vshttp.OK, systemStatus)
	})
}
