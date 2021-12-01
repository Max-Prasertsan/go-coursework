package gol
//
//import (
//	"fmt"
//	"time"
//)
//
//type reporterChannels struct {
//	command	chan   reporterCommand
//	idle    chan   bool
//	events  chan<- Event
//}
//
//type reporterState struct {
//	channels reporterChannels
//}
//
//type reporterCommand uint8
//
//const (
//	reporterCheckIdle reporterCommand = iota
//)
//
//func (reporter *reporterState) countCells() {
//	count := 0
//	time.Sleep(time.Second * 2)
//	for i := range World { //iterate through the array and count the alive cells
//		for j := range World[i] {
//			if World[i][j] == ALIVE {
//				count++
//			}
//		}
//	}
//	reporter.channels.events <- AliveCellsCount{ //send the event through the events channel
//		CompletedTurns: CompletedTurns,
//		CellsCount:     count,
//	}
//}
//
//func startReporter(w [][]uint8, t int, r reporterChannels) {
//	reporter := reporterState{
//		channels: r,
//	}
//
//	for {
//		//fmt.Println("got here - reporter")
//		go reporter.countCells()
//		select {
//		case command := <-reporter.channels.command:
//			switch command {
//			case reporterCheckIdle:
//				reporter.channels.idle <- true
//			default:
//				fmt.Println("got here - reporter")
//				//reporter.countCells()
//			}
//		}
//	}
//}