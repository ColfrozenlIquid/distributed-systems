package world

import (
	"context"
	"fmt"
	"math/rand"
	"shared"
	pb "simulation/gen"
	"simulation/internal/config"
	"simulation/internal/simulatables"
	"sync"
	"time"

	"go.uber.org/zap"
)

const MAX_PERSON_TILES = 2
const MAX_PERSONS_PER_TILE = 10

type world struct {
	Width            uint32
	Height           uint32
	Tiles            [][]Tile
	UpdateIntervalMS float32
	PersonTiles      int
	mu               sync.RWMutex
	logger           *zap.Logger
}

type MovableType int

const (
	Land MovableType = iota
	Water
	Air
	Amphibious
)

func (w *world) addPersonsSimulatableToTiles() {
	if w.PersonTiles == MAX_PERSON_TILES {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for w.PersonTiles < MAX_PERSON_TILES {
		x_pos := rand.Intn(int(w.Width))
		y_pos := rand.Intn(int(w.Height))
		count := rand.Intn(MAX_PERSONS_PER_TILE) + 1

		tile := &w.Tiles[y_pos][x_pos]
		if tile.Terrain == simulatables.Land && tile.Infrastructure == simulatables.NoInfrastructure {
			tile.Sensors[simulatables.PersonConfidence] = simulatables.NewPersonsSim(count)
			w.logger.Info("add person to tile", zap.Int("x", x_pos), zap.Int("y", y_pos), zap.Int("count", count))
			w.PersonTiles++
		}
	}
}

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

func NewWorld(width uint32, height uint32, cfg config.Config, logger *zap.Logger) World {
	tiles, err := GenerateTiles(cfg)
	if err != nil {
		panic("world is nil")
	}
	world := &world{
		Width:            width,
		Height:           height,
		Tiles:            tiles,
		UpdateIntervalMS: cfg.World.UpdateIntervalMS,
		PersonTiles:      0,
		logger:           logger,
	}

	world.addPersonsSimulatableToTiles()

	return world
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

func (w *world) getTile(position shared.Position) (Tile, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if position.X > uint32(w.Width-1) || position.Y > uint32(w.Height-1) {
		return Tile{}, fmt.Errorf("requested tile (%d, %d) out of bounds", position.X, position.Y)
	}
	return w.Tiles[position.Y][position.X], nil
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
