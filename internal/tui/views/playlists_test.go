//go:build testmpv

package views

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

type playlistsFakeClient struct {
	fakeAPIClient
}

func (f playlistsFakeClient) GetPlaylists(ctx context.Context) ([]models.Playlist, error) {
	return []models.Playlist{
		{ID: "p1", Name: "Morning Mix", TrackCount: 2},
		{ID: "p2", Name: "Evening Chill", TrackCount: 5},
	}, nil
}

func (f playlistsFakeClient) GetPlaylistTracks(ctx context.Context, playlistID string) ([]models.Track, error) {
	if playlistID != "p1" {
		return nil, nil
	}
	return []models.Track{
		{ID: "t1", Title: "Sunrise Tune", StreamURL: "http://x/t1"},
		{ID: "t2", Title: "Coffee Song", StreamURL: "http://x/t2"},
	}, nil
}

func drainPlaylists(t *testing.T, m PlaylistsModel, cmd tea.Cmd) PlaylistsModel {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return m
		}
		m, cmd = m.Update(msg)
	}
	return m
}

func newLoadedPlaylists(t *testing.T) (PlaylistsModel, *fakePlayerService) {
	t.Helper()
	pl := &fakePlayerService{}
	m := NewPlaylistsModel(playlistsFakeClient{}, pl)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = drainPlaylists(t, m, m.Init())
	return m, pl
}

func TestPlaylistsShowsList(t *testing.T) {
	m, _ := newLoadedPlaylists(t)
	out := m.View()
	if !strings.Contains(out, "Morning Mix") || !strings.Contains(out, "Evening Chill") {
		t.Fatalf("playlists missing:\n%s", out)
	}
}

func TestPlaylistsOpensAndPlays(t *testing.T) {
	m, pl := newLoadedPlaylists(t)

	var cmd tea.Cmd
	m, cmd = m.Update(keyMsg("l"))
	m = drainPlaylists(t, m, cmd)
	if out := m.View(); !strings.Contains(out, "Sunrise Tune") {
		t.Fatalf("playlist tracks missing:\n%s", out)
	}

	m, cmd = m.Update(keyMsg("p"))
	m = drainPlaylists(t, m, cmd)
	if cur := pl.CurrentTrack(); cur == nil || cur.ID != "t1" {
		t.Fatalf("current = %v, want t1", cur)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if out := m.View(); !strings.Contains(out, "Evening Chill") {
		t.Fatalf("esc must return to playlist list:\n%s", out)
	}
}
