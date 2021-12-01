package stubs

import "uk.ac.bris.cs/gameoflife/util"

var ComputeNextTurnHandler = "GolOperations.ComputeNextTurn"

type Request struct {
	World [][]uint8
	ImageWidth int
	ImageHeight int
	SliceStart int
	SliceEnd int
}

type Response struct {
	WorldSlice [][]uint8
	FlippedCells []util.Cell
}