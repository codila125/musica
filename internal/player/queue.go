package player

import (
	"fmt"
	"math/rand"

	"github.com/codila125/musica/internal/models"
)

// RepeatMode is aliased from models so views and app can share the type
// without importing the player package.
type RepeatMode = models.RepeatMode

const (
	RepeatOff = models.RepeatOff
	RepeatAll = models.RepeatAll
	RepeatOne = models.RepeatOne
)

// CycleRepeat advances off -> all -> one -> off and returns the new mode.
func (p *Player) CycleRepeat() RepeatMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.repeat = (p.repeat + 1) % 3
	return p.repeat
}

func (p *Player) Repeat() RepeatMode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.repeat
}

// nextQueueIndex decides where playback goes after current. Repeat-one
// only pins the index on automatic advance (track end); a manual next
// still moves on, wrapping like repeat-all.
func nextQueueIndex(current, qlen int, repeat RepeatMode, manual bool) (int, bool) {
	if qlen == 0 {
		return 0, false
	}
	if repeat == RepeatOne && !manual {
		return current, true
	}
	if current+1 < qlen {
		return current + 1, true
	}
	if repeat == RepeatAll || repeat == RepeatOne {
		return 0, true
	}
	return 0, false
}

// Shuffle randomizes the upcoming tracks, leaving the played portion and
// the current track in place.
func (p *Player) Shuffle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	start := p.current + 1
	if start >= len(p.queue) {
		return
	}
	rest := p.queue[start:]
	rand.Shuffle(len(rest), func(i, j int) {
		rest[i], rest[j] = rest[j], rest[i]
	})
}

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
