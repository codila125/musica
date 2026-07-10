package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

func newPlayingTUIModel(t *testing.T) Model {
	t.Helper()
	pl, err := player.New()
	if err != nil {
		t.Fatalf("player.New: %v", err)
	}
	t.Cleanup(func() { pl.Close() })
	m := NewModel(fakeClient{}, pl, nil, 0)
	if err := m.playback.PlayTrack(models.Track{ID: "t1", StreamURL: "http://x/t1", Duration: 300}); err != nil {
		t.Fatalf("PlayTrack: %v", err)
	}
	return m
}

func pressKey(t *testing.T, m Model, r rune) Model {
	t.Helper()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
	updated, _ := m.Update(msg)
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", updated)
	}
	return next
}

func TestDotSeeksForward(t *testing.T) {
	m := newPlayingTUIModel(t)
	m = pressKey(t, m, '.')
	if pos, _ := m.playback.Position(); pos != 10 {
		t.Fatalf("position = %d, want 10", pos)
	}
}

func TestCommaSeeksBackward(t *testing.T) {
	m := newPlayingTUIModel(t)
	m = pressKey(t, m, '.')
	m = pressKey(t, m, '.')
	m = pressKey(t, m, ',')
	if pos, _ := m.playback.Position(); pos != 10 {
		t.Fatalf("position = %d, want 10", pos)
	}
}

func TestVolumeKeysAdjustAndShowStatus(t *testing.T) {
	m := newPlayingTUIModel(t)
	m = pressKey(t, m, '-')
	if v := m.playback.Volume(); v != 95 {
		t.Fatalf("volume = %d, want 95", v)
	}
	if !strings.Contains(m.status, "95") {
		t.Fatalf("status = %q, want volume feedback", m.status)
	}
	m = pressKey(t, m, '=')
	if v := m.playback.Volume(); v != 100 {
		t.Fatalf("volume = %d, want 100", v)
	}
}

func TestSeekVolumeKeysIgnoredInSearchInput(t *testing.T) {
	m := newPlayingTUIModel(t)
	m.activeTab = TabSearch // search starts in input mode
	m = pressKey(t, m, '.')
	if pos, _ := m.playback.Position(); pos != 0 {
		t.Fatalf("position = %d, want 0 (key must go to search input)", pos)
	}
	m = pressKey(t, m, '-')
	if v := m.playback.Volume(); v != 100 {
		t.Fatalf("volume = %d, want 100 (key must go to search input)", v)
	}
}
