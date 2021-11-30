package gol

type aliveCellReporterChannels struct {
	lastTurn	chan bool
	done 		chan bool
}

