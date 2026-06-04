package simulatables

import (
	pb "simulation/gen"
	proto "simulation/gen"
)

var _ Simulatable = (*PersonsSim)(nil)

type PersonsSim struct {
	Count int
}

func NewPersonsSim(count int) *PersonsSim {
	return &PersonsSim{
		Count: count,
	}
}

func (p *PersonsSim) Init() {
	// Do nothing
}

func (p *PersonsSim) Read() float32 {
	return float32(p.Count)
}

func (p *PersonsSim) Set(value float32) {
	p.Count = int(value)
}

func (p *PersonsSim) Type() proto.SensorType {
	return pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE
}

func (p *PersonsSim) Update() {
	// Do nothing
}
