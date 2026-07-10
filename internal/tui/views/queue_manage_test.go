package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

func queueOf(ids ...string) []models.Track {
	ts := make([]models.Track, len(ids))
	for i, id := range ids {
		ts[i] = models.Track{ID: id, Title: id, StreamURL: "http://x/" + id}
	}
	return ts
}

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestQueueViewRemovesTrackUnderCursor(t *testing.T) {
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a", "b", "c"), current: 0}
	m := NewQueueModel(pl)

	m, _ = m.Update(keyMsg("j")) // cursor to b
	m, _ = m.Update(keyMsg("d"))

	q := pl.Queue()
	if len(q) != 2 || q[0].ID != "a" || q[1].ID != "c" {
		t.Fatalf("queue = %v, want [a c]", q)
	}
	_ = m
}

func TestQueueViewClearKeepsPlaying(t *testing.T) {
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a", "b", "c"), current: 1}
	m := NewQueueModel(pl)

	m, _ = m.Update(keyMsg("c"))

	q := pl.Queue()
	if len(q) != 1 || q[0].ID != "b" {
		t.Fatalf("queue = %v, want [b]", q)
	}
	_ = m
}
