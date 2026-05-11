package config

import (
	"os"
	"strconv"
	"time"
)

const SENSOR_ID string = "SENSOR_ID"
const SENSOR_TYPE string = "SENSOR_TYPE"
const SENSOR_INTERVAL_MS string = "SENSOR_INTERVAL_MS"
const SENSOR_POSITION_X string = "SENSOR_POSITION_X"
const SENSOR_POSITION_Y string = "SENSOR_POSITION_Y"
const SENSOR_BASE_STATION_ENDPOINT string = "SENSOR_BASE_STATION_ENDPOINT"
const SENSOR_WORLD_SIMULATION_ENDPOINT string = "SENSOR_WORLD_SIMULATION_ENDPOINT"

type SensorConfig struct {
	Id                      string        // Sensor ID
	SensorType              string        // Sensor Type
	IntervalMs              time.Duration // How often does the sensor produce a reading
	XPosition               int32         // Where does the sensor start on the x-axis
	YPosition               int32         // Where does the sensor start on the y-axis
	BaseStationEndpoint     string        // HTTP endpoint for sending data to the control-center
	WorldSimulationEndpoint string        // gRPC endpoint for extracting data from the simulated world
}

func LoadConfig() (*SensorConfig, error) {
	intervalMS, _ := strconv.Atoi(os.Getenv(SENSOR_INTERVAL_MS))
	xPos, _ := strconv.Atoi(os.Getenv(SENSOR_POSITION_X))
	yPos, _ := strconv.Atoi(os.Getenv(SENSOR_POSITION_Y))

	cfg := &SensorConfig{
		Id:                      os.Getenv(SENSOR_ID),
		SensorType:              os.Getenv(SENSOR_TYPE),
		IntervalMs:              time.Duration(intervalMS) * time.Millisecond,
		XPosition:               int32(xPos),
		YPosition:               int32(yPos),
		BaseStationEndpoint:     os.Getenv(SENSOR_BASE_STATION_ENDPOINT),
		WorldSimulationEndpoint: os.Getenv(SENSOR_WORLD_SIMULATION_ENDPOINT),
	}
	return cfg, nil
}
