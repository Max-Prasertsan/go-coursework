package gol

import (
	"fmt"
	"net/rpc"
	"strconv"
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
const DEAD byte = 0x00

//initialise global world, completedTurns, and lastTurn variables
var world [][]uint8
var completedTurns int
var lastTurn = false


//reportAliveCellCount sends AliveCellsCount event in the events channel
//func reportAliveCellCount(eventsChan chan<- Event, done chan bool) {
//	count := 0
//	for {
//		if !lastTurn { //if it's not the last turn
//			time.Sleep(time.Second * 2)
//			for i := range world { //iterate through the array and count the alive cells
//				for j := range world[i] {
//					if world[i][j] == ALIVE {
//						count ++
//					}
//				}
//			}
//			eventsChan <- AliveCellsCount{ //send the event through the events channel
//				CompletedTurns: completedTurns,
//				CellsCount:     count,
//			}
//			count = 0
//
//		} else { //otherwise
//			done <- true //mark the routine as done
//			break //exit
//		}
//	}
//}

func makeCall(client rpc.Client) [][]uint8 {
	request := stubs.Request{
		World: world,
	}
	response := new(stubs.Response)
	client.Call(stubs.ComputeNextTurnHandler, request, response)
	return response.WorldSlice
}

////worker distributes the slices to computeNextTurn and outputs the result in the corresponding channel
//func worker(eventsChan chan<- Event, imageWidth, imageHeight, sliceStart, sliceEnd int, out chan<- [][]uint8) {
//	out <- computeNextTurn(eventsChan, imageWidth, imageHeight, sliceStart, sliceEnd)
//}

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
		Filename: filename,
	}
}

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

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	lastTurn = true
	<-cellCountDone //wait for the reportAliveCellCount routine to finish
	c.events <- StateChange{completedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

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
	//go reportAliveCellCount(c.events, cellCountDone)

	//Execute all turns of the Game of Life.
	completedTurns = 0
	var outChan [16]chan [][]uint8 //create an array of channels
	//TODO: find a way to allocate channels dynamically
	//'var outChan []chan [][]uint8' 'var outChan [p.Threads]chan [][]uint8' don't work (???)
	response := new(stubs.Response)

	for i := 0; i < p.Turns; i++ { //for each turn of the game

		select {
		case keyPress := <-keyPresses:
			switch keyPress {
			case 's':
				output(c, filename)
			case 'q':
				finish(c, cellCountDone, filename)
			case 'p':
				pLoop: for {
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
		
		var newWorld [][]uint8           //create new empty 2D matrix
		for i := 0; i < p.Threads; i++ { //for each thread
			outChan[i] = make(chan [][]uint8)               //initialize the i-th output channel
			sliceStart := (imageHeight / p.Threads) * i     //mark the beginning of the 2D slice
			sliceEnd := (imageHeight / p.Threads) * (i + 1) //mark the end of the 2D slice
			if i == p.Threads - 1 { //if this the last thread
				sliceEnd += imageHeight % p.Threads //the slice will include the last few lines left over
			}
			request := stubs.Request{
				World: world,
				ImageWidth: imageWidth,
				ImageHeight: imageHeight,
				SliceStart: sliceStart,
				SliceEnd: sliceEnd,
			}
			client.Call(stubs.ComputeNextTurnHandler, request, response)
//			go worker(c.events, imageWidth, imageHeight, sliceStart, sliceEnd, outChan[i]) //hand over the slice to the worker

		}

		newWorld = response.WorldSlice

		world = newWorld
		completedTurns++
		c.events <- TurnComplete{CompletedTurns: completedTurns}
	}

	finish(c, cellCountDone, filename)

}
