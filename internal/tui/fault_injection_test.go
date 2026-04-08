//go:build testmpv

package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/app"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/player"
)

func TestRapidSwitchSpamIgnoredWhileSwitching(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	defer pl.Close()

	servers := []config.ServerConfig{{Name: "A"}, {Name: "B"}}
	m := NewModel(fakeClient{}, pl, servers, 0)
	m.state = stateReady
	m.coordinator = fakeCoordinator{nextIndex: 1, nextOK: true, result: app.SwitchResult{Client: fakeClient{}, Index: 1}}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatalf("expected switch command on first keypress")
	}
	model := updated.(Model)

	for i := 0; i < 10; i++ {
		nextUpdated, nextCmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		if nextCmd != nil {
			t.Fatalf("expected no command while switching (iteration %d)", i)
		}
		model = nextUpdated.(Model)
	}
}

func TestSwitchAuthFailureTransition(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	defer pl.Close()

	servers := []config.ServerConfig{{Name: "A"}, {Name: "B"}}
	m := NewModel(fakeClient{}, pl, servers, 0)
	m.state = stateReady
	m.coordinator = fakeCoordinator{
		nextIndex: 1,
		nextOK:    true,
		result: app.SwitchResult{
			Err: api.Wrap(api.ErrorKindAuth, "jellyfin.authenticate", errors.New("bad credentials")),
		},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatalf("expected switch command")
	}
	model := updated.(Model)

	msg := cmd()
	updated2, _ := model.Update(msg)
	model2 := updated2.(Model)

	if model2.state != stateError {
		t.Fatalf("expected error state, got %s", model2.state.String())
	}
	if model2.status != "Server switch failed: authentication error" {
		t.Fatalf("unexpected status: %s", model2.status)
	}
}

func TestSwitchRecoveryAfterNetworkFailure(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	defer pl.Close()

	servers := []config.ServerConfig{{Name: "A"}, {Name: "B"}}
	m := NewModel(fakeClient{}, pl, servers, 0)
	m.state = stateReady

	// First attempt fails.
	m.coordinator = fakeCoordinator{
		nextIndex: 1,
		nextOK:    true,
		result: app.SwitchResult{
			Err: api.Wrap(api.ErrorKindNetwork, "jellyfin.ping", errors.New("timeout")),
		},
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatalf("expected switch command")
	}
	model := updated.(Model)
	updated2, _ := model.Update(cmd())
	model = updated2.(Model)
	if model.state != stateError {
		t.Fatalf("expected error state after failed switch")
	}

	// Second attempt succeeds.
	model.coordinator = fakeCoordinator{nextIndex: 1, nextOK: true, result: app.SwitchResult{Client: fakeClient{}, Index: 1}}
	updated3, cmd2 := model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd2 == nil {
		t.Fatalf("expected switch command on recovery attempt")
	}
	model = updated3.(Model)
	updated4, _ := model.Update(cmd2())
	model = updated4.(Model)

	if model.state != stateReady {
		t.Fatalf("expected ready state after recovery, got %s", model.state.String())
	}
}
