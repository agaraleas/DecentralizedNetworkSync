package main

type ReturnCode int

const (
	NormalExit ReturnCode = iota
	CantListenToPortError
	InvalidHostError
	InvalidPortError
)