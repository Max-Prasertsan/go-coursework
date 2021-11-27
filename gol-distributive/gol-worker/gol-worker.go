package gol

import "uk.ac.bris.cs/gameoflife/util"

//modulo computes the modulo of 2 integers so the result is an integer that neatly wraps in an array
func modulo(x, m int) int {
	return (x%m + m) % m

	//return (x & m + m) % m
}

func ComputeNextTurn(eventsChan chan<- Event, imageWidth, imageHeight, sliceStart, sliceEnd int) [][]uint8 {

	//create new 2D slice to store the result in
	newWorld := make([][]byte, sliceEnd-sliceStart)
	for i := range newWorld {
		newWorld[i] = make([]byte, imageWidth)
	}

	modifiers := []int{-1, 0, 1}

	for y := 0; y < imageWidth; y++ { //for each cell in the slice of the old world
		for x := sliceStart; x < sliceEnd; x++ {
			var aliveNeighbours = 0
			for _, modx := range modifiers { //for each neighbour of the cell
				for _, mody := range modifiers {
					if !(modx == 0 && mody == 0) {
						var modifiedX = modulo(x+modx, imageHeight)
						var modifiedY = modulo(y+mody, imageWidth)
						var state = world[modifiedX][modifiedY]
						if state == ALIVE { //check if the cell is alive
							aliveNeighbours++ //and add it to the counter
						}
					}
				}
			}

			//decide the status of the cell in the new world based on the rules of the game of life
			if world[x][y] == ALIVE {
				if aliveNeighbours < 2 || aliveNeighbours > 3 {
					newWorld[x-sliceStart][y] = DEAD
					eventsChan <- CellFlipped{
						CompletedTurns: completedTurns,
						Cell:           util.Cell{X: y, Y: x},
					}
				} else {
					newWorld[x-sliceStart][y] = ALIVE
				}

			} else {
				if aliveNeighbours == 3 {
					newWorld[x-sliceStart][y] = ALIVE
					eventsChan <- CellFlipped{
						CompletedTurns: completedTurns,
						Cell:           util.Cell{X: y, Y: x},
					}
				} else {
					newWorld[x-sliceStart][y] = DEAD
				}
			}
		}
	}

	return newWorld
}
