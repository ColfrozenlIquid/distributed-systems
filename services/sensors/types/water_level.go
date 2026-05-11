package types

import (
	"sensors/config"
	"time"
)

var _ Sensor = (*WaterLevelSensor)(nil)

type WaterLevelSensor struct {
	cfg config.SensorConfig
}

// ID implements [Sensor].
func (w *WaterLevelSensor) ID() string {
	panic("unimplemented")
}

// Interval implements [Sensor].
func (w *WaterLevelSensor) Interval() time.Duration {
	panic("unimplemented")
}

// Read implements [Sensor].
func (w *WaterLevelSensor) Read() (Reading, error) {
	panic("unimplemented")
}

// Type implements [Sensor].
func (w *WaterLevelSensor) Type() string {
	panic("unimplemented")
}
