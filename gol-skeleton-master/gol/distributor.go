package gol

import (
	"fmt"
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

//modulo computes the modulo of 2 integers so the result is an integer that neatly wraps in an array
func modulo(x, m int) int {
	return (x%m + m) % m
}

//computeNextTurn computes the next turn of the game of life on a slice of the game matrix
func computeNextTurn(oldWorld [][]uint8, imageWidth, imageHeight, sliceStart, sliceEnd int) [][]uint8 {

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
						var state = oldWorld[modifiedX][modifiedY]
						if state == ALIVE { //check if the cell is alive
							aliveNeighbours++ //and add it to the counter
						}
					}
				}
			}

			//decide the status of the cell in the new world based on the rules of the game of life
			//fmt.Println(sliceEnd)
			//fmt.Println("X: ", x, "Y: ", y)

			if oldWorld[x][y] == ALIVE {
				if aliveNeighbours < 2 {
					newWorld[x - sliceStart][y] = DEAD
				} else if aliveNeighbours > 3 {
					newWorld[x - sliceStart][y] = DEAD
				} else {
					newWorld[x - sliceStart][y] = ALIVE
				}

			} else {
				if aliveNeighbours == 3 {
					newWorld[x - sliceStart][y] = ALIVE
				} else {
					newWorld[x - sliceStart][y] = DEAD
				}
			}
		}
	}

	return newWorld
}

//worker distributes the slices to computeNextTurn and outputs the result in the corresponding channel
func worker(oldWorld [][]uint8, imageWidth, imageHeight, sliceStart, sliceEnd int, out chan<- [][]uint8) {
	out <- computeNextTurn(oldWorld, imageWidth, imageHeight, sliceStart, sliceEnd)
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	//Create a 2D slice to store the world.
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	var filename = strconv.Itoa(imageHeight) + "x" + strconv.Itoa(imageWidth)

	world := make([][]uint8, imageHeight) //initialize empty 2D matrix
	for i := range world {
		world[i] = make([]uint8, imageWidth)
	}

	c.ioCommand <- ioInput //tell io to read from image
	c.ioFilename <- filename

	for i := 0; i < imageHeight; i++ { //fill the grid with the corresponding values
		for j := 0; j < imageWidth; j++ {
			world[i][j] = <-c.ioInput
		}
	}

	//Execute all turns of the Game of Life.
	completedTurns := 0
	var outChan [32]chan [][]uint8 //create an array of channels
	//TODO: find a way to allocate channels dynamically
	//'var outChan []chan [][]uint8' 'var outChan [p.Threads]chan [][]uint8' don't work (???)

	for i := 0; i < p.Turns; i++ { //for each turn of the game
		var newWorld [][]uint8           //create new empty 2D matrix
		for i := 0; i < p.Threads; i++ { //for each thread
			outChan[i] = make(chan [][]uint8)               //initialize the i-th output channel
			sliceStart := (imageHeight / p.Threads) * i     //mark the beginning of the 2D slice
			sliceEnd := (imageHeight / p.Threads) * (i + 1) //mark the end of the 2D slice
			fmt.Println(i)
			go worker(world, imageWidth, imageHeight, sliceStart, sliceEnd, outChan[i]) //hand over the slice to the worker
		}
		for i := 0; i < p.Threads; i++ { //for each thread
			newWorld = append(newWorld, <-outChan[i]...) //append the slices together
		}
		world = newWorld
		completedTurns++
	}

	c.ioCommand <- ioOutput //tell io to write to image
	c.ioFilename <- filename

	for _, i := range world { //hand over
		for _, j := range i {
			c.ioOutput <- j
		}
	}

	//Report the final state using FinalTurnCompleteEvent.
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
