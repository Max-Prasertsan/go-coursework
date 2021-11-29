package main

import (
    "fmt"
    "os"
    //"runtime/trace"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
	//"uk.ac.bris.cs/gameoflife/util"
)
// TODO: go test -run=BenchmarkGol -v -bench .
func BenchmarkGol(t *testing.B){
    os.Stdout = nil

    for threads := 1; threads <= 16; threads ++{
        p := gol.Params{Turns: 100, Threads: threads, ImageWidth: 512, ImageHeight: 512}
        testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
        t.Run(testName, func(t *testing.B){
           for i := 0; i < p.Turns; i++ {
               events := make(chan gol.Event)
               go gol.Run(p, events, nil)
           }
        })
    }
}
