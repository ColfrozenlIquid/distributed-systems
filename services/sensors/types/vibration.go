package types

import (
	"sensors/config"
	"time"
)

var _ Sensor = (*VibrationSensor)(nil)

type VibrationSensor struct {
	cfg config.SensorConfig
}

// ID implements [Sensor].
func (v *VibrationSensor) ID() string {
	panic("unimplemented")
}

// Interval implements [Sensor].
func (v *VibrationSensor) Interval() time.Duration {
	panic("unimplemented")
}

// Read implements [Sensor].
func (v *VibrationSensor) Read() (Reading, error) {
	panic("unimplemented")

}

// Type implements [Sensor].
func (v *VibrationSensor) Type() string {
	panic("unimplemented")
}
