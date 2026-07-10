package views

import (
	"testing"

	"github.com/codila125/musica/internal/models"
)

func TestQueueViewShuffleKey(t *testing.T) {
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a", "b", "c"), current: 0}
	m := NewQueueModel(pl)

	m, _ = m.Update(keyMsg("s"))

	if !pl.shuffled {
		t.Fatal("pressing s must shuffle the queue")
	}
	_ = m
}

func TestQueueViewRepeatKey(t *testing.T) {
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a"), current: 0}
	m := NewQueueModel(pl)

	m, _ = m.Update(keyMsg("t"))

	if pl.repeat != models.RepeatAll {
		t.Fatalf("repeat = %v, want ALL after one press", pl.repeat)
	}
	_ = m
}
