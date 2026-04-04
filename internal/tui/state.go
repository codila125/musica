package tui

type appState int

const (
	stateBooting appState = iota
	stateReady
	stateLoading
	stateSwitchingServer
	stateError
)
