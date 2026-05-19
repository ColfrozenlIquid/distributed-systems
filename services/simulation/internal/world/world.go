package world

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

func (w *World) RunWorldLoop() {
	w.update()
	time.Sleep(2000 * time.Millisecond)
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
