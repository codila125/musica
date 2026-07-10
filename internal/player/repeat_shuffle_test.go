package player

import (
	"sort"
	"testing"

	"github.com/codila125/musica/internal/models"
)

func TestCycleRepeat(t *testing.T) {
	p, _ := New()
	defer p.Close()

	want := []RepeatMode{RepeatAll, RepeatOne, RepeatOff}
	for _, w := range want {
		if got := p.CycleRepeat(); got != w {
			t.Fatalf("CycleRepeat() = %v, want %v", got, w)
		}
	}
}

func TestNextQueueIndex(t *testing.T) {
	cases := []struct {
		name    string
		current int
		qlen    int
		repeat  RepeatMode
		manual  bool
		want    int
		wantOK  bool
	}{
		{"mid queue advances", 0, 3, RepeatOff, false, 1, true},
		{"end stops", 2, 3, RepeatOff, false, 0, false},
		{"end wraps repeat all", 2, 3, RepeatAll, false, 0, true},
		{"repeat one replays on auto", 1, 3, RepeatOne, false, 1, true},
		{"repeat one advances on manual", 1, 3, RepeatOne, true, 2, true},
		{"repeat one manual at end wraps", 2, 3, RepeatOne, true, 0, true},
		{"empty queue", 0, 0, RepeatAll, false, 0, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := nextQueueIndex(tt.current, tt.qlen, tt.repeat, tt.manual)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("nextQueueIndex(%d, %d, %v, %v) = (%d, %v), want (%d, %v)",
					tt.current, tt.qlen, tt.repeat, tt.manual, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestNextWrapsWithRepeatAll(t *testing.T) {
	p, _ := New()
	defer p.Close()
	if err := p.PlayQueue(tracks("a", "b"), 1); err != nil {
		t.Fatal(err)
	}
	p.CycleRepeat() // -> all
	if err := p.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	if cur := p.CurrentTrack(); cur == nil || cur.ID != "a" {
		t.Fatalf("current = %v, want a", cur)
	}
	if p.State() != models.StatePlaying {
		t.Fatalf("state = %v, want playing", p.State())
	}
}

func TestShuffleKeepsCurrentTrackAndSet(t *testing.T) {
	p, _ := New()
	defer p.Close()
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	if err := p.PlayQueue(tracks(ids...), 2); err != nil {
		t.Fatal(err)
	}
	p.Shuffle()

	if cur := p.CurrentTrack(); cur == nil || cur.ID != "c" {
		t.Fatalf("current after shuffle = %v, want c", cur)
	}
	got := queueIDs(p)
	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	wantSorted := append([]string(nil), ids...)
	sort.Strings(wantSorted)
	if len(sorted) != len(wantSorted) {
		t.Fatalf("queue size changed: %v", got)
	}
	for i := range sorted {
		if sorted[i] != wantSorted[i] {
			t.Fatalf("queue set changed: %v", got)
		}
	}
}
