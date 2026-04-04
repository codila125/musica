//go:build testmpv

package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

func TestSearchEscCancelsLoadingAndReturnsInput(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewSearchModelWithService(fakeAPIClient{}, pl)
	m.state = SearchResults
	m.loading = true
	called := false
	m.cancelSearch = func() { called = true }

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m2.state != SearchInput {
		t.Fatalf("expected SearchInput state")
	}
	if m2.loading {
		t.Fatalf("expected loading false after esc")
	}
	if !called {
		t.Fatalf("expected in-flight search cancellation")
	}
}

func TestQueueCursorClampsWhenQueueShrinks(t *testing.T) {
	pl := &fakePlayerService{}
	pl.queue = []models.Track{{ID: "1", Title: "a", StreamURL: "u"}}
	m := NewQueueModelWithService(pl)
	m.cursor = 5

	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = m2
	if m.cursor != 0 {
		t.Fatalf("expected cursor to clamp to 0, got %d", m.cursor)
	}
}
