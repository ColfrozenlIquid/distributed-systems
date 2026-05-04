package main

import (
	"math/rand"
)

type TerrainType int

type InfrastructureType int

const (
	Land TerrainType = iota
	Water
)

const (
	NoInfrastructure InfrastructureType = iota - 1
	ControlCenter
	Bridge
	Port
	ChargingStation
	LandingStrip
	Depot
)

type SensorReading interface {
	Read() float32
	Update()
	Init()
}

type WaterSensor struct {
	WaterLevelCm  float32
	MaxWaterLevel float32
	MinWaterLevel float32
	target        float32
	smoothing     float32
}

type VibrationSensor struct {
	VibrationIndex    float32
	MaxVibrationIndex float32
	MinVibrationIndex float32
	target            float32
	smoothing         float32
}

func (s *WaterSensor) Read() float32 { return s.WaterLevelCm }

func (s *VibrationSensor) Read() float32 { return s.VibrationIndex }

func (s *WaterSensor) Update() {
	if rand.Float32() < 0.02 {
		s.target = s.randomTarget()
	}

	s.WaterLevelCm += (s.target - s.WaterLevelCm) * s.smoothing
}

func (s *WaterSensor) Init() {
	s.target = s.randomTarget()
}

func (s *WaterSensor) randomTarget() float32 {
	return s.MinWaterLevel + rand.Float32()*(s.MaxWaterLevel-s.MinWaterLevel)
}

func (w *VibrationSensor) Init() {
	w.target = w.randomTarget()
}

func (s *VibrationSensor) randomTarget() float32 {
	return s.MinVibrationIndex + rand.Float32()*(s.MaxVibrationIndex-s.MinVibrationIndex)
}

func (s *VibrationSensor) Update() {
	if rand.Float32() < 0.02 {
		s.target = s.randomTarget()
	}

	s.VibrationIndex += (s.target - s.VibrationIndex) * s.smoothing
}

func NewWaterSensor() *WaterSensor {
	s := &WaterSensor{
		WaterLevelCm:  50.0,
		MaxWaterLevel: 100.0,
		MinWaterLevel: 0.0,
		smoothing:     0.05,
	}
	s.Init()
	return s
}

func NewVibrationSensor() *VibrationSensor {
	s := &VibrationSensor{
		VibrationIndex:    0.0,
		MaxVibrationIndex: 10.0,
		MinVibrationIndex: 0.0,
		smoothing:         0.05,
	}
	s.Init()
	return s
}
