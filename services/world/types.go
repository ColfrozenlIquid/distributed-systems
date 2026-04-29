package main

type TerrainType int

type InfrastructureType int

const (
	Land TerrainType = iota
	Water
)

const (
	NoInfrastructure InfrastructureType = iota - 1
	Bridge
	Port
	ChargingStation
	LandingStrip
	Depot
)

type SensorReading interface {
	Read() float64
}

type WaterSensor struct {
	WaterLevelCm float64
}

type VibrationSensor struct {
	VibrationIndex float64
}

func (s WaterSensor) Read() float64 { return s.WaterLevelCm }

func (s VibrationSensor) Read() float64 { return s.VibrationIndex }
