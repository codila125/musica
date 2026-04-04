//go:build testmpv

package tui

import (
	"context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

type adapterFakeClient struct{}

func (f adapterFakeClient) Ping(ctx context.Context) error { return nil }
func (f adapterFakeClient) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	return []models.Track{{ID: "1", Title: "Song", StreamURL: "url"}}, nil
}
func (f adapterFakeClient) GetArtists(ctx context.Context) ([]models.Artist, error) { return nil, nil }
func (f adapterFakeClient) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	return nil, nil
}
func (f adapterFakeClient) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	return nil, nil
}
func (f adapterFakeClient) GetPlaylists(ctx context.Context) ([]models.Playlist, error) {
	return nil, nil
}
func (f adapterFakeClient) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	return nil, nil
}
func (f adapterFakeClient) Search(ctx context.Context, query string) (models.SearchResult, error) {
	return models.SearchResult{}, nil
}
func (f adapterFakeClient) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	return nil, nil
}
func (f adapterFakeClient) GetStreamURL(trackID string) string { return "" }
func (f adapterFakeClient) GetCoverURL(albumID string) string  { return "" }

type adapterFakePlayback struct{}

func (a adapterFakePlayback) ToggleTrack(track models.Track) error { return nil }
func (a adapterFakePlayback) ToggleQueueTrack(queue []models.Track, cursor int) error {
	return nil
}
func (a adapterFakePlayback) PlayTrack(track models.Track) error  { return nil }
func (a adapterFakePlayback) QueueTrack(track models.Track) error { return nil }
func (a adapterFakePlayback) Stop() error                         { return nil }
func (a adapterFakePlayback) CurrentTrack() *models.Track         { return nil }
func (a adapterFakePlayback) State() models.PlayerState           { return models.StateStopped }
func (a adapterFakePlayback) Queue() []models.Track               { return nil }
func (a adapterFakePlayback) CurrentIndex() int                   { return 0 }

func TestViewAdapterLifecycle(t *testing.T) {
	v := newViewAdapter(adapterFakeClient{}, adapterFakePlayback{})

	if cmd := v.Init(); cmd == nil {
		t.Fatalf("expected init cmd")
	}

	v.Resize(tea.WindowSizeMsg{Width: 100, Height: 30})
	v.CancelInFlight()

	_ = v.UpdateAll(tea.WindowSizeMsg{Width: 90, Height: 24})

	_ = v.UpdateActive(TabBrowse, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if got := v.View(TabBrowse); got == "" {
		t.Fatalf("expected non-empty browse view")
	}
}
