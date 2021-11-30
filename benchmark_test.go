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
func BenchmarkGol(b *testing.B){
    for threads := 1; threads <= 16; threads ++{
        os.Stdout = nil
        p := gol.Params{
            Turns: 100,
            Threads: threads,
            ImageWidth: 512,
            ImageHeight: 512,
        }

        testName := fmt.Sprintf("%dx%ddx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
        b.Run(testName, func(b *testing.B){
           for i := 0; i < b.N; i++ {
               events := make(chan gol.Event)
               go gol.Run(p, events, nil)
               for  range events{
               }
           }
        })
    }
