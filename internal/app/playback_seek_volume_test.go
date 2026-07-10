package app

import (
	"testing"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

func newPlayingController(t *testing.T) *PlaybackController {
	t.Helper()
	pl, err := player.New()
	if err != nil {
		t.Fatalf("player.New: %v", err)
	}
	t.Cleanup(func() { pl.Close() })
	c := NewPlaybackController(pl)
	if err := c.PlayTrack(models.Track{ID: "t1", StreamURL: "http://x/t1", Duration: 300}); err != nil {
		t.Fatalf("PlayTrack: %v", err)
	}
	return c
}

func TestSeekByMovesForwardAndBack(t *testing.T) {
	c := newPlayingController(t)

	if err := c.SeekBy(10); err != nil {
		t.Fatalf("SeekBy(10): %v", err)
	}
	if pos, _ := c.Position(); pos != 10 {
		t.Fatalf("position = %d, want 10", pos)
	}
	if err := c.SeekBy(-4); err != nil {
		t.Fatalf("SeekBy(-4): %v", err)
	}
	if pos, _ := c.Position(); pos != 6 {
		t.Fatalf("position = %d, want 6", pos)
	}
}

func TestSeekByClampsAtZero(t *testing.T) {
	c := newPlayingController(t)

	if err := c.SeekBy(-100); err != nil {
		t.Fatalf("SeekBy(-100): %v", err)
	}
	if pos, _ := c.Position(); pos != 0 {
		t.Fatalf("position = %d, want 0", pos)
	}
}

func TestVolumeByAdjustsAndClamps(t *testing.T) {
	cases := []struct {
		name   string
		deltas []int
		want   int
	}{
		{"down from default", []int{-30}, 70},
		{"clamps at 100", []int{50}, 100},
		{"clamps at 0", []int{-80, -80}, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := newPlayingController(t)
			got := 0
			for _, d := range tt.deltas {
				got = c.VolumeBy(d)
			}
			if got != tt.want {
				t.Fatalf("VolumeBy(%v) = %d, want %d", tt.deltas, got, tt.want)
			}
		})
	}
}
