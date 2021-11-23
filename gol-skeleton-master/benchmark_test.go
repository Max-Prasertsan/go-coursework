package main

import (
    "fmt"
	"os"
	//"runtime/trace"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
	//"uk.ac.bris.cs/gameoflife/util"
)
// TODO: go test -run=Bench -bench benchmark_test
func benchMark(t *testing.T){
    os.Stdout = nil
    p := gol.Params{ImageWidth: 512, ImageHeight: 512, Turns: 1000}
    for threads := 1; threads <= 16; threads ++{
        p.Threads = threads
        testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
        t.Run(testName, func(t *testing.T){
	// Issue with t.N. Need TA help later	
            for i := 0; i < t.N; i++ {
                events := make(chan gol.Event)
                go gol.Run(p, events, nil)
            }
        })
    }
}
