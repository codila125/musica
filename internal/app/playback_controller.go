package app

import (
	"fmt"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

type PlaybackController struct {
	player *player.Player
}

func NewPlaybackController(pl *player.Player) *PlaybackController {
	return &PlaybackController{player: pl}
}

func (c *PlaybackController) ToggleTrack(track models.Track) error {
	cur := c.player.CurrentTrack()
	state := c.player.State()

	if cur != nil && cur.ID == track.ID {
		switch state {
		case models.StatePlaying:
			return c.player.Pause()
		case models.StatePaused:
			return c.player.Resume()
		}
	}

	return c.player.Play(track)
}

func (c *PlaybackController) ToggleQueueTrack(queue []models.Track, cursor int) error {
	if cursor < 0 || cursor >= len(queue) {
		return fmt.Errorf("invalid queue index: %d", cursor)
	}

	selected := queue[cursor]
	cur := c.player.CurrentTrack()
	state := c.player.State()

	if cur != nil && cur.ID == selected.ID {
		switch state {
		case models.StatePlaying:
			return c.player.Pause()
		case models.StatePaused:
			return c.player.Resume()
		}
	}

	return c.player.PlayQueue(queue, cursor)
}

func (c *PlaybackController) PlayTrack(track models.Track) error {
	return c.player.Play(track)
}

func (c *PlaybackController) QueueTrack(track models.Track) error {
	return c.player.AppendToQueue(track)
}

func (c *PlaybackController) Stop() error {
	return c.player.Stop()
}

func (c *PlaybackController) Replay() error {
	queue := c.player.Queue()
	if len(queue) > 0 {
		idx := c.player.CurrentIndex()
		if idx >= 0 && idx < len(queue) {
			return c.player.PlayQueue(queue, idx)
		}
	}

	cur := c.player.CurrentTrack()
	if cur == nil {
		return nil
	}
	return c.player.Play(*cur)
}

func (c *PlaybackController) Next() error {
	return c.player.Next()
}

func (c *PlaybackController) CurrentTrack() *models.Track {
	return c.player.CurrentTrack()
}

func (c *PlaybackController) State() models.PlayerState {
	return c.player.State()
}

func (c *PlaybackController) Queue() []models.Track {
	return c.player.Queue()
}

func (c *PlaybackController) CurrentIndex() int {
	return c.player.CurrentIndex()
}

func (c *PlaybackController) Position() (int, error) {
	return c.player.Position()
}

func (c *PlaybackController) Duration() (int, error) {
	return c.player.Duration()
}
