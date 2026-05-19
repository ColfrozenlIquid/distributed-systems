package simulatables

import (
	"math/rand"
	pb "simulation/gen"
	"simulation/internal/config"
)

var _ Simulatable = (*VibrationSim)(nil)

type VibrationSim struct {
	VibrationIndex    float32
	MaxVibrationIndex float32
	MinVibrationIndex float32
	target            float32
	smoothing         float32
}

func (s *VibrationSim) Set(value float32) {
	s.target = s.randomTarget()
	s.VibrationIndex = config.STARTING_VIBRATION_INDEX
}

func NewVibrationSim(cfg config.Config) *VibrationSim {
	s := &VibrationSim{
		VibrationIndex:    cfg.Vibration.StartingVibrationIndex,
		MaxVibrationIndex: cfg.Vibration.MaxVibrationIndex,
		MinVibrationIndex: cfg.Vibration.MinVibrationIndex,
		smoothing:         cfg.Vibration.VibrationIndexSmoothing,
	}
	s.Init()
	return s
}

func (s *VibrationSim) Type() pb.SensorType { return pb.SensorType_SENSOR_TYPE_VIBRATION }

func (s *VibrationSim) Color() uint8 {
	reading := (s.VibrationIndex - s.MinVibrationIndex) / (s.MaxVibrationIndex - s.MinVibrationIndex)
	if reading < 0 {
		reading = 0
	}
	if reading > 1 {
		reading = 1
	}
	return uint8(reading * 255)
}

func (s *VibrationSim) Read() float32 { return s.VibrationIndex }

func (w *VibrationSim) Init() {
	w.target = w.randomTarget()
}

func (s *VibrationSim) randomTarget() float32 {
	return s.MinVibrationIndex + rand.Float32()*(s.MaxVibrationIndex-s.MinVibrationIndex)
}

func (s *VibrationSim) Update() {
	if rand.Float32() < 0.02 {
		s.target = s.randomTarget()
	}

	s.VibrationIndex += (s.target - s.VibrationIndex) * s.smoothing
}
