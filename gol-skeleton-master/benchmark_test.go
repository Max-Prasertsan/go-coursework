package main

import (
    "fmt"
	//"runtime/trace"
	"testing"
	"uk.ac.bris.cs/gameoflife/gol"
	//"uk.ac.bris.cs/gameoflife/util"
)
// TODO: go test -run=BenchmarkGol -v -bench .
func BenchmarkGol(t *testing.B){
    p := gol.Params{ImageWidth: 512, ImageHeight: 512, Turns: 100, Threads: 16}
    for threads := 1; threads <= 16; threads ++{
        p.Threads = threads
        testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
        t.Run(testName, func(t *testing.B){
           for i := 0; i < p.Turns; i++ {
               fmt.Println("after turns")
               fmt.Println(i)
               events := make(chan gol.Event)
               go gol.Run(p, events, nil)
           }
        })
    }
}

