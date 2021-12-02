package stubs

import "uk.ac.bris.cs/gameoflife/util"

var ComputeNextTurnHandler = "GolOperations.ComputeNextTurn"
var BrokerHandler = "BrokerOperations.Broker"

type Request struct {
	World [][]	uint8
	ImageWidth 	int
	ImageHeight int
	SliceStart 	int
	SliceEnd	int
	Threads 	int
	SliceNo 	int
}

type Response struct {
	WorldSlice 	 [][]uint8
	FlippedCells []util.Cell
}