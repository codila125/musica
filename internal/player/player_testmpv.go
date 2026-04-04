//go:build testmpv

package player

import (
	"errors"
	"sync"
	"time"

	"github.com/codila125/musica/internal/models"
)

type Player struct {
	queue   []models.Track
	current int
	state   models.PlayerState
	mu      sync.Mutex
	done    chan struct{}
	ended   chan struct{}
}

func New() (*Player, error) {
	return &Player{
		state: models.StateStopped,
		done:  make(chan struct{}),
		ended: make(chan struct{}, 1),
	}, nil
}

func (p *Player) Play(track models.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if track.StreamURL == "" {
		return errors.New("empty stream URL")
	}
	p.queue = []models.Track{track}
	p.current = 0
	p.state = models.StatePlaying
	return nil
}

func (p *Player) PlayQueue(tracks []models.Track, startIdx int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if startIdx < 0 || startIdx >= len(tracks) {
		return errors.New("invalid start index")
	}
	p.queue = append([]models.Track(nil), tracks...)
	p.current = startIdx
	p.state = models.StatePlaying
	return nil
}

func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == models.StatePlaying {
		p.state = models.StatePaused
	}
	return nil
}

func (p *Player) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == models.StatePaused {
		p.state = models.StatePlaying
	}
	return nil
}

func (p *Player) TogglePause() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == models.StatePlaying {
		p.state = models.StatePaused
	} else if p.state == models.StatePaused {
		p.state = models.StatePlaying
	}
	return nil
}

func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = nil
	p.current = 0
	p.state = models.StateStopped
	return nil
}

func (p *Player) Next() error            { return nil }
func (p *Player) Previous() error        { return nil }
func (p *Player) SetVolume(vol int)      {}
func (p *Player) Seek(seconds int) error { return nil }
func (p *Player) Position() (int, error) { return 0, nil }
func (p *Player) Duration() (int, error) { return 0, nil }

func (p *Player) CurrentTrack() *models.Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current < 0 || p.current >= len(p.queue) {
		return nil
	}
	t := p.queue[p.current]
	return &t
}

func (p *Player) State() models.PlayerState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Queue() []models.Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]models.Track(nil), p.queue...)
}

func (p *Player) QueueHistory() []models.Track {
	return p.Queue()
}

func (p *Player) AppendToQueue(track models.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if track.StreamURL == "" {
		return errors.New("empty stream URL")
	}
	p.queue = append(p.queue, track)
	if p.state == models.StateStopped {
		p.current = len(p.queue) - 1
		p.state = models.StatePlaying
	}
	return nil
}

func (p *Player) CurrentIndex() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}

func (p *Player) Ended() <-chan struct{} { return p.ended }

func (p *Player) Close() error {
	close(p.done)
	return nil
}

func (p *Player) Monitor(onTrackEnd func()) {}

func (p *Player) ProgressTicker(interval time.Duration, onTick func(pos, dur int)) {}
