//go:build testmpv

package views

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

type libraryFakeClient struct {
	fakeAPIClient
}

func (f libraryFakeClient) GetArtists(ctx context.Context) ([]models.Artist, error) {
	return []models.Artist{
		{ID: "ar1", Name: "Alpha Artist", AlbumCount: 2},
		{ID: "ar2", Name: "Beta Artist", AlbumCount: 1},
	}, nil
}

func (f libraryFakeClient) GetAlbums(ctx context.Context, artistID string) ([]models.Album, error) {
	if artistID != "ar1" {
		return nil, nil
	}
	return []models.Album{
		{ID: "al1", Name: "First Album", Artist: "Alpha Artist", Year: 1999, TrackCount: 2},
	}, nil
}

func (f libraryFakeClient) GetTracks(ctx context.Context, albumID string) ([]models.Track, error) {
	if albumID != "al1" {
		return nil, nil
	}
	return []models.Track{
		{ID: "t1", Title: "Opening Song", Artist: "Alpha Artist", StreamURL: "http://x/t1"},
		{ID: "t2", Title: "Closing Song", Artist: "Alpha Artist", StreamURL: "http://x/t2"},
	}, nil
}

func drainLibrary(t *testing.T, m LibraryModel, cmd tea.Cmd) LibraryModel {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return m
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c != nil {
					m, _ = m.Update(c())
				}
			}
			return m
		}
		m, cmd = m.Update(msg)
	}
	return m
}

func newLoadedLibrary(t *testing.T) (LibraryModel, *fakePlayerService) {
	t.Helper()
	pl := &fakePlayerService{}
	m := NewLibraryModel(libraryFakeClient{}, pl)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = drainLibrary(t, m, m.Init())
	return m, pl
}

func TestLibraryShowsArtists(t *testing.T) {
	m, _ := newLoadedLibrary(t)
	out := m.View()
	if !strings.Contains(out, "Alpha Artist") || !strings.Contains(out, "Beta Artist") {
		t.Fatalf("artists missing:\n%s", out)
	}
}

func TestLibraryDrillsDownToTracksAndPlays(t *testing.T) {
	m, pl := newLoadedLibrary(t)

	var cmd tea.Cmd
	m, cmd = m.Update(keyMsg("l")) // into Alpha Artist's albums
	m = drainLibrary(t, m, cmd)
	if out := m.View(); !strings.Contains(out, "First Album") {
		t.Fatalf("albums missing:\n%s", out)
	}

	m, cmd = m.Update(keyMsg("l")) // into First Album's tracks
	m = drainLibrary(t, m, cmd)
	if out := m.View(); !strings.Contains(out, "Opening Song") {
		t.Fatalf("tracks missing:\n%s", out)
	}

	m, cmd = m.Update(keyMsg("j"))
	m = drainLibrary(t, m, cmd)
	m, cmd = m.Update(keyMsg("p"))
	_ = drainLibrary(t, m, cmd)
	if cur := pl.CurrentTrack(); cur == nil || cur.ID != "t2" {
		t.Fatalf("current = %v, want t2", cur)
	}
}

func TestLibraryEscGoesBackUp(t *testing.T) {
	m, _ := newLoadedLibrary(t)

	var cmd tea.Cmd
	m, cmd = m.Update(keyMsg("l"))
	m = drainLibrary(t, m, cmd)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if out := m.View(); !strings.Contains(out, "Beta Artist") {
		t.Fatalf("esc must return to artist list:\n%s", out)
	}
}

func TestLibraryQueuesTrack(t *testing.T) {
	m, pl := newLoadedLibrary(t)

	var cmd tea.Cmd
	m, cmd = m.Update(keyMsg("l"))
	m = drainLibrary(t, m, cmd)
	m, cmd = m.Update(keyMsg("l"))
	m = drainLibrary(t, m, cmd)
	m, cmd = m.Update(keyMsg("q"))
	_ = drainLibrary(t, m, cmd)

	q := pl.Queue()
	if len(q) != 1 || q[0].ID != "t1" {
		t.Fatalf("queue = %v, want [t1]", q)
	}
}
