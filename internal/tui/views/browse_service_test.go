//go:build testmpv

package views

import (
	"context"
	"errors"
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
func (f *fakePlayerService) AppendToQueue(track models.Track) error {
	f.queue = append(f.queue, track)
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
