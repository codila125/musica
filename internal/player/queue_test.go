package player

import (
	"testing"

	"github.com/codila125/musica/internal/models"
)

func tracks(ids ...string) []models.Track {
	ts := make([]models.Track, len(ids))
	for i, id := range ids {
		ts[i] = models.Track{ID: id, StreamURL: "http://x/" + id}
	}
	return ts
}

func queueIDs(p *Player) []string {
	q := p.Queue()
	ids := make([]string, len(q))
	for i, t := range q {
		ids[i] = t.ID
	}
	return ids
}

func TestRemoveFromQueueAfterCurrent(t *testing.T) {
	p, _ := New()
	defer p.Close()
	if err := p.PlayQueue(tracks("a", "b", "c"), 0); err != nil {
		t.Fatal(err)
	}
	if err := p.RemoveFromQueue(2); err != nil {
		t.Fatalf("RemoveFromQueue(2): %v", err)
	}
	got := queueIDs(p)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("queue = %v, want [a b]", got)
	}
	if p.CurrentIndex() != 0 {
		t.Fatalf("current = %d, want 0", p.CurrentIndex())
	}
}

func TestRemoveFromQueueBeforeCurrentShiftsIndex(t *testing.T) {
	p, _ := New()
	defer p.Close()
	if err := p.PlayQueue(tracks("a", "b", "c"), 2); err != nil {
		t.Fatal(err)
	}
	if err := p.RemoveFromQueue(0); err != nil {
		t.Fatalf("RemoveFromQueue(0): %v", err)
	}
	if p.CurrentIndex() != 1 {
		t.Fatalf("current = %d, want 1", p.CurrentIndex())
	}
	if cur := p.CurrentTrack(); cur == nil || cur.ID != "c" {
		t.Fatalf("current track = %v, want c", cur)
	}
}

func TestRemoveFromQueueRejectsPlayingTrack(t *testing.T) {
	p, _ := New()
	defer p.Close()
	if err := p.PlayQueue(tracks("a", "b"), 0); err != nil {
		t.Fatal(err)
	}
	if err := p.RemoveFromQueue(0); err == nil {
		t.Fatal("RemoveFromQueue(current) must error while playing")
	}
}

func TestRemoveFromQueueRejectsBadIndex(t *testing.T) {
	p, _ := New()
	defer p.Close()
	if err := p.PlayQueue(tracks("a"), 0); err != nil {
		t.Fatal(err)
	}
	if err := p.RemoveFromQueue(5); err == nil {
		t.Fatal("RemoveFromQueue(5) must error")
	}
}

func TestClearQueueKeepsPlayingTrack(t *testing.T) {
	p, _ := New()
	defer p.Close()
	if err := p.PlayQueue(tracks("a", "b", "c"), 1); err != nil {
		t.Fatal(err)
	}
	p.ClearQueue()
	got := queueIDs(p)
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("queue = %v, want [b]", got)
	}
	if p.CurrentIndex() != 0 {
		t.Fatalf("current = %d, want 0", p.CurrentIndex())
	}
	if p.State() != models.StatePlaying {
		t.Fatalf("state = %v, want playing", p.State())
	}
}
