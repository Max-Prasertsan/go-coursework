package main
import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

const ALIVE byte = 0xff
const DEAD byte = 0x00

type GolOperations struct {}

func modulo(x, m int) int {
	return (x%m + m) % m
}

func (g *GolOperations) ComputeNextTurn(req stubs.Request, res *stubs.Response) (err error) {

	fmt.Println("Computing a turn...")
	//create new 2D slice to store the result in
	newWorld := make([][]byte, req.SliceEnd-req.SliceStart)
	for i := range newWorld {
		newWorld[i] = make([]byte, req.ImageWidth)
	}

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
				} else {
					newWorld[x - req.SliceStart][y] = ALIVE
				}

			} else {
				if aliveNeighbours == 3 {
					newWorld[x - req.SliceStart][y] = ALIVE
				} else {
					newWorld[x - req.SliceStart][y] = DEAD
				}
			}
		}
	}

	res.WorldSlice = newWorld
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