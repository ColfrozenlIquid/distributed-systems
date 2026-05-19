package world

import (
	"context"
	"fmt"
	pb "simulation/gen"
	"simulation/internal/config"
	"simulation/internal/simulatables"
	"sync"
	"time"
)

type World interface {
	RunWorldLoop(ctx context.Context)
	TileReading(req *pb.TileReadingRequest) (*pb.TileReadingResponse, error)
	TileUpdate(req *pb.TileUpdateRequest) (*pb.TileUpdateResponse, error)
}

type world struct {
	Width            int
	Height           int
	Tiles            [][]Tile
	UpdateIntervalMS float32
	mu               sync.RWMutex
}

func (w *world) TileReading(req *pb.TileReadingRequest) (*pb.TileReadingResponse, error) {
	tile := w.GetTile(int(req.GetXPos()), int(req.GetYPos()))

	sensorType, err := simulatables.ToDomainSensorType(req.GetSensorType())
	if err != nil {
		return nil, fmt.Errorf("unknown sensor type: %w", err)
	}

	tileReading, ok := tile.Read(sensorType)
	if !ok {
		return nil, fmt.Errorf("tile does not have this sensor")
	}

	return &pb.TileReadingResponse{
		SensorType:    req.GetSensorType(),
		SensorReading: tileReading,
	}, nil
}

func (w *world) TileUpdate(req *pb.TileUpdateRequest) (*pb.TileUpdateResponse, error) {
	tile := w.GetTile(int(req.GetXPos()), int(req.GetYPos()))

	sensorType, err := simulatables.ToDomainSensorType(req.GetSensorType())
	if err != nil {
		return nil, fmt.Errorf("unknown sensor type %w", err)
	}

	ok := tile.Set(sensorType, req.Value)
	if !ok {
		return nil, fmt.Errorf("cannot set value of tile")
	}

	return &pb.TileUpdateResponse{
		Success: true,
	}, nil
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

func (w *world) GetTile(xpos int, ypos int) Tile {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Tiles[ypos][xpos]
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
