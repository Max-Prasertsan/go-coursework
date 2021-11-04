package gol

import "uk.ac.bris.cs/gameoflife/util"

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

const ALIVE byte = 0xff
const DEAD byte = 0x00

func modulo(x, m int) int {
	return (x%m + m) % m
}

func computeNextTurn(oldWorld [][]uint8, imageWidth, imageHeight int) [][]uint8 {
	modifiers := []int{-1, 0, 1}
	newWorld := make([][]byte, imageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, imageWidth)
	}

	for x := 0; x < imageHeight; x++ {
		for y := 0; y < imageWidth; y++ {

			var alive_neighbours int = 0
			for _, modx := range modifiers {
				for _, mody := range modifiers {
					if !(modx == 0 && mody == 0) {
						var modified_x = modulo(x+modx, imageHeight)
						var modified_y = modulo(y+mody, imageWidth)
						var state = oldWorld[modified_x][modified_y]
						if state == ALIVE {
							alive_neighbours++
						}
					}
				}
			}

			if oldWorld[x][y] == ALIVE {
				if alive_neighbours < 2 {
					newWorld[x][y] = DEAD
				} else if alive_neighbours > 3 {
					newWorld[x][y] = DEAD
				} else {
					newWorld[x][y] = ALIVE
				}

			} else {
				if alive_neighbours == 3 {
					newWorld[x][y] = ALIVE
				} else {
					newWorld[x][y] = DEAD
				}
			}
		}
	}

	return newWorld
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	world := make([][]uint8, imageHeight)
	for i := range world {
		world[i] = make([]uint8, imageWidth)
	}

	for i := 0; i < imageHeight; i++ {
		for j := 0; j < imageWidth; j++ {
			world[i][j] = <-c.ioInput
		}
	}

	// TODO: Execute all turns of the Game of Life.

	completedTurns := 0

	for i := 0; i < p.Turns; i++ {
		world = computeNextTurn(world, p.ImageWidth, p.ImageHeight)
		completedTurns++
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	var aliveCells []util.Cell
	for i := 0; i < imageHeight; i++ {
		for j := 0; j < imageWidth; j++ {
			if world[i][j] == ALIVE {
				aliveCells = append(aliveCells, util.Cell{X: i, Y: j})
			}
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: completedTurns,
		Alive:          aliveCells,
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{completedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
