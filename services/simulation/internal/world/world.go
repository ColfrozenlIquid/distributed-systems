package world

import (
	"context"
	"fmt"
	pb "simulation/gen"
	"simulation/internal/config"
	"sync"
	"time"
)

type world struct {
	Width            int
	Height           int
	Tiles            [][]Tile
	UpdateIntervalMS float32
	mu               sync.RWMutex
}

type MovableType int

const (
	Land MovableType = iota
	Water
	Air
	Amphibious
)

func mapToMovableType(movableType pb.MovableType) (MovableType, error) {
	switch movableType {
	case pb.MovableType_MOVABLE_TYPE_AIR:
		return Air, nil
	case pb.MovableType_MOVABLE_TYPE_LAND:
		return Land, nil
	case pb.MovableType_MOVABLE_TYPE_AMPHIBIOUS:
		return Amphibious, nil
	case pb.MovableType_MOVABLE_TYPE_WATER:
		return Water, nil
	default:
		return 0, fmt.Errorf("unspecified movable type: %d", movableType)
	}
}

func NewWorld(width int, height int, cfg config.Config) World {
	tiles, err := GenerateTiles(cfg)
	if err != nil {
		panic("world is nil")
	}
	return &world{
		Width:            width,
		Height:           height,
		Tiles:            tiles,
		UpdateIntervalMS: cfg.World.UpdateIntervalMS,
	}
}

func (w *world) RunWorldLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.UpdateIntervalMS * float32(time.Millisecond)))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.update()
		case <-ctx.Done():
			return
		}
	}
}

func (w *world) getTile(xpos int, ypos int) (Tile, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if xpos < 0 || xpos > w.Width-1 || ypos < 0 || ypos > w.Height-1 {
		return Tile{}, fmt.Errorf("requested tile (%d, %d) out of bounds", xpos, ypos)
	}
	return w.Tiles[ypos][xpos], nil
}

func (w *world) update() {
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
