package simulatables

import (
	"math/rand"
	pb "simulation/gen"
	"simulation/internal/config"
)

var _ Simulatable = (*WaterSim)(nil)

type WaterSim struct {
	WaterLevelCm    float32
	MaxWaterLevel   float32
	MinWaterLevel   float32
	target          float32
	smoothing       float32
	newTargetChance float32
}

func (s *WaterSim) Set(value float32) {
	s.target = s.randomTarget()
	s.WaterLevelCm = value
}

func NewWaterLevelSim(cfg config.Config) *WaterSim {
	s := &WaterSim{
		WaterLevelCm:    cfg.Water.StartingWaterLevelCM,
		MaxWaterLevel:   cfg.Water.MaxWaterLevelCM,
		MinWaterLevel:   cfg.Water.MinWaterLevelCM,
		smoothing:       cfg.Water.WaterLevelSmoothing,
		newTargetChance: cfg.Water.WaterLevelNewTargetChance,
	}
	s.Init()
	return s
}

func (s *WaterSim) Type() pb.SensorType { return pb.SensorType_SENSOR_TYPE_WATER_LEVEL }

func (s *WaterSim) Read() float32 { return s.WaterLevelCm }

func (s *WaterSim) Update() {
	if rand.Float32() < s.newTargetChance {
		s.target = s.randomTarget()
	}

	s.WaterLevelCm += (s.target - s.WaterLevelCm) * s.smoothing
}

func (s *WaterSim) Init() {
	s.target = s.randomTarget()
}

func (s *WaterSim) randomTarget() float32 {
	return s.MinWaterLevel + rand.Float32()*(s.MaxWaterLevel-s.MinWaterLevel)
}
