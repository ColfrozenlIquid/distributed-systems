package sensor

import (
	"sensors/internal/config"
	"shared"
	"time"
)

type BaseSensor struct {
	id         string
	position   shared.Position
	interval   time.Duration
	sensorType string
}

func NewBaseSensor(cfg config.SensorConfig) BaseSensor {
	return BaseSensor{
		id: cfg.Id,
		position: shared.Position{
			X: cfg.XPosition,
			Y: cfg.YPosition,
		},
		interval:   cfg.IntervalMs,
		sensorType: cfg.SensorType,
	}
}

func (b *BaseSensor) ID() string { return b.id }

func (b *BaseSensor) Type() string { return b.sensorType }

func (b *BaseSensor) Position() shared.Position { return b.position }

func (b *BaseSensor) Interval() time.Duration { return b.interval }
