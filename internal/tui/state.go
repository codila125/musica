package tui

type appState int

const (
	stateReady appState = iota
	stateSwitchingServer
)
