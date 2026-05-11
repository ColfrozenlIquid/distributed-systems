package world

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"

	"simulation/sensors"
)

//go:embed terrain.map
var terrainMap []byte

type Tile struct {
	Terrain        sensors.TerrainType
	Infrastructure sensors.InfrastructureType
	Sensors        map[sensors.SensorType]sensors.Simulatable
}

type TileVisual struct {
	Char rune
	FG   string
	BG   string
}

func (t *Tile) Read(sensor sensors.SensorType) (float32, bool) {
	sim, ok := t.Sensors[sensor]
	if !ok {
		return 0, false
	}
	return sim.Read(), true
}

func RenderTile(t sensors.TerrainType, i sensors.InfrastructureType, sensor uint8) TileVisual {
	v := TileVisual{
		FG: "\033[37m",
	}

	switch i {
	case sensors.ControlCenter:
		v.Char = 'C'
	case sensors.Bridge:
		v.Char = 'B'
	case sensors.ChargingStation:
		v.Char = 'S'
	case sensors.Port:
		v.Char = 'P'
	case sensors.Depot:
		v.Char = 'D'
	default:
		if t == sensors.Land {
			v.Char = '.'
		} else {
			v.Char = '~'
		}
	}

	v.BG = SensorColor(sensor)

	return v
}

func SensorColor(sensor uint8) string {
	value := int(sensor)

	var color int
	switch {
	case value < 128:
		t := float64(value) / 128.0
		color = int((1-t)*46 + t*226)
	default:
		t := float64(value-128) / 127.0
		color = int((1-t)*226 + t*196)
	}

	return fmt.Sprintf("\033[48;5;%dm", color)
}

func GenerateTiles() ([][]Tile, error) {
	var tiles [][]Tile

	scanner := bufio.NewScanner(bytes.NewReader(terrainMap))
	for scanner.Scan() {
		line := scanner.Text()
		var row []Tile
		for _, ch := range line {
			switch ch {
			case '~': //Water
				row = append(row, Tile{
					Terrain:        sensors.Water,
					Infrastructure: sensors.NoInfrastructure,
					Sensors: map[sensors.SensorType]sensors.Simulatable{
						sensors.SensorWaterLevel: sensors.NewWaterLevelSim(),
					},
				})
			case '#': //Land
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.NoInfrastructure,
					Sensors: map[sensors.SensorType]sensors.Simulatable{
						sensors.SensorVibration: sensors.NewVibrationSim(),
					},
				})
			case 'C': //Control Station
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.ControlCenter,
					Sensors: map[sensors.SensorType]sensors.Simulatable{
						sensors.SensorWaterLevel: sensors.NewWaterLevelSim(),
					},
				})
			case 'B': //Bridge
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.Bridge,
					Sensors:        nil,
				})
			case 'P': //Port
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.Port,
					Sensors:        nil,
				})
			case 'S': //ChargingStation
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.ChargingStation,
					Sensors:        nil,
				})
			case 'L': //LandingStrip
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.LandingStrip,
					Sensors:        nil,
				})
			case 'D': //Depot
				row = append(row, Tile{
					Terrain:        sensors.Land,
					Infrastructure: sensors.Depot,
					Sensors:        nil,
				})
			}
		}
		tiles = append(tiles, row)
	}
	return tiles, nil
}
