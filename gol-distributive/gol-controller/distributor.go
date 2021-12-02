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

// World initialise global World, CompletedTurns, and LastTurn variables
var World [][]uint8
var CompletedTurns int
var LastTurn = false


//reportAliveCellCount sends AliveCellsCount event in the events channel
func reportAliveCellCount(eventsChan chan<- Event, done chan bool) {
	count := 0
	for {
		if !LastTurn { //if it's not the last turn
			time.Sleep(time.Second * 2)
			for i := range World { //iterate through the array and count the alive cells
				for j := range World[i] {
					if World[i][j] == ALIVE {
						count ++
					}
				}
			}
			eventsChan <- AliveCellsCount{ //send the event through the events channel
				CompletedTurns: CompletedTurns,
				CellsCount:     count,
			}
			count = 0

		} else { //otherwise
			done <- true //mark the routine as done
			break //exit
		}
	}
}

func output(c distributorChannels, filename string) {
	c.ioCommand <- ioOutput //tell io to write to image
	c.ioFilename <- filename + "x" + strconv.Itoa(CompletedTurns)

	for _, i := range World { //hand over
		for _, j := range i {
			c.ioOutput <- j
		}
	}

	c.events <- ImageOutputComplete{
		CompletedTurns: CompletedTurns,
		Filename:       filename,
	}
}

func finish(c distributorChannels,cellCountDone <-chan bool ,filename string) {
	output(c, filename)

	//Report the final state using FinalTurnCompleteEvent.
	var aliveCells []util.Cell
	for i := range World {
		for j := range World[i] {
			if World[i][j] == ALIVE {
				aliveCells = append(aliveCells, util.Cell{X: j, Y: i})
			}
		}
	}

	c.events <- FinalTurnComplete{
		CompletedTurns: CompletedTurns,
		Alive:          aliveCells,
	}
	LastTurn = true

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	<-cellCountDone //wait for the reportAliveCellCount routine to finish
	c.events <- StateChange{CompletedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

	client, _ := rpc.Dial("tcp", "127.0.0.1:8030")
	defer client.Close()

	//Create a 2D slice to store the World.
	imageHeight := p.ImageHeight
	imageWidth := p.ImageWidth

	var filename = strconv.Itoa(imageHeight) + "x" + strconv.Itoa(imageWidth)

	World = make([][]uint8, imageHeight) //initialize empty 2D matrix
	for i := range World {
		World[i] = make([]uint8, imageWidth)
	}

	c.ioCommand <- ioInput //tell io to read from image
	c.ioFilename <- filename

	for i := 0; i < imageHeight; i++ { //fill the grid with the corresponding values
		for j := 0; j < imageWidth; j++ {
			World[i][j] = <-c.ioInput
			if World[i][j] == ALIVE {
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
	CompletedTurns = 0

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

		request := stubs.Request{
			World: World,
			ImageWidth: imageWidth,
			ImageHeight: imageHeight,
			SliceStart: 0,
			SliceEnd: imageHeight,
			Threads: p.Threads,
		}

		response := new(stubs.Response)
		client.Call(stubs.BrokerHandler, request, response)

		t := response.WorldSlice
		World = t

		for i := range response.FlippedCells {
			c.events <- CellFlipped{
				Cell: response.FlippedCells[i],
				CompletedTurns: CompletedTurns,
			}
		}

		CompletedTurns++
		c.events <- TurnComplete{CompletedTurns: CompletedTurns}
	}

	finish(c, cellCountDone, filename)

}
