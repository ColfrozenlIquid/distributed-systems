package world

import (
	"context"
	"fmt"
	pb "simulation/gen"
	"simulation/internal/simulatables"
)

type World interface {
	RunWorldLoop(ctx context.Context)
	TileReading(ctx context.Context, req *pb.TileReadingRequest) (*pb.TileReadingResponse, error)
	TileUpdate(ctx context.Context, req *pb.TileUpdateRequest) (*pb.TileUpdateResponse, error)
	WorldData(ctx context.Context, req *pb.WorldDataRequest) (*pb.WorldDataResponse, error)
	MoveTo(ctx context.Context, req *pb.MoveToRequest) (*pb.MoveToResponse, error)
}

func (w *world) MoveTo(_ context.Context, req *pb.MoveToRequest) (*pb.MoveToResponse, error) {
	tile, err := w.getTile(int(req.GetXPos()), int(req.GetYPos()))
	if err != nil {
		return nil, fmt.Errorf("get tile: %w", err)
	}

	movableType, err := mapToMovableType(req.GetMovableType())
	if err != nil {
		return nil, fmt.Errorf("map movableType: %w", err)
	}

	ok := tile.IsMovable(movableType)

	if !ok {
		return &pb.MoveToResponse{
			Allowed: false,
		}, nil
	}

	return &pb.MoveToResponse{
		Allowed: true,
	}, nil
}

func (w *world) TileReading(_ context.Context, req *pb.TileReadingRequest) (*pb.TileReadingResponse, error) {
	tile, err := w.getTile(int(req.GetXPos()), int(req.GetYPos()))
	if err != nil {
		return nil, fmt.Errorf("get tile: %w", err)
	}

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

func (w *world) TileUpdate(_ context.Context, req *pb.TileUpdateRequest) (*pb.TileUpdateResponse, error) {
	tile, err := w.getTile(int(req.GetXPos()), int(req.GetYPos()))
	if err != nil {
		return nil, fmt.Errorf("get tile: %w", err)
	}

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

func (w *world) WorldData(_ context.Context, _ *pb.WorldDataRequest) (*pb.WorldDataResponse, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	tiles := make([]*pb.TileState, 0, w.Width*w.Height)
	for _, row := range w.Tiles {
		for _, tile := range row {
			pbTile := &pb.TileState{
				Terrain:        simulatables.ToProtoTerrainType(tile.Terrain),
				Infrastructure: simulatables.ToProtoInfrastructureType(tile.Infrastructure),
			}
			tiles = append(tiles, pbTile)
		}
	}

	return &pb.WorldDataResponse{
		Width:  int32(w.Width),
		Height: int32(w.Height),
		Tiles:  tiles,
	}, nil
}
