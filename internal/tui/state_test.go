package tui

import "testing"

func TestCanTransition(t *testing.T) {
	valid := []struct {
		from appState
		to   appState
	}{
		{stateBooting, stateLoading},
		{stateLoading, stateReady},
		{stateReady, stateSwitchingServer},
		{stateSwitchingServer, stateLoading},
		{stateError, stateReady},
	}

	for _, tt := range valid {
		if !canTransition(tt.from, tt.to) {
			t.Fatalf("expected valid transition %s -> %s", tt.from.String(), tt.to.String())
		}
	}
}

func TestCanTransitionInvalid(t *testing.T) {
	invalid := []struct {
		from appState
		to   appState
	}{
		{stateBooting, stateReady},
		{stateSwitchingServer, stateBooting},
		{stateReady, stateBooting},
	}

	for _, tt := range invalid {
		if canTransition(tt.from, tt.to) {
			t.Fatalf("expected invalid transition %s -> %s", tt.from.String(), tt.to.String())
		}
	}
}
