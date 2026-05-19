package world

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"simulation/internal/config"
	sim "simulation/internal/simulatables"
)

//go:embed terrain.map
var terrainMap []byte

type Tile struct {
	Terrain        sim.TerrainType
	Infrastructure sim.InfrastructureType
	Sensors        map[sim.SimulatableType]sim.Simulatable
}

type TileVisual struct {
	Char rune
	FG   string
	BG   string
}

func (t *Tile) Read(sensor sim.SimulatableType) (float32, bool) {
	sim, ok := t.Sensors[sensor]
	if !ok {
		return 0, false
	}
	return sim.Read(), true
}

func (t *Tile) Set(sensor sim.SimulatableType, value float32) bool {
	sim, ok := t.Sensors[sensor]
	if !ok {
		return false
	}
	sim.Set(value)

	return true
}

func RenderTile(t sim.TerrainType, i sim.InfrastructureType, sensor uint8) TileVisual {
	v := TileVisual{
		FG: "\033[37m",
	}

	switch i {
	case sim.ControlCenter:
		v.Char = 'C'
	case sim.Bridge:
		v.Char = 'B'
	case sim.ChargingStation:
		v.Char = 'S'
	case sim.Port:
		v.Char = 'P'
	case sim.Depot:
		v.Char = 'D'
	default:
		if t == sim.Land {
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

func GenerateTiles(cfg config.Config) ([][]Tile, error) {
	var tiles [][]Tile

	scanner := bufio.NewScanner(bytes.NewReader(terrainMap))
	for scanner.Scan() {
		line := scanner.Text()
		var row []Tile
		for _, ch := range line {
			switch ch {
			case '~': //Water
				row = append(row, Tile{
					Terrain:        sim.Water,
					Infrastructure: sim.NoInfrastructure,
					Sensors: map[sim.SimulatableType]sim.Simulatable{
						sim.WaterLevel: sim.NewWaterLevelSim(cfg),
					},
				})
			case '#': //Land
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.NoInfrastructure,
					Sensors: map[sim.SimulatableType]sim.Simulatable{
						sim.Vibration: sim.NewVibrationSim(cfg),
					},
				})
			case 'C': //Control Station
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.ControlCenter,
					Sensors: map[sim.SimulatableType]sim.Simulatable{
						sim.WaterLevel: sim.NewWaterLevelSim(cfg),
					},
				})
			case 'B': //Bridge
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.Bridge,
					Sensors:        nil,
				})
			case 'P': //Port
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.Port,
					Sensors:        nil,
				})
			case 'S': //ChargingStation
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.ChargingStation,
					Sensors:        nil,
				})
			case 'L': //LandingStrip
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.LandingStrip,
					Sensors:        nil,
				})
			case 'D': //Depot
				row = append(row, Tile{
					Terrain:        sim.Land,
					Infrastructure: sim.Depot,
					Sensors:        nil,
				})
			}
		}
		tiles = append(tiles, row)
	}
	return tiles, nil
}
