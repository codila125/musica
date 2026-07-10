package player

import (
	"fmt"

	"github.com/codila125/musica/internal/models"
)

// Queue mutations are pure index math on fields shared by both the mpv
// and testmpv Player structs, so this file builds under either tag.

// RemoveFromQueue drops the track at idx. The currently loaded track
// cannot be removed while playback is active.
func (p *Player) RemoveFromQueue(idx int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if idx < 0 || idx >= len(p.queue) {
		return fmt.Errorf("invalid queue index: %d", idx)
	}
	if idx == p.current && p.state != models.StateStopped {
		return fmt.Errorf("cannot remove the playing track")
	}

	p.queue = append(p.queue[:idx], p.queue[idx+1:]...)
	if idx < p.current {
		p.current--
	} else if p.current >= len(p.queue) && p.current > 0 {
		p.current = len(p.queue) - 1
	}
	return nil
}

// ClearQueue removes every queued track except the one currently loaded;
// when stopped it empties the queue entirely.
func (p *Player) ClearQueue() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == models.StateStopped || p.current < 0 || p.current >= len(p.queue) {
		p.queue = nil
		p.current = 0
		return
	}
	p.queue = []models.Track{p.queue[p.current]}
	p.current = 0
}
