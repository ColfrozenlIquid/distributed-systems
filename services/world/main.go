package main

import (
	"bufio"
	"fmt"
	"os"
)

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

func loadMap() {
	file, err := os.Open("map")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		
	}

	var level [][]Tile
}

func main() {
	fmt.Println("Hello world")
}

func update() {

}
