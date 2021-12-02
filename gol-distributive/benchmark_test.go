package main

import (
    "fmt"
    "testing"
    "uk.ac.bris.cs/gameoflife/gol_controller"
    //"uk.ac.bris.cs/gameoflife/util"
)
// TODO: go test -run=BenchmarkGol -v -bench .
func BenchmarkGol(t *testing.B){
    p := gol_controller.Params{ImageWidth: 512, ImageHeight: 512, Turns: 100, Threads: 16}
    for threads := 1; threads <= 16; threads ++{
        p.Threads = threads
        testName := fmt.Sprintf("%dx%dx%d-%d", p.ImageWidth, p.ImageHeight, p.Turns, p.Threads)
        t.Run(testName, func(t *testing.B){
            for i := 0; i < t.N; i++ {
                fmt.Println("after turns")
                fmt.Println(i)
                events := make(chan gol_controller.Event)
                go gol_controller.Run(p, events, nil)
            }
        })
    }
}