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

type WorldState struct {
	World          [][]uint8
	CompletedTurns int
}

const ALIVE byte = 0xff
const DEAD byte = 0x00


//modulo computes the modulo of 2 integers so the result is an integer that neatly wraps in an array
func modulo(x, m int) int {
	return (x%m + m) % m
}

//computeNextTurn computes the next turn of the game of life on a slice of the game matrix
func computeNextTurn(eventsChan chan<- Event, s WorldState, imageWidth, imageHeight, sliceStart, sliceEnd int) [][]uint8 {

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
						var state = s.World[modifiedX][modifiedY]
						if state == ALIVE { //check if the cell is alive
							aliveNeighbours++ //and add it to the counter
						}
					}
				}
			}


			//decide the status of the cell in the new world based on the rules of the game of life
			if s.World[x][y] == ALIVE {
				if aliveNeighbours < 2 || aliveNeighbours > 3{
					newWorld[x - sliceStart][y] = DEAD
					eventsChan <- CellFlipped{
						CompletedTurns: s.CompletedTurns,
						Cell: util.Cell{X: y, Y: x},
					}
				} else {
					newWorld[x - sliceStart][y] = ALIVE
				}

			} else {
				if aliveNeighbours == 3 {
					newWorld[x - sliceStart][y] = ALIVE
					eventsChan <- CellFlipped{
						CompletedTurns: s.CompletedTurns,
						Cell: util.Cell{X: y, Y: x},
					}
				} else {
					newWorld[x - sliceStart][y] = DEAD
				}
			}
		}
	}

	return newWorld
}

//worker distributes the slices to computeNextTurn and outputs the result in the corresponding channel
func worker(eventsChan chan<- Event, s WorldState, imageWidth, imageHeight, sliceStart, sliceEnd int, out chan<- [][]uint8) {
	out <- computeNextTurn(eventsChan, s,imageWidth, imageHeight, sliceStart, sliceEnd)
}

func output(c distributorChannels, s WorldState, filename string) {
	c.ioCommand <- ioOutput //tell io to write to image
	c.ioFilename <- filename + "x" + strconv.Itoa(s.CompletedTurns)

	for _, i := range s.World { //hand over
		for _, j := range i {
			c.ioOutput <- j
		}
	}

	c.events <- ImageOutputComplete{
		CompletedTurns: s.CompletedTurns,
		Filename: filename,
	}
}

func finish(c distributorChannels, r reporterChannels, s WorldState, filename string) {
	output(c, s, filename)

	r.command <- reporterCheckIdle
	<-r.idle

	//Report the final state using FinalTurnCompleteEvent.
	var aliveCells []util.Cell
	for i := range s.World {
		for j := range s.World[i] {
			if s.World[i][j] == ALIVE {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}



	c.events <- FinalTurnComplete{
		CompletedTurns: s.CompletedTurns,
		Alive:          aliveCells,
	}

	// Make sure that the Io has finished any output before exiting.




	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{s.CompletedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

	//Create a 2D slice to store the world.
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	var filename = strconv.Itoa(imageHeight) + "x" + strconv.Itoa(imageWidth)

	state := WorldState{
		World:          make([][]uint8, imageHeight),
		CompletedTurns: 0,
	}
	for i := range state.World{
		state.World[i] = make([]uint8, imageWidth)
	}

	c.ioCommand <- ioInput //tell io to read from image
	c.ioFilename <- filename

	for i := 0; i < imageHeight; i++ { //fill the grid with the corresponding values
		for j := 0; j < imageWidth; j++ {
			state.World[i][j] = <-c.ioInput
			if state.World[i][j] == ALIVE {
				c.events <- CellFlipped{
					CompletedTurns: 0,
					Cell: util.Cell{X: j, Y: i},
				}
			}
		}
	}

	reporterCommand := make(chan reporterCommand, 5)
	reporterIdle := make(chan bool)
	reporterWorld := make(chan [][]uint8)
	reporterTurns := make(chan int)

	r := reporterChannels{
		command: reporterCommand,
		idle:    reporterIdle,
		world:   reporterWorld,
		turns:   reporterTurns,
		events:  c.events,
	}

	go startReporter(state.World, 0,r)

	//Execute all turns of the Game of Life.
	var outChan [16]chan [][]uint8 //create an array of channels
	//TODO: find a way to allocate channels dynamically
	//'var outChan []chan [][]uint8' 'var outChan [p.Threads]chan [][]uint8' don't work (???)

	for i := 0; i < p.Turns; i++ { //for each turn of the game

		select {
		case keyPress := <-keyPresses:
			switch keyPress {
			case 's':
				output(c, state, filename)
			case 'q':
				finish(c, r, state, filename)
			case 'p':
				pLoop: for {
					select {
					case keyPress := <-keyPresses:
						switch keyPress {
						case 's':
							output(c, state, filename)
						case 'q':
							finish(c, r, state, filename)
						case 'p':
							fmt.Println("Continuing")
							break pLoop
						}
					}
				}

			}
		default:
		}
		
		var newWorld [][]uint8           //create new empty 2D matrix
		for i := 0; i < p.Threads; i++ { //for each thread
			outChan[i] = make(chan [][]uint8)               //initialize the i-th output channel
			sliceStart := (imageHeight / p.Threads) * i     //mark the beginning of the 2D slice
			sliceEnd := (imageHeight / p.Threads) * (i + 1) //mark the end of the 2D slice
			if i == p.Threads - 1 { //if this the last thread
				sliceEnd += imageHeight % p.Threads //the slice will include the last few lines left over
			}
			go worker(c.events, state,imageWidth, imageHeight, sliceStart, sliceEnd, outChan[i]) //hand over the slice to the worker

		}
		for i := 0; i < p.Threads; i++ { //for each thread
			newWorld = append(newWorld, <-outChan[i]...) //append the slices together
		}
		state.World = newWorld
		state.CompletedTurns++

		r.command <- reporterUpdate
		r.world <- state.World
		r.turns <- state.CompletedTurns


		c.events <- TurnComplete{CompletedTurns: state.CompletedTurns}
	}

	finish(c, r, state, filename)

}