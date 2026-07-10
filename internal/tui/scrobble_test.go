package tui

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

type scrobbleRecorder struct {
	fakeClient
	mu  sync.Mutex
	ids []string
}

func (r *scrobbleRecorder) Scrobble(ctx context.Context, trackID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = append(r.ids, trackID)
	return nil
}

func (r *scrobbleRecorder) scrobbled() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.ids...)
}

func waitForScrobbles(t *testing.T, r *scrobbleRecorder, want int) []string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ids := r.scrobbled(); len(ids) >= want {
			return ids
		}
		time.Sleep(5 * time.Millisecond)
	}
	return r.scrobbled()
}

func TestTickScrobblesNewTrackOnce(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatal(err)
	}
	defer pl.Close()

	rec := &scrobbleRecorder{}
	m := NewModel(rec, pl, nil, 0)
	if err := m.playback.PlayTrack(models.Track{ID: "t1", StreamURL: "http://x/t1"}); err != nil {
		t.Fatal(err)
	}

	tick := func() {
		updated, _ := m.Update(uiTickMsg(time.Now()))
		m = updated.(Model)
	}

	tick()
	ids := waitForScrobbles(t, rec, 1)
	if len(ids) != 1 || ids[0] != "t1" {
		t.Fatalf("scrobbled = %v, want [t1]", ids)
	}

	tick() // same track, no duplicate
	time.Sleep(50 * time.Millisecond)
	if ids := rec.scrobbled(); len(ids) != 1 {
		t.Fatalf("scrobbled = %v, want single entry", ids)
	}

	if err := m.playback.PlayTrack(models.Track{ID: "t2", StreamURL: "http://x/t2"}); err != nil {
		t.Fatal(err)
	}
	tick()
	ids = waitForScrobbles(t, rec, 2)
	if len(ids) != 2 || ids[1] != "t2" {
		t.Fatalf("scrobbled = %v, want [t1 t2]", ids)
	}
}
