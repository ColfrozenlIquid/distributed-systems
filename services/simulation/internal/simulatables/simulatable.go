package simulatables

import (
	"fmt"
	pb "simulation/gen"
	proto "simulation/gen"
)

type TerrainType int

type InfrastructureType int

type SimulatableType int

const (
	Vibration SimulatableType = iota
	WaterLevel
	SupplyLevel
	PersonConfidence
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
	Set(value float32, setDefault bool) float32
}

func ToDomainSensorType(s pb.SensorType) (SimulatableType, error) {
	switch s {
	case proto.SensorType_SENSOR_TYPE_SUPPLY_LEVEL:
		return SupplyLevel, nil
	case proto.SensorType_SENSOR_TYPE_VIBRATION:
		return Vibration, nil
	case proto.SensorType_SENSOR_TYPE_WATER_LEVEL:
		return WaterLevel, nil
	case proto.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE:
		return PersonConfidence, nil
	default:
		return 0, fmt.Errorf("unknown or unspecified sensor type: %v", s)
	}
}

func ToProtoSensorType(s SimulatableType) (pb.SensorType, error) {
	switch s {
	case SupplyLevel:
		return pb.SensorType_SENSOR_TYPE_SUPPLY_LEVEL, nil
	case Vibration:
		return pb.SensorType_SENSOR_TYPE_VIBRATION, nil
	case WaterLevel:
		return pb.SensorType_SENSOR_TYPE_WATER_LEVEL, nil
	case PersonConfidence:
		return pb.SensorType_SENSOR_TYPE_PERSON_CONFIDENCE, nil
	default:
		return pb.SensorType_SENSOR_TYPE_UNSPECIFIED, fmt.Errorf("unknown sensor type: %v", s)
	}
}

func ToProtoTerrainType(t TerrainType) pb.TerrainType {
	switch t {
	case Land:
		return pb.TerrainType_TERRAIN_TYPE_LAND
	case Water:
		return pb.TerrainType_TERRAIN_TYPE_WATER
	default:
		return pb.TerrainType_TERRAIN_TYPE_UNSPECIFIED
	}
}

func ToProtoInfrastructureType(t InfrastructureType) pb.InfrastructureType {
	switch t {
	case NoInfrastructure:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_NONE
	case ControlCenter:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_CONTROL_CENTER
	case Bridge:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_BRIDGE
	case Port:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_PORT
	case ChargingStation:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_CHARGING_STATION
	case LandingStrip:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_LANDING_STRIP
	case Depot:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_DEPOT
	default:
		return pb.InfrastructureType_INFRASTRUCTURE_TYPE_UNSPECIFIED
	}
}
