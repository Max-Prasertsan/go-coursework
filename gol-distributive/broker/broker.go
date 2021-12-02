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

func makeCall(client *rpc.Client, request stubs.Request) *stubs.Response {
	response := new(stubs.Response)
	client.Call(stubs.ComputeNextTurnHandler, request, response)
	return response
}

//worker distributes the slices to computeNextTurn and outputs the result in the corresponding channel
func worker(client *rpc.Client, request stubs.Request, out chan<- *stubs.Response) {
	out <- makeCall(client, request)
}

func (b *BrokerOperations) Broker(req stubs.Request, res *stubs.Response) (err error) {
	file, _ := os.Open("ipList")
	var ipAddrs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := scanner.Text()
		ipAddrs = append(ipAddrs, t)
	}
	var newWorld [][]uint8
	var flippedCells []util.Cell
	var outChan [16]chan *stubs.Response//create an array of channels
	//TODO: find a way to allocate channels dynamically
	//'var outChan []chan [][]uint8' 'var outChan [p.Threads]chan [][]uint8' don't work (???)

	fmt.Println("got here")
	fmt.Println(ipAddrs)
	for i := range ipAddrs {
		fmt.Println("Distributing slice to ", ipAddrs[i])

		client, _ := rpc.Dial("tcp", ipAddrs[i])
		defer client.Close()

		outChan[i] = make(chan *stubs.Response)

		sliceStart := (req.ImageHeight / len(ipAddrs)) * i
		sliceEnd := (req.ImageHeight / len(ipAddrs)) * (i + 1)
		if i == len(ipAddrs) - 1 { //if this the last thread
			sliceEnd += req.ImageHeight % len(ipAddrs) //the slice will include the last few lines left over
		}
		request := stubs.Request{
			World:       req.World,
			ImageWidth:  req.ImageWidth,
			ImageHeight: req.ImageHeight,
			SliceStart:  sliceStart,
			SliceEnd:    sliceEnd,
			Threads:	 req.Threads,
			SliceNo: i,
		}

		go worker(client, request, outChan[i])
	}

	for i := range ipAddrs {
		t := <-outChan[i]
		newWorld = append(newWorld, t.WorldSlice ...)
		for i := range t.FlippedCells {
			flippedCells = append(flippedCells, t.FlippedCells[i])
		}
	}

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
