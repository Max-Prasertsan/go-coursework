package gol

import (
	"bufio"
	"net/rpc"
	"os"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func makeCall(client *rpc.Client, request stubs.Request, eventChan chan<- Event) [][]uint8 {
	response := new(stubs.Response)
	client.Call(stubs.ComputeNextTurnHandler, request, response)
	for i := range response.FlippedCells {
		eventChan <- CellFlipped{
			Cell: response.FlippedCells[i],
			CompletedTurns: CompletedTurns,
		}
	}
	return response.WorldSlice
}

//worker distributes the slices to computeNextTurn and outputs the result in the corresponding channel
func worker(client *rpc.Client, request stubs.Request, eventChan chan<- Event, out chan<- [][]uint8) {
	out <- makeCall(client, request, eventChan)
}

func Broker(world [][]uint8, imageHeight, imageWidth int, eventChan chan <- Event) [][]uint8 {
	file, _ := os.Open("ipList")
	var ipAddrs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := scanner.Text()
		ipAddrs = append(ipAddrs, t)
	}
	var newWorld [][]uint8
	var outChan [16]chan [][]uint8//create an array of channels
	//TODO: find a way to allocate channels dynamically
	//'var outChan []chan [][]uint8' 'var outChan [p.Threads]chan [][]uint8' don't work (???)

	for i := range ipAddrs {
		client, _ := rpc.Dial("tcp", ipAddrs[i])
		defer client.Close()

		sliceStart := (imageHeight / len(ipAddrs)) * i
		sliceEnd := (imageHeight / len(ipAddrs)) * (i + 1)
		if i == len(ipAddrs) - 1 { //if this the last thread
			sliceEnd += imageHeight % len(ipAddrs) //the slice will include the last few lines left over
		}
		request := stubs.Request{
			World:       world,
			ImageWidth:  imageWidth,
			ImageHeight: imageHeight,
			SliceStart:  sliceStart,
			SliceEnd:    sliceEnd,
		}

		worker(client, request, eventChan, outChan[i])
	}

	for i:= range ipAddrs {
		newWorld = append(newWorld, <-outChan[i]...)
	}

	return newWorld
}
