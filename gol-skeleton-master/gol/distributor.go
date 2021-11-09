package gol

import (
	"strconv"

	"uk.ac.bris.cs/gameoflife/util"
)

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

func computeNextTurn(oldWorld [][]uint8, imageWidth, imageHeight, sliceStart, sliceEnd int) [][]uint8 {

	modifiers := []int{-1, 0, 1}
	newWorld := make([][]byte, sliceEnd-sliceStart)
	for i := range newWorld {
		newWorld[i] = make([]byte, imageWidth)
	}

	for y := 0; y < imageWidth; y++ {
		for x := sliceStart; x < sliceEnd; x++ {
			var aliveNeighbours = 0
			for _, modx := range modifiers {
				for _, mody := range modifiers {
					if !(modx == 0 && mody == 0) {
						var modifiedX = modulo(x+modx, imageHeight)
						var modifiedY = modulo(y+mody, imageWidth)
						var state = oldWorld[modifiedX][modifiedY]
						if state == ALIVE {
							aliveNeighbours++
						}
					}
				}
			}

			if oldWorld[x][y] == ALIVE {
				if aliveNeighbours < 2 {
					newWorld[x][y] = DEAD
				} else if aliveNeighbours > 3 {
					newWorld[x][y] = DEAD
				} else {
					newWorld[x][y] = ALIVE
				}

			} else {
				if aliveNeighbours == 3 {
					newWorld[x][y] = ALIVE
				} else {
					newWorld[x][y] = DEAD
				}
			}
		}
	}
	return newWorld
}

func worker(oldWorld [][]uint8, imageWidth, imageHeight, sliceStart, sliceEnd int, out chan<- [][]uint8) {
	out <- computeNextTurn(oldWorld, imageWidth, imageHeight, sliceStart, sliceEnd)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	var filename = strconv.Itoa(imageHeight) + "x" + strconv.Itoa(imageWidth)

	world := make([][]uint8, imageHeight)
	for i := range world {
		world[i] = make([]uint8, imageWidth)
	}

	c.ioCommand <- ioInput
	c.ioFilename <- filename

	for i := 0; i < imageHeight; i++ {
		for j := 0; j < imageWidth; j++ {
			world[i][j] = <-c.ioInput
		}
	}

	// TODO: Execute all turns of the Game of Life.

	var outChan []chan [][]uint8

	completedTurns := 0
	for i := 0; i < p.Turns; i++ {
		var newWorld [][]uint8
		for i := 0; i < p.Threads; i++ {
			outChan[i] = make(chan [][]uint8)
			sliceStart := (imageHeight / p.Threads) * i
			sliceEnd := (imageHeight / p.Threads) * (i + 1)
			go worker(world, imageWidth, imageHeight, sliceStart, sliceEnd, outChan[i])
		}
		for i := 0; i < p.Threads; i++ {
			newWorld = append(newWorld, <-outChan[i]...)
		}
		world = newWorld
		completedTurns++
	}

	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	for _, i := range world {
		for _, j := range i {
			c.ioOutput <- j
		}
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	var aliveCells []util.Cell
	for i := 0; i < imageHeight; i++ {
		for j := 0; j < imageWidth; j++ {
			if world[i][j] == ALIVE {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
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
