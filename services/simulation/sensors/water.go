package sensors

import (
	"math/rand"
	"simulation/config"
	pb "simulation/gen/proto"
)

var _ Simulatable = (*WaterSim)(nil)

type WaterSim struct {
	WaterLevelCm  float32
	MaxWaterLevel float32
	MinWaterLevel float32
	target        float32
	smoothing     float32
}

func (s *WaterSim) Set(value float32) {
	s.target = s.randomTarget()
	s.WaterLevelCm = config.STARTING_WATER_LEVEL_CM
}

func NewWaterLevelSim() *WaterSim {
	s := &WaterSim{
		WaterLevelCm:  config.STARTING_WATER_LEVEL_CM,
		MaxWaterLevel: config.MAX_WATER_LEVEL_CM,
		MinWaterLevel: config.MIN_WATER_LEVEL_CM,
		smoothing:     config.WATER_LEVEL_SMOOTHING,
	}
	s.Init()
	return s
}

func (s *WaterSim) Type() pb.SensorType { return pb.SensorType_SENSOR_TYPE_WATER_LEVEL }

func (s *WaterSim) Color() uint8 {
	reading := (s.WaterLevelCm - s.MinWaterLevel) / (s.MaxWaterLevel - s.MinWaterLevel)
	if reading < 0 {
		reading = 0
	}
	if reading > 1 {
		reading = 1
	}
	return uint8(reading * 255)
}

func (s *WaterSim) Read() float32 { return s.WaterLevelCm }

func (s *WaterSim) Update() {
	if rand.Float32() < 0.02 {
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
