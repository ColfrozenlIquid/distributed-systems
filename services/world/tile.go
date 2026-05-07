package main

import (
	"bufio"
	"fmt"
	"os"
)

type Tile struct {
	Terrain        TerrainType
	Infrastructure InfrastructureType
	Sensors        map[SensorType]Simulatable
}

type TileVisual struct {
	Char rune
	FG   string
	BG   string
}

func (t *Tile) Read(sensor SensorType) (float32, bool) {
	sim, ok := t.Sensors[sensor]
	if !ok {
		return 0, false
	}
	return sim.Read(), true
}

func RenderTile(t TerrainType, i InfrastructureType, sensor uint8) TileVisual {
	v := TileVisual{
		FG: "\033[37m",
	}

	switch i {
	case ControlCenter:
		v.Char = 'C'
	case Bridge:
		v.Char = 'B'
	case ChargingStation:
		v.Char = 'S'
	case Port:
		v.Char = 'P'
	case Depot:
		v.Char = 'D'
	default:
		if t == Land {
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
	data, err := os.Open("terrain.map")
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	defer data.Close()

	var tiles [][]Tile

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		line := scanner.Text()
		var row []Tile
		for _, ch := range line {
			switch ch {
			case '~': //Water
				row = append(row, Tile{
					Terrain:        Water,
					Infrastructure: NoInfrastructure,
					Sensors: map[SensorType]Simulatable{
						SensorWaterLevel: NewWaterLevelSim(),
					},
				})
			case '#': //Land
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: NoInfrastructure,
					Sensors: map[SensorType]Simulatable{
						SensorVibration: NewVibrationSim(),
					},
				})
			case 'C': //Control Station
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: ControlCenter,
					Sensors: map[SensorType]Simulatable{
						SensorWaterLevel: NewWaterLevelSim(),
					},
				})
			case 'B': //Bridge
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: Bridge,
					Sensors:        nil,
				})
			case 'P': //Port
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: Port,
					Sensors:        nil,
				})
			case 'S': //ChargingStation
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: ChargingStation,
					Sensors:        nil,
				})
			case 'L': //LandingStrip
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: LandingStrip,
					Sensors:        nil,
				})
			case 'D': //Depot
				row = append(row, Tile{
					Terrain:        Land,
					Infrastructure: Depot,
					Sensors:        nil,
				})
			}
		}
		tiles = append(tiles, row)
	}
	return tiles, nil
}
