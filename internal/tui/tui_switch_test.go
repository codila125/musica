//go:build testmpv

package tui

import (
	"context"
	"io"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

type fakeClient struct{}

func (f fakeClient) Ping(ctx context.Context) error { return nil }
func (f fakeClient) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	return []models.Track{{ID: "1", Title: "t", Artist: "a", Album: "b", Duration: 60, StreamURL: "u"}}, nil
}
func (f fakeClient) GetArtists(ctx context.Context) ([]models.Artist, error) { return nil, nil }
func (f fakeClient) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	return nil, nil
}
func (f fakeClient) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	return nil, nil
}
func (f fakeClient) GetPlaylists(ctx context.Context) ([]models.Playlist, error) { return nil, nil }
func (f fakeClient) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	return nil, nil
}
func (f fakeClient) Search(ctx context.Context, query string) (models.SearchResult, error) {
	return models.SearchResult{}, nil
}
func (f fakeClient) StreamTrack(ctx context.Context, trackID string) (io.ReadCloser, error) {
	return nil, nil
}
func (f fakeClient) GetStreamURL(trackID string) string { return "" }
func (f fakeClient) GetCoverURL(albumID string) string  { return "" }

func TestSwitchServerStopsPlaybackAndAppliesClient(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	defer pl.Close()

	track := models.Track{ID: "x", Title: "song", StreamURL: "stream"}
	if err := pl.Play(track); err != nil {
		t.Fatalf("play: %v", err)
	}

	m := NewModel(fakeClient{}, pl, nil, 0)

	updated, _ := m.Update(switchServerMsg{client: fakeClient{}, index: 0, err: nil})
	model := updated.(Model)

	if model.player.State() != models.StateStopped {
		t.Fatalf("expected stopped state after switch, got %v", model.player.State())
	}
	if q := model.player.Queue(); len(q) != 0 {
		t.Fatalf("expected empty queue after switch, got %d", len(q))
	}
}

func TestSwitchGuardPreventsConcurrentSwitches(t *testing.T) {
	pl, err := player.New()
	if err != nil {
		t.Fatalf("new player: %v", err)
	}
	defer pl.Close()

	m := NewModel(fakeClient{}, pl, nil, 0)
	m.switching = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	model := updated.(Model)

	if !model.switching {
		t.Fatalf("expected switching guard to remain true")
	}
	if cmd != nil {
		t.Fatalf("expected no command when switching is already in progress")
	}
}
