package gol

import "time"

type reporterChannels struct {
	command	chan   reporterCommand
	idle    chan   bool
	world   chan   [][]uint8
	turns   chan   int
	events  chan<- Event
}

type reporterState struct {
	world [][]uint8
	turns int
	channels reporterChannels
}

type reporterCommand uint8

const (
	reporterUpdate reporterCommand = iota
	reporterCheckIdle
)

func (reporter *reporterState) countCells() {
	count := 0
	time.Sleep(time.Second * 2)
	for i := range reporter.world { //iterate through the array and count the alive cells
		for j := range reporter.world[i] {
			if reporter.world[i][j] == ALIVE {
				count++
			}
		}
	}
	reporter.channels.events <- AliveCellsCount{ //send the event through the events channel
		CompletedTurns: reporter.turns,
		CellsCount:     count,
	}
}

func startReporter(w [][]uint8, t int, r reporterChannels) {
	reporter := reporterState{
		world: w,
		turns: t,
		channels: r,
	}

	for {
		select {
		case command := <-reporter.channels.command:
			switch command {
			case reporterUpdate:
				w = <-reporter.channels.world
				t = <-reporter.channels.turns
			case reporterCheckIdle:
				reporter.channels.idle <- true
			}
		default:
			go reporter.countCells()
		}
	}
}