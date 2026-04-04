package tui

type appState int

const (
	stateBooting appState = iota
	stateReady
	stateLoading
	stateSwitchingServer
	stateError
)

func (s appState) String() string {
	switch s {
	case stateBooting:
		return "booting"
	case stateReady:
		return "ready"
	case stateLoading:
		return "loading"
	case stateSwitchingServer:
		return "switching_server"
	case stateError:
		return "error"
	default:
		return "unknown"
	}
}

func canTransition(from, to appState) bool {
	if from == to {
		return true
	}

	switch from {
	case stateBooting:
		return to == stateLoading || to == stateError
	case stateLoading:
		return to == stateReady || to == stateError || to == stateSwitchingServer
	case stateReady:
		return to == stateLoading || to == stateSwitchingServer || to == stateError
	case stateSwitchingServer:
		return to == stateLoading || to == stateError || to == stateReady
	case stateError:
		return to == stateLoading || to == stateSwitchingServer || to == stateReady
	default:
		return false
	}
}
