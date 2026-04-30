//go:build testmpv

package views

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

type fakeAPIClient struct {
	recent []models.Track
	err    error
}

func (f fakeAPIClient) Ping(ctx context.Context) error { return nil }
func (f fakeAPIClient) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	return f.recent, f.err
}
func (f fakeAPIClient) GetArtists(ctx context.Context) ([]models.Artist, error) { return nil, nil }
func (f fakeAPIClient) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	return nil, nil
}
func (f fakeAPIClient) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	return nil, nil
}
func (f fakeAPIClient) GetPlaylists(ctx context.Context) ([]models.Playlist, error) { return nil, nil }
func (f fakeAPIClient) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	return nil, nil
}
func (f fakeAPIClient) Search(ctx context.Context, query string) (models.SearchResult, error) {
	return models.SearchResult{}, nil
}
func (f fakeAPIClient) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	return nil, nil
}
func (f fakeAPIClient) GetStreamURL(trackID string) string { return "" }
func (f fakeAPIClient) GetCoverURL(albumID string) string  { return "" }

type fakePlayerService struct {
	state   models.PlayerState
	queue   []models.Track
	current int
}

func (f *fakePlayerService) ToggleTrack(track models.Track) error {
	cur := f.CurrentTrack()
	if cur != nil && cur.ID == track.ID {
		if f.state == models.StatePlaying {
			f.state = models.StatePaused
			return nil
		}
		if f.state == models.StatePaused {
			f.state = models.StatePlaying
			return nil
		}
	}
	return f.Play(track)
}

func (f *fakePlayerService) ToggleQueueTrack(queue []models.Track, cursor int) error {
	if cursor < 0 || cursor >= len(queue) {
		return nil
	}
	cur := f.CurrentTrack()
	if cur != nil && cur.ID == queue[cursor].ID {
		if f.state == models.StatePlaying {
			f.state = models.StatePaused
			return nil
		}
		if f.state == models.StatePaused {
			f.state = models.StatePlaying
			return nil
		}
	}
	return f.PlayQueue(queue, cursor)
}

func (f *fakePlayerService) PlayTrack(track models.Track) error { return f.Play(track) }
func (f *fakePlayerService) QueueTrack(track models.Track) error {
	return f.AppendToQueue(track)
}
func (f *fakePlayerService) Replay() error {
	if len(f.queue) == 0 {
		return nil
	}
	if f.current < 0 || f.current >= len(f.queue) {
		return nil
	}
	track := f.queue[f.current]
	return f.Play(track)
}

func (f *fakePlayerService) Play(track models.Track) error {
	f.queue = []models.Track{track}
	f.current = 0
	f.state = models.StatePlaying
	return nil
}
func (f *fakePlayerService) PlayQueue(tracks []models.Track, startIdx int) error {
	f.queue = append([]models.Track(nil), tracks...)
	f.current = startIdx
	f.state = models.StatePlaying
	return nil
}
func (f *fakePlayerService) Pause() error  { f.state = models.StatePaused; return nil }
func (f *fakePlayerService) Resume() error { f.state = models.StatePlaying; return nil }
func (f *fakePlayerService) Stop() error {
	f.queue = nil
	f.current = 0
	f.state = models.StateStopped
	return nil
}
func (f *fakePlayerService) Next() error {
	if len(f.queue) == 0 {
		return nil
	}
	if f.current >= len(f.queue)-1 {
		f.queue = nil
		f.current = 0
		f.state = models.StateStopped
		return nil
	}
	f.current++
	f.state = models.StatePlaying
	return nil
}
func (f *fakePlayerService) Previous() error {
	if len(f.queue) == 0 {
		return nil
	}
	if f.current <= 0 {
		return nil
	}
	f.current--
	f.state = models.StatePlaying
	return nil
}
func (f *fakePlayerService) CurrentTrack() *models.Track {
	if f.current < 0 || f.current >= len(f.queue) {
		return nil
	}
	t := f.queue[f.current]
	return &t
}
func (f *fakePlayerService) State() models.PlayerState { return f.state }
func (f *fakePlayerService) Queue() []models.Track     { return append([]models.Track(nil), f.queue...) }
func (f *fakePlayerService) CurrentIndex() int         { return f.current }
func (f *fakePlayerService) Position() (int, error)    { return 0, nil }
func (f *fakePlayerService) Duration() (int, error)    { return 0, nil }
func (f *fakePlayerService) AppendToQueue(track models.Track) error {
	if len(f.queue) > 0 && f.current >= 0 && f.current < len(f.queue) {
		insertIdx := f.current + 1
		f.queue = append(f.queue, models.Track{})
		copy(f.queue[insertIdx+1:], f.queue[insertIdx:])
		f.queue[insertIdx] = track
	} else {
		f.queue = append(f.queue, track)
	}
	return nil
}

func TestBrowseModelIgnoresStaleMessages(t *testing.T) {
	pl := &fakePlayerService{}
	client := fakeAPIClient{recent: []models.Track{{ID: "1", Title: "Song", StreamURL: "url"}}}
	m := NewBrowseModelWithService(client, pl)

	updated, cmd := m.beginLoadRecentTracks()
	if cmd == nil {
		t.Fatalf("expected load command")
	}
	_ = updated

	stale := browseTracksMsg{id: m.loadReqID - 1, tracks: []models.Track{{ID: "stale"}}}
	m2, _ := m.Update(stale)
	if len(m2.tracks) != 0 {
		t.Fatalf("expected stale message to be ignored")
	}
}

func TestBrowseModelHandlesLoadError(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewBrowseModelWithService(fakeAPIClient{}, pl)
	err := errors.New("load failed")

	m2, _ := m.Update(browseTracksMsg{id: m.loadReqID, err: err})
	if m2.err == nil {
		t.Fatalf("expected error to be set")
	}
}

func TestBrowseQueueShortcut(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewBrowseModelWithService(fakeAPIClient{}, pl)
	m.tracks = []models.Track{{ID: "1", Title: "Song", StreamURL: "url"}}

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if len(pl.queue) != 1 {
		t.Fatalf("expected queue length 1, got %d", len(pl.queue))
	}
	if m2.err != nil {
		t.Fatalf("unexpected error: %v", m2.err)
	}
}

func TestBrowseQueueShortcutAddsAfterCurrent(t *testing.T) {
	pl := &fakePlayerService{}
	pl.queue = []models.Track{
		{ID: "1", Title: "Song 1", StreamURL: "url1"},
		{ID: "2", Title: "Song 2", StreamURL: "url2"},
	}
	pl.current = 0
	pl.state = models.StatePlaying

	m := NewBrowseModelWithService(fakeAPIClient{}, pl)
	m.tracks = []models.Track{{ID: "3", Title: "Song 3", StreamURL: "url3"}}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if len(pl.queue) != 3 {
		t.Fatalf("expected queue length 3, got %d", len(pl.queue))
	}
	if pl.queue[1].ID != "3" {
		t.Fatalf("expected queued track to be next, got %s", pl.queue[1].ID)
	}
}

func TestBrowsePlaySeedsQueueWhenEmpty(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewBrowseModelWithService(fakeAPIClient{}, pl)
	m.tracks = []models.Track{{ID: "1", Title: "Song", StreamURL: "url"}}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	if len(pl.queue) != 1 {
		t.Fatalf("expected queue length 1, got %d", len(pl.queue))
	}
	if pl.current != 0 {
		t.Fatalf("expected current index 0, got %d", pl.current)
	}
}

func TestBrowseNextSeedsQueueWhenEmpty(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewBrowseModelWithService(fakeAPIClient{}, pl)
	m.tracks = []models.Track{
		{ID: "1", Title: "Song 1", StreamURL: "url1"},
		{ID: "2", Title: "Song 2", StreamURL: "url2"},
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if len(pl.queue) != 2 {
		t.Fatalf("expected queue length 2, got %d", len(pl.queue))
	}
	if pl.current != 1 {
		t.Fatalf("expected current index 1, got %d", pl.current)
	}
}

func TestBrowsePaginationSlicesTracks(t *testing.T) {
	recent := make([]models.Track, 0, 120)
	for i := 0; i < 120; i++ {
		recent = append(recent, models.Track{ID: fmt.Sprintf("%d", i+1), Title: fmt.Sprintf("Song %d", i+1), StreamURL: "url"})
	}
	pl := &fakePlayerService{}
	m := NewBrowseModelWithService(fakeAPIClient{recent: recent}, pl)

	updated, cmd := m.beginLoadRecentTracks()
	if cmd == nil {
		t.Fatalf("expected load command")
	}
	m = updated

	m2, _ := m.Update(browseTracksMsg{id: m.loadReqID, tracks: recent})
	if len(m2.tracks) != 50 {
		t.Fatalf("expected 50 tracks on page 1, got %d", len(m2.tracks))
	}

	m2.page = 1
	m3, _ := m2.Update(browseTracksMsg{id: m2.loadReqID, tracks: recent})
	if len(m3.tracks) != 50 {
		t.Fatalf("expected 50 tracks on page 2, got %d", len(m3.tracks))
	}
	if m3.tracks[0].ID != "51" {
		t.Fatalf("expected first track id 51, got %s", m3.tracks[0].ID)
	}
}
