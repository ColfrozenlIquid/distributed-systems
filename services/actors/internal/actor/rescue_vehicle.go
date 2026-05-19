package actor

import (
	"actors/internal/config"
	"time"

	pb "actors/gen"
)

var _ Actor = (*RescueVehicle)(nil)

type RescueVehicle struct {
	cfg        config.ActorConfig
	simulation pb.WorldServiceClient
	actor      rescueVehicle
}

func (v *RescueVehicle) Type() string {
	return "rescue_vehicle"
}

func (v *RescueVehicle) Id() string {
	return v.actor.id
}

func (v *RescueVehicle) Interval() time.Duration {
	return v.actor.interval
}

func (v *RescueVehicle) Position() Position {
	return v.actor.position
}

type rescueVehicle struct {
	id          string
	position    Position
	interval    time.Duration
	reading     float32
	destination Position
	status      ActorStatus
}

func NewRescueVehicle(cfg config.ActorConfig, simulation pb.WorldServiceClient) *RescueVehicle {
	return &RescueVehicle{
		cfg:        cfg,
		simulation: simulation,
		actor: rescueVehicle{
			id: cfg.Id,
			position: Position{
				X: int(cfg.XPosition),
				Y: int(cfg.YPosition),
			},
			interval: cfg.IntervalMs,
			reading:  0,
			destination: Position{
				X: 0,
				Y: 0,
			},
			status: ActorStatusIdle,
		},
	}
}
