package main

import "time"

type World struct {
	Width  int
	Height int
	Tiles  [][]Tile
}

type Tile struct {
	Terrain        TerrainType
	Infrastructure InfrastructureType
	Sensor         SensorReading
}

func (t *Tile) update() {
	t.Sensor.Update()
}

func main() {
	world := NewWorld(20, 20)
	world.init()

	for {
		world.update()
		time.Sleep(2 * time.Second)
	}
}

func NewWorld(width int, height int) World {
	tiles := make([][]Tile, height)
	for y := range tiles {
		tiles[y] = make([]Tile, width)
	}
	return World{
		Width:  width,
		Height: height,
		Tiles:  tiles,
	}
}

func (w *World) init() {
	for i := 0; i < len(w.Tiles); i++ {
		for j := 0; j < len(w.Tiles[i]); j++ {
			t := w.Tiles[i][j]
			t.Sensor.Init()
		}
	}
}

func (w *World) update() {
	for i := 0; i < len(w.Tiles); i++ {
		for j := 0; j < len(w.Tiles[i]); j++ {
			t := w.Tiles[i][j]
			t.Sensor.Update()
		}
	}
}
