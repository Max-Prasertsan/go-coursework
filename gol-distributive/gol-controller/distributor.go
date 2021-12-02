package gol_controller

import (
	"fmt"
	"net/rpc"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
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

// initialise global world, completedTurns, and lastTurn variables
var world [][]uint8
var completedTurns int
var lastTurn = false


//reportAliveCellCount sends AliveCellsCount event in the events channel
func reportAliveCellCount(eventsChan chan<- Event, done chan bool) {
	count := 0
	for {
		if !lastTurn { //if it's not the last turn
			time.Sleep(time.Second * 2)
			for i := range world { //iterate through the array and count the alive cells
				for j := range world[i] {
					if world[i][j] == ALIVE {
						count ++
					}
				}
			}
			eventsChan <- AliveCellsCount{ //send the event through the events channel
				CompletedTurns: completedTurns,
				CellsCount:     count,
			}
			count = 0

		} else { //otherwise
			done <- true //mark the routine as done
			break //exit
		}
	}
}

//output the current world state to image
func output(c distributorChannels, filename string) {
	c.ioCommand <- ioOutput //tell io to write to image
	c.ioFilename <- filename + "x" + strconv.Itoa(completedTurns)

	for _, i := range world { //hand over
		for _, j := range i {
			c.ioOutput <- j
		}
	}

	c.events <- ImageOutputComplete{
		CompletedTurns: completedTurns,
		Filename:       filename,
	}
}

//finish closes the program
func finish(c distributorChannels,cellCountDone <-chan bool ,filename string) {
	output(c, filename)

	//Report the final state using FinalTurnCompleteEvent.
	var aliveCells []util.Cell
	for i := range world {
		for j := range world[i] {
			if world[i][j] == ALIVE {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: completedTurns,
		Alive:          aliveCells,
	}
	lastTurn = true

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	<-cellCountDone //wait for the reportAliveCellCount routine to finish
	c.events <- StateChange{completedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

	//dial localhost:8030 for broker
	client, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer client.Close()

	//Create a 2D slice to store the world.
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	var filename = strconv.Itoa(imageHeight) + "x" + strconv.Itoa(imageWidth)

	world = make([][]uint8, imageHeight) //initialize empty 2D matrix
	for i := range world {
		world[i] = make([]uint8, imageWidth)
	}

	c.ioCommand <- ioInput //tell io to read from image
	c.ioFilename <- filename

	for i := 0; i < imageHeight; i++ { //fill the grid with the corresponding values
		for j := 0; j < imageWidth; j++ {
			world[i][j] = <-c.ioInput
			if world[i][j] == ALIVE {
				c.events <- CellFlipped{
					CompletedTurns: 0,
					Cell: util.Cell{X: j, Y: i},
				}
			}
		}
	}

	cellCountDone := make(chan bool)
	go reportAliveCellCount(c.events, cellCountDone)

	//Execute all turns of the Game of Life.
	completedTurns = 0

	for i := 0; i < p.Turns; i++ { //for each turn of the game

		select {
		case keyPress := <-keyPresses:
			switch keyPress {
			case 's':
				output(c, filename)
			case 'q':
				finish(c, cellCountDone, filename)
			case 'p':
			pLoop:
				for {
					select {
					case keyPress := <-keyPresses:
						switch keyPress {
						case 's':
							output(c, filename)
						case 'q':
							finish(c, cellCountDone, filename)
						case 'p':
							fmt.Println("Continuing")
							break pLoop
						}
					}
				}

			}
		default:
		}

		request := stubs.Request{ //form request
			World:       world,
			ImageWidth:  imageWidth,
			ImageHeight: imageHeight,
			SliceStart:  0,
			SliceEnd:    imageHeight,
			Threads:     p.Threads,
		}

		response := new(stubs.Response)
		client.Call(stubs.BrokerHandler, request, response) //hand request to broker

		t := response.WorldSlice
		world = t //update world state

		for i := range response.FlippedCells { //for every cell that has been flipped
			c.events <- CellFlipped{ //send CellFlipped event in the event channel
				Cell:           response.FlippedCells[i],
				CompletedTurns: completedTurns,
			}
		}

		completedTurns++
		c.events <- TurnComplete{CompletedTurns: completedTurns}
	}

	finish(c, cellCountDone, filename)

}
