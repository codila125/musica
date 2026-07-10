package tui

import (
	"strings"
	"testing"

	"github.com/codila125/musica/internal/player"
)

func TestHelpOverlayFitsTerminalHeight(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("player.New: %v", err)
	}
	defer pl.Close()

	m := NewModel(fakeClient{}, pl, nil, 0)
	m.width = 100
	m.height = 30
	m.helpVisible = true

	out := m.View()
	if lines := strings.Count(out, "\n") + 1; lines > m.height {
		t.Fatalf("View() with help visible renders %d lines, want <= %d", lines, m.height)
	}
	if !strings.Contains(out, "KEYBOARD SHORTCUTS") {
		t.Fatalf("View() with help visible must show the help panel")
	}
}
