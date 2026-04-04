package player

import (
	"fmt"
	"sync"
	"time"

	mpv "github.com/gen2brain/go-mpv"

	"github.com/codila125/musica/internal/logger"
	"github.com/codila125/musica/internal/models"
)

type Player struct {
	mpv     *mpv.Mpv
	queue   []models.Track
	current int
	state   models.PlayerState
	volume  int
	mu      sync.Mutex
	done    chan struct{}
	ended   chan struct{}
}

func New() (*Player, error) {
	m := mpv.New()

	m.SetOptionString("vo", "null")
	m.SetOptionString("audio-display", "no")
	m.SetOptionString("terminal", "no")
	m.SetOptionString("quiet", "yes")

	if err := m.Initialize(); err != nil {
		return nil, fmt.Errorf("initialize mpv: %w", err)
	}

	m.RequestEvent(mpv.EventEnd, true)

	p := &Player{
		mpv:    m,
		state:  models.StateStopped,
		volume: 100,
		done:   make(chan struct{}),
		ended:  make(chan struct{}, 1),
	}

	go p.eventLoop()

	return p, nil
}

func (p *Player) eventLoop() {
	for {
		ev := p.mpv.WaitEvent(1)
		if ev == nil || ev.EventID == mpv.EventShutdown {
			return
		}
		if ev.EventID == mpv.EventEnd {
			endFile := ev.EndFile()
			if endFile.Error != nil {
				logger.Get().Error("Player end-file error: %v", endFile.Error)
			}
			if endFile.Reason == mpv.EndFileEOF {
				select {
				case p.ended <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (p *Player) Play(track models.Track) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if track.StreamURL == "" {
		return fmt.Errorf("empty stream URL for track: %s", track.Title)
	}
	logger.Get().Debug("Player loadfile: %s", track.StreamURL)

	p.queue = []models.Track{track}
	p.current = 0

	if err := p.mpv.Command([]string{"loadfile", track.StreamURL, "replace"}); err != nil {
		logger.Get().Error("Player loadfile failed: %v", err)
		return fmt.Errorf("loadfile: %w", err)
	}

	p.state = models.StatePlaying
	return nil
}

func (p *Player) PlayQueue(tracks []models.Track, startIdx int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if startIdx < 0 || startIdx >= len(tracks) {
		return fmt.Errorf("invalid start index: %d", startIdx)
	}
	if tracks[startIdx].StreamURL == "" {
		return fmt.Errorf("empty stream URL for track: %s", tracks[startIdx].Title)
	}
	logger.Get().Debug("Player queue loadfile: %s", tracks[startIdx].StreamURL)

	p.queue = tracks
	p.current = startIdx

	if err := p.mpv.Command([]string{"loadfile", tracks[startIdx].StreamURL, "replace"}); err != nil {
		logger.Get().Error("Player queue loadfile failed: %v", err)
		return fmt.Errorf("loadfile: %w", err)
	}

	p.state = models.StatePlaying
	return nil
}

func (p *Player) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != models.StatePlaying {
		return nil
	}

	p.mpv.SetPropertyString("pause", "yes")
	p.state = models.StatePaused
	return nil
}

func (p *Player) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != models.StatePaused {
		return nil
	}

	p.mpv.SetPropertyString("pause", "no")
	p.state = models.StatePlaying
	return nil
}

func (p *Player) TogglePause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.state {
	case models.StatePlaying:
		p.mpv.SetPropertyString("pause", "yes")
		p.state = models.StatePaused
	case models.StatePaused:
		p.mpv.SetPropertyString("pause", "no")
		p.state = models.StatePlaying
	}

	return nil
}

func (p *Player) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mpv.Command([]string{"stop"})
	p.state = models.StateStopped
	return nil
}

func (p *Player) Next() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return fmt.Errorf("queue is empty")
	}

	if p.current >= len(p.queue)-1 {
		return fmt.Errorf("end of queue")
	}

	p.current++

	if err := p.mpv.Command([]string{"loadfile", p.queue[p.current].StreamURL, "replace"}); err != nil {
		return fmt.Errorf("loadfile: %w", err)
	}

	p.state = models.StatePlaying
	return nil
}

func (p *Player) Previous() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return fmt.Errorf("queue is empty")
	}

	if p.current <= 0 {
		return fmt.Errorf("start of queue")
	}

	p.current--

	if err := p.mpv.Command([]string{"loadfile", p.queue[p.current].StreamURL, "replace"}); err != nil {
		return fmt.Errorf("loadfile: %w", err)
	}

	p.state = models.StatePlaying
	return nil
}

func (p *Player) SetVolume(vol int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}

	p.volume = vol
	p.mpv.SetPropertyString("volume", fmt.Sprintf("%d", vol))
}

func (p *Player) Seek(seconds int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.mpv.Command([]string{"seek", fmt.Sprintf("%d", seconds), "absolute"})
}

func (p *Player) Position() (int, error) {
	val := p.mpv.GetPropertyString("time-pos")
	if val == "" {
		return 0, nil
	}

	var pos float64
	if _, err := fmt.Sscanf(val, "%f", &pos); err != nil {
		return 0, err
	}

	return int(pos), nil
}

func (p *Player) Duration() (int, error) {
	val := p.mpv.GetPropertyString("duration")
	if val == "" {
		return 0, nil
	}

	var dur float64
	if _, err := fmt.Sscanf(val, "%f", &dur); err != nil {
		return 0, err
	}

	return int(dur), nil
}

func (p *Player) CurrentTrack() *models.Track {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.current < 0 || p.current >= len(p.queue) {
		return nil
	}

	return &p.queue[p.current]
}

func (p *Player) State() models.PlayerState {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.state
}

func (p *Player) Queue() []models.Track {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.queue
}

func (p *Player) CurrentIndex() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.current
}

func (p *Player) Ended() <-chan struct{} {
	return p.ended
}

func (p *Player) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mpv.TerminateDestroy()
	close(p.done)
	return nil
}

func (p *Player) Monitor(onTrackEnd func()) {
	for {
		select {
		case <-p.done:
			return
		case <-p.ended:
			p.mu.Lock()
			if p.current < len(p.queue)-1 {
				p.current++
				p.mpv.Command([]string{"loadfile", p.queue[p.current].StreamURL, "replace"})
				p.state = models.StatePlaying
			} else {
				p.state = models.StateStopped
			}
			p.mu.Unlock()

			if onTrackEnd != nil {
				onTrackEnd()
			}
		}
	}
}

func (p *Player) ProgressTicker(interval time.Duration, onTick func(pos, dur int)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			if p.State() == models.StatePlaying {
				pos, _ := p.Position()
				dur, _ := p.Duration()
				onTick(pos, dur)
			}
		}
	}
}
