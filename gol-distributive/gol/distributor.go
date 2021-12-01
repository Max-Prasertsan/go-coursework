package gol

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
const DEAD byte = 0x00

//initialise global World, CompletedTurns, and LastTurn variables
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

func makeCall(client rpc.Client, request stubs.Request, eventChan chan<- Event) [][]uint8 {
	response := new(stubs.Response)
	client.Call(stubs.ComputeNextTurnHandler, request, response)
	//fmt.Println("Called server")
	for i := range response.FlippedCells {
		eventChan <- CellFlipped{
			Cell: response.FlippedCells[i],
			CompletedTurns: CompletedTurns,
		}
	}
	return response.WorldSlice
}

//worker distributes the slices to computeNextTurn and outputs the result in the corresponding channel
func worker(client rpc.Client, request stubs.Request, out chan<- [][]uint8, eventChan chan<- Event) {
	out <- makeCall(client, request, eventChan)
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
	//client, _ := rpc.Dial("tcp", "54.147.90.44:8030")
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
	var outChan [16]chan [][]uint8 //create an array of channels
	//TODO: find a way to allocate channels dynamically
	//'var outChan []chan [][]uint8' 'var outChan [p.Threads]chan [][]uint8' don't work (???)

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
				World:       World,
				ImageWidth:  imageWidth,
				ImageHeight: imageHeight,
				SliceStart:  sliceStart,
				SliceEnd:    sliceEnd,
			}
			go worker(*client, request, outChan[i], c.events) //hand over the slice to the worker
			//fmt.Println("made worker for thread ", i)
		}

		for i := 0; i < p.Threads; i++ {
			newWorld = append(newWorld, <-outChan[i]...)
		}

		World = newWorld
		CompletedTurns++
		c.events <- TurnComplete{CompletedTurns: CompletedTurns}
	}

	finish(c, cellCountDone, filename)

}
