package tui

import (
	"testing"
	"time"

	"github.com/codila125/musica/internal/models"
)

func TestTickIntervalFor(t *testing.T) {
	cases := []struct {
		name  string
		state models.PlayerState
		want  time.Duration
	}{
		{"playing animates fast", models.StatePlaying, 80 * time.Millisecond},
		{"paused idles slow", models.StatePaused, 400 * time.Millisecond},
		{"stopped idles slow", models.StateStopped, 400 * time.Millisecond},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tickIntervalFor(tt.state); got != tt.want {
				t.Fatalf("tickIntervalFor(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestShouldPollProgress(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name     string
		state    models.PlayerState
		lastPoll time.Time
		want     bool
	}{
		{"playing and stale poll", models.StatePlaying, now.Add(-2 * time.Second), true},
		{"playing but fresh poll", models.StatePlaying, now.Add(-200 * time.Millisecond), false},
		{"paused and stale poll", models.StatePaused, now.Add(-2 * time.Second), true},
		{"stopped never polls", models.StateStopped, now.Add(-time.Hour), false},
		{"playing never polled yet", models.StatePlaying, time.Time{}, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldPollProgress(tt.state, tt.lastPoll, now); got != tt.want {
				t.Fatalf("shouldPollProgress(%v, %v) = %v, want %v", tt.state, tt.lastPoll, got, tt.want)
			}
		})
	}
}
