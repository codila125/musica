package views

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/models"
)

type lyricsFakeClient struct {
	fakeAPIClient
	lyrics models.Lyrics
	calls  int
}

func (f *lyricsFakeClient) GetLyrics(ctx context.Context, track models.Track) (models.Lyrics, error) {
	f.calls++
	return f.lyrics, nil
}

func syncedLyrics() models.Lyrics {
	return models.Lyrics{
		Synced: true,
		Lines: []models.LyricLine{
			{StartMs: 0, Text: "hello first line"},
			{StartMs: 5000, Text: "hello second line"},
			{StartMs: 9000, Text: "hello third line"},
		},
	}
}

func drainCmd(m NowPlayingModel, cmd tea.Cmd) (NowPlayingModel, tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				if c == nil {
					continue
				}
				m, _ = m.Update(c())
			}
			break
		}
		m, cmd = m.Update(msg)
	}
	return m, nil
}

func TestNowPlayingShowsTrackAndLyrics(t *testing.T) {
	client := &lyricsFakeClient{lyrics: syncedLyrics()}
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a", "b"), current: 0}
	pl.queue[0].Title = "My Song"
	pl.queue[0].Artist = "My Artist"
	pl.queue[0].Album = "My Album"

	m := NewNowPlayingModel(client, pl)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	var cmd tea.Cmd
	m, cmd = m.Update(ProgressMsg{PositionMs: 0, DurationS: 300})
	m, _ = drainCmd(m, cmd)

	out := m.View()
	if !strings.Contains(out, "My Song") || !strings.Contains(out, "My Artist") {
		t.Fatalf("view missing track metadata:\n%s", out)
	}
	if !strings.Contains(out, "hello first line") {
		t.Fatalf("view missing lyrics:\n%s", out)
	}
}

func TestNowPlayingHighlightsCurrentLyric(t *testing.T) {
	client := &lyricsFakeClient{lyrics: syncedLyrics()}
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a"), current: 0}

	m := NewNowPlayingModel(client, pl)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	var cmd tea.Cmd
	m, cmd = m.Update(ProgressMsg{PositionMs: 6000, DurationS: 300})
	m, _ = drainCmd(m, cmd)

	out := m.View()
	if !strings.Contains(out, "▶ hello second line") {
		t.Fatalf("second line must carry the current marker:\n%s", out)
	}
	if strings.Contains(out, "▶ hello first line") {
		t.Fatalf("only one line may carry the marker:\n%s", out)
	}
}

func TestNowPlayingPlaybackKeys(t *testing.T) {
	client := &lyricsFakeClient{}
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a", "b"), current: 0}
	m := NewNowPlayingModel(client, pl)

	m, _ = m.Update(keyMsg("n"))
	if cur := pl.CurrentTrack(); cur == nil || cur.ID != "b" {
		t.Fatalf("after n current = %v, want b", cur)
	}
	m, _ = m.Update(keyMsg("m"))
	if cur := pl.CurrentTrack(); cur == nil || cur.ID != "a" {
		t.Fatalf("after m current = %v, want a", cur)
	}
	m, _ = m.Update(keyMsg("p"))
	if pl.State() != models.StatePaused {
		t.Fatalf("after p state = %v, want paused", pl.State())
	}
	_ = m
}

func TestNowPlayingRendersCoverForTrackWithoutAlbumID(t *testing.T) {
	// 2x2 red PNG served over HTTP; track has a CoverURL but no AlbumID
	// (modern Jellyfin flat responses).
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer server.Close()

	client := &lyricsFakeClient{}
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a"), current: 0}
	pl.queue[0].CoverURL = server.URL + "/Items/a/Images/Primary"

	m := NewNowPlayingModel(client, pl)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	var cmd tea.Cmd
	m, cmd = m.Update(ProgressMsg{PositionMs: 0, DurationS: 300})
	m, _ = drainCmd(m, cmd)

	if !strings.Contains(m.View(), "▀") {
		t.Fatalf("cover half-blocks missing from view")
	}
}

func TestNowPlayingFetchesLyricsOncePerTrack(t *testing.T) {
	client := &lyricsFakeClient{lyrics: syncedLyrics()}
	pl := &fakePlayerService{state: models.StatePlaying, queue: queueOf("a"), current: 0}

	m := NewNowPlayingModel(client, pl)
	var cmd tea.Cmd
	m, cmd = m.Update(ProgressMsg{PositionMs: 0, DurationS: 300})
	m, _ = drainCmd(m, cmd)
	m, cmd = m.Update(ProgressMsg{PositionMs: 1000, DurationS: 300})
	_, _ = drainCmd(m, cmd)

	if client.calls != 1 {
		t.Fatalf("GetLyrics calls = %d, want 1", client.calls)
	}
}
