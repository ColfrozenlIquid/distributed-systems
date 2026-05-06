package main

import (
	"fmt"
	"strings"
	"time"
)

type World struct {
	Width  int
	Height int
	Tiles  [][]Tile
}

func (t *Tile) update() {
	t.Sensor.Update()
}

func main() {
	world := NewWorld(20, 20)

	for {
		world.update()
		fmt.Print("\033[H\033[2J")
		fmt.Print(world.DrawMap())
		time.Sleep(2000 * time.Millisecond)
	}
}

func (w *World) DrawMap() string {
	var sb strings.Builder

	for y := 0; y < len(w.Tiles); y++ {
		for x := 0; x < len(w.Tiles[y]); x++ {
			tile := w.Tiles[y][x]

			var reading uint8 = 0
			if tile.Sensor != nil {
				reading = tile.Sensor.Color()
			}

			vis := RenderTile(tile.Terrain, tile.Infrastructure, reading)

			sb.WriteString(vis.BG)
			sb.WriteString(vis.FG)
			sb.WriteRune(vis.Char)
			sb.WriteString("\033[0m")
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func NewWorld(width int, height int) World {
	tiles := make([][]Tile, height)
	tiles, err := GenerateTiles()
	if err != nil {
		panic("world is nil")
	}
	return World{
		Width:  width,
		Height: height,
		Tiles:  tiles,
	}
}

func (w *World) update() {
	for i := 0; i < len(w.Tiles); i++ {
		for j := 0; j < len(w.Tiles[i]); j++ {
			t := &w.Tiles[i][j]
			if t.Sensor != nil {
				t.Sensor.Update()
			}
		}
	}
}

func (w *World) display() {
	for i := 0; i < len(w.Tiles); i++ {
		for j := 0; j < len(w.Tiles[i]); j++ {
			t := w.Tiles[i][j]
			if t.Sensor != nil {
				t.Sensor.Update()
			}
		}
	}
}
