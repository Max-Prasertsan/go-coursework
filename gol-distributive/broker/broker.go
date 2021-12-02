package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type BrokerOperations struct {}

//makeCall hands over the request to the corresponding worker and outputs the response
func makeCall(client *rpc.Client, request stubs.Request) *stubs.Response {
	response := new(stubs.Response)
	client.Call(stubs.ComputeNextTurnHandler, request, response)
	return response
}

//worker distributes the request to makeCall and outputs the result in the corresponding channel
func worker(client *rpc.Client, request stubs.Request, out chan<- *stubs.Response) {
	out <- makeCall(client, request)
}

func (b *BrokerOperations) Broker(req stubs.Request, res *stubs.Response) (err error) {
	file, _ := os.Open("ipList")
	var ipAddrs []string //make list that contains all the ip addresses of the workers
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := scanner.Text()
		ipAddrs = append(ipAddrs, t)
	}

	var newWorld [][]uint8
	var flippedCells []util.Cell
	var outChan [16]chan *stubs.Response //create an array of channels

	for i := range ipAddrs { //for each worker
		fmt.Println("Distributing slice to ", ipAddrs[i])

		client, _ := rpc.Dial("tcp", ipAddrs[i]) //dial the worker
		defer client.Close()

		outChan[i] = make(chan *stubs.Response)  //initialize the i-th output channel

		sliceStart := (req.ImageHeight / len(ipAddrs)) * i //mark the start of the slice
		sliceEnd := (req.ImageHeight / len(ipAddrs)) * (i + 1) //mark the end of the slice
		if i == len(ipAddrs) - 1 { //if this the last thread
			sliceEnd += req.ImageHeight % len(ipAddrs) //the slice will include the last few lines left over
		}
		request := stubs.Request{ //form the request
			World:       req.World,
			ImageWidth:  req.ImageWidth,
			ImageHeight: req.ImageHeight,
			SliceStart:  sliceStart,
			SliceEnd:    sliceEnd,
			Threads:	 req.Threads,
			SliceNo: i,
		}

		go worker(client, request, outChan[i]) //hand over the request to worker
	}

	for i := range ipAddrs {
		t := <-outChan[i]
		newWorld = append(newWorld, t.WorldSlice ...) //append the slices together
		flippedCells = append(flippedCells, t.FlippedCells...) //append the lists of flipped cells together
	}

	//update the response
	res.WorldSlice = newWorld
	res.FlippedCells = flippedCells

	return
}

func main () {
	pAddr := flag.String("port","8030","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&BrokerOperations{})
	if err != nil {

		return
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)
	rpc.Accept(listener)
}
