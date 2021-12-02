package main
import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

const ALIVE byte = 0xff
const DEAD byte = 0x00

type GolOperations struct {}

func modulo(x, m int) int {
	return (x%m + m) % m
}

//computeNextTurn computes the next turn of the game of life on a slice of the game matrix
func computeNextTurn(req stubs.Request) *stubs.Response {
	response := new(stubs.Response)

	newWorld := make([][]byte, req.SliceEnd-req.SliceStart)
	for i := range newWorld {
		newWorld[i] = make([]byte, req.ImageWidth)
	}
	var flippedCells []util.Cell

	modifiers := []int{-1, 0, 1}

	for y := 0; y < req.ImageWidth; y++ { //for each cell in the slice of the old world
		for x := req.SliceStart; x < req.SliceEnd; x++ {
			var aliveNeighbours = 0
			for _, modx := range modifiers { //for each neighbour of the cell
				for _, mody := range modifiers {
					if !(modx == 0 && mody == 0) {
						var modifiedX = modulo(x+modx, req.ImageHeight)
						var modifiedY = modulo(y+mody, req.ImageWidth)
						var state = req.World[modifiedX][modifiedY]
						if state == ALIVE { //check if the cell is alive
							aliveNeighbours++ //and add it to the counter
						}
					}
				}
			}


			//decide the status of the cell in the new world based on the rules of the game of life
			if req.World[x][y] == ALIVE {
				if aliveNeighbours < 2 || aliveNeighbours > 3{
					newWorld[x - req.SliceStart][y] = DEAD
					flippedCells = append(flippedCells, util.Cell{X:y, Y:x}) //if a cell dies add it to the flippedCells list
				} else {
					newWorld[x - req.SliceStart][y] = ALIVE
				}

			} else {
				if aliveNeighbours == 3 {
					newWorld[x - req.SliceStart][y] = ALIVE
					flippedCells = append(flippedCells, util.Cell{X:y, Y:x}) //if a cell is born add it to the flippedCells list
				} else {
					newWorld[x - req.SliceStart][y] = DEAD
				}
			}
		}
	}

	//update the response
	response.WorldSlice = newWorld
	response.FlippedCells =flippedCells

	return response
}

func worker (req stubs.Request, out chan<- *stubs.Response) {
	out <- computeNextTurn(req)
}

//ComputeNextTurn distributes the handed slice into smaller slices to workers
func (g *GolOperations) ComputeNextTurn(req stubs.Request, res *stubs.Response) (err error) {

	fmt.Println("Computing a turn...")
	fmt.Println("World size: ", req.ImageHeight, "x", req.ImageWidth)
	fmt.Println("Slice start: ", req.SliceStart)
	fmt.Println("Slice end: ", req.SliceEnd)

	var newWorld [][]uint8
	var flippedCells []util.Cell
	var outChan [16]chan *stubs.Response //create an array of channels
	ipSliceSize := req.SliceEnd - req.SliceStart

	for i := 0 ; i <  req.Threads ; i ++ {
		outChan[i] = make(chan *stubs.Response)

		sliceStart := ipSliceSize * req.SliceNo + (ipSliceSize / req.Threads) * i //mark the start of the slice
		sliceEnd := ipSliceSize * req.SliceNo + (ipSliceSize / req.Threads) * (i + 1) //mark the end of the slice

		fmt.Println("Internal slice start:", sliceStart)
		fmt.Println("Internal Slice end: ", sliceEnd)

		if i == req.Threads - 1 { //if this the last thread
			sliceEnd = req.SliceEnd  //the end of the slice will be the same as the end of the handed over slice
		}
		request := stubs.Request{ //form request
			World:       req.World,
			ImageWidth:  req.ImageWidth,
			ImageHeight: req.ImageHeight,
			SliceStart:  sliceStart,
			SliceEnd:    sliceEnd,
		}

		go worker(request, outChan[i]) //hand over the request to the worker
	}

	for i := 0 ; i <  req.Threads ; i ++ {
		t := <-outChan[i]
		newWorld = append(newWorld, t.WorldSlice ...) //append the slices
		flippedCells = append(flippedCells, t.FlippedCells...) //append the flipped cells lists
	}

	//update the response
	res.WorldSlice = newWorld
	res.FlippedCells = flippedCells

	return
}

func main() {
	pAddr := flag.String("port","8030","Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&GolOperations{})
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