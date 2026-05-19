package sensors

import (
	"fmt"
	pb "simulation/gen"
	proto "simulation/gen"
)

type TerrainType int

type InfrastructureType int

type SensorType int

const (
	SensorVibration SensorType = iota
	SensorWaterLevel
	SensorSupplyLevel
	SensorPersonConfidence
)

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

type Simulatable interface {
	Type() pb.SensorType
	Read() float32
	Update()
	Init()
	Color() uint8
	Set(value float32)
}

func ToDomainSensorType(s pb.SensorType) (SensorType, error) {
	switch s {
	case proto.SensorType_SENSOR_TYPE_SUPPLY_LEVEL:
		return SensorSupplyLevel, nil
	case proto.SensorType_SENSOR_TYPE_VIBRATION:
		return SensorVibration, nil
	case proto.SensorType_SENSOR_TYPE_WATER_LEVEL:
		return SensorWaterLevel, nil
	case proto.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE:
		return SensorPersonConfidence, nil
	default:
		return 0, fmt.Errorf("unknown or unspecified sensor type: %v", s)
	}
}

func ToProtoSensorType(s SensorType) (pb.SensorType, error) {
	switch s {
	case SensorSupplyLevel:
		return pb.SensorType_SENSOR_TYPE_SUPPLY_LEVEL, nil
	case SensorVibration:
		return pb.SensorType_SENSOR_TYPE_VIBRATION, nil
	case SensorWaterLevel:
		return pb.SensorType_SENSOR_TYPE_WATER_LEVEL, nil
	case SensorPersonConfidence:
		return pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE, nil
	default:
		return pb.SensorType_SENSOR_TYPE_UNSPECIFIED, fmt.Errorf("unknown sensor type: %v", s)
	}
}
