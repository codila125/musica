//go:build testmpv

package app

import (
	"testing"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

func TestToggleTrackPlayPauseResume(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	defer pl.Close()

	c := NewPlaybackController(pl)
	track := models.Track{ID: "1", Title: "Song", StreamURL: "url"}

	if err := c.ToggleTrack(track); err != nil {
		t.Fatalf("toggle play: %v", err)
	}
	if c.State() != models.StatePlaying {
		t.Fatalf("expected playing")
	}

	if err := c.ToggleTrack(track); err != nil {
		t.Fatalf("toggle pause: %v", err)
	}
	if c.State() != models.StatePaused {
		t.Fatalf("expected paused")
	}

	if err := c.ToggleTrack(track); err != nil {
		t.Fatalf("toggle resume: %v", err)
	}
	if c.State() != models.StatePlaying {
		t.Fatalf("expected playing after resume")
	}
}
