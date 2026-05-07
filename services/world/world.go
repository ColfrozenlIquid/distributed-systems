package main

import (
	"sync"
	"time"
)

type World struct {
	Width  int
	Height int
	Tiles  [][]Tile
	mu     sync.RWMutex
}

func (w *World) RunWorldLoop() {
	w.update()
	// fmt.Print("\033[H\033[2J")
	// fmt.Print(w.DrawMap())
	time.Sleep(2000 * time.Millisecond)
}

// No longer works with multiple sensors implementation
// func (w *World) DrawMap() string {
// 	var sb strings.Builder

// 	for y := 0; y < len(w.Tiles); y++ {
// 		for x := 0; x < len(w.Tiles[y]); x++ {
// 			tile := w.Tiles[y][x]

// 			var reading uint8 = 0
// 			if tile.Sensors != nil {
// 				reading = tile.Sensor.Color()
// 			}

// 			vis := RenderTile(tile.Terrain, tile.Infrastructure, reading)

// 			sb.WriteString(vis.BG)
// 			sb.WriteString(vis.FG)
// 			sb.WriteRune(vis.Char)
// 			sb.WriteString("\033[0m")
// 		}
// 		sb.WriteByte('\n')
// 	}
// 	return sb.String()
// }

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

func (w *World) GetTile(xpos int, ypos int) Tile {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Tiles[ypos][xpos]
}

func (w *World) update() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i := 0; i < len(w.Tiles); i++ {
		for j := 0; j < len(w.Tiles[i]); j++ {
			if w.Tiles[i][j].Sensors != nil {
				for _, sensor := range w.Tiles[i][j].Sensors {
					sensor.Update()
				}
			}
		}
	}
}
