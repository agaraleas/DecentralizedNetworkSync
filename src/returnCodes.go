package main

type ReturnCode int

const (
	NormalExit ReturnCode = iota
	InvalidDriverUrlError
	InvalidSharedDirectoryError
)
