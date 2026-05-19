package model

import pb "control-center/gen"

type World struct {
	Width  int
	Height int
	Tiles  [][]Tile
}

type Tile struct {
	Terrain        pb.TerrainType
	Infrastructure pb.InfrastructureType
}

func NewWorldFromWorldDataResponse(response *pb.WorldDataResponse) World {
	width := int(response.Width)
	height := int(response.Height)

	tiles := make([][]Tile, height)
	for y := 0; y < height; y++ {
		tiles[y] = make([]Tile, width)
		for x := 0; x < width; x++ {
			tileState := response.Tiles[y*width+x]
			tiles[y][x] = Tile{
				Terrain:        tileState.Terrain,
				Infrastructure: tileState.Infrastructure,
			}
		}
	}
	return World{
		Width:  width,
		Height: height,
		Tiles:  tiles,
	}
}
