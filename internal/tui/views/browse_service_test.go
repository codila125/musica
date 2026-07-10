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
	count  int
}

func (f fakeAPIClient) Ping(ctx context.Context) error { return nil }
func (f fakeAPIClient) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.recent[:min(limit, len(f.recent))], nil
}
func (f fakeAPIClient) GetRecentTracksCount(ctx context.Context) (int, error) {
	return f.count, f.err
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
func (f fakeAPIClient) GetStreamURL(trackID string) string                 { return "" }
func (f fakeAPIClient) Scrobble(ctx context.Context, trackID string) error { return nil }
func (f fakeAPIClient) GetLyrics(ctx context.Context, track models.Track) (models.Lyrics, error) {
	return models.Lyrics{}, nil
}
func (f fakeAPIClient) GetCoverURL(albumID string) string { return "" }

type fakePlayerService struct {
	state    models.PlayerState
	queue    []models.Track
	current  int
	shuffled bool
	repeat   models.RepeatMode
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
func (f *fakePlayerService) RemoveQueueTrack(idx int) error {
	if idx < 0 || idx >= len(f.queue) {
		return nil
	}
	if idx == f.current && f.state != models.StateStopped {
		return nil
	}
	f.queue = append(f.queue[:idx], f.queue[idx+1:]...)
	if idx < f.current {
		f.current--
	}
	return nil
}
func (f *fakePlayerService) Shuffle() { f.shuffled = true }
func (f *fakePlayerService) CycleRepeat() models.RepeatMode {
	f.repeat = (f.repeat + 1) % 3
	return f.repeat
}
func (f *fakePlayerService) Repeat() models.RepeatMode { return f.repeat }
func (f *fakePlayerService) ClearQueue() {
	if f.state == models.StateStopped || f.current < 0 || f.current >= len(f.queue) {
		f.queue = nil
		f.current = 0
		return
	}
	f.queue = []models.Track{f.queue[f.current]}
	f.current = 0
}
func (f *fakePlayerService) Position() (int, error) { return 0, nil }
func (f *fakePlayerService) Duration() (int, error) { return 0, nil }
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
	m := NewBrowseModel(client, pl)

	updated, cmd := m.beginLoadRecentTracks(true)
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
	m := NewBrowseModel(fakeAPIClient{}, pl)
	err := errors.New("load failed")

	m2, _ := m.Update(browseTracksMsg{id: m.loadReqID, err: err})
	if m2.err == nil {
		t.Fatalf("expected error to be set")
	}
}

func TestBrowseQueueShortcut(t *testing.T) {
	pl := &fakePlayerService{}
	m := NewBrowseModel(fakeAPIClient{}, pl)
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

	m := NewBrowseModel(fakeAPIClient{}, pl)
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
	m := NewBrowseModel(fakeAPIClient{}, pl)
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
	m := NewBrowseModel(fakeAPIClient{}, pl)
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

type countingAPIClient struct {
	fakeAPIClient
	countCalls  *int
	recentCalls *int
}

func (f countingAPIClient) GetRecentTracksCount(ctx context.Context) (int, error) {
	*f.countCalls++
	return f.fakeAPIClient.count, f.fakeAPIClient.err
}

func (f countingAPIClient) GetRecentTracks(ctx context.Context, limit int) ([]models.Track, error) {
	if f.recentCalls != nil {
		*f.recentCalls++
	}
	return f.fakeAPIClient.GetRecentTracks(ctx, limit)
}

func TestBrowsePageTurnReusesKnownCount(t *testing.T) {
	recent := make([]models.Track, 0, 120)
	for i := 0; i < 120; i++ {
		recent = append(recent, models.Track{ID: fmt.Sprintf("%d", i+1), StreamURL: "url"})
	}
	calls := 0
	client := countingAPIClient{fakeAPIClient: fakeAPIClient{recent: recent, count: 120}, countCalls: &calls}
	pl := &fakePlayerService{}
	m := NewBrowseModel(client, pl)

	updated, cmd := m.beginLoadRecentTracks(true)
	msg := cmd().(browseTracksMsg)
	m, _ = updated.Update(msg)
	if calls != 1 {
		t.Fatalf("expected 1 count call after initial load, got %d", calls)
	}
	if !m.totalKnown {
		t.Fatalf("expected total known after initial load")
	}

	m.page = 1
	updated2, cmd2 := m.beginLoadRecentTracks(false)
	msg2 := cmd2().(browseTracksMsg)
	_, _ = updated2.Update(msg2)
	if calls != 1 {
		t.Fatalf("expected count not re-fetched on page turn, got %d calls", calls)
	}
}

func TestBrowsePageTurnWithinCacheSkipsNetwork(t *testing.T) {
	recent := make([]models.Track, 0, 120)
	for i := 0; i < 120; i++ {
		recent = append(recent, models.Track{ID: fmt.Sprintf("%d", i+1), StreamURL: "url"})
	}
	countCalls, recentCalls := 0, 0
	client := countingAPIClient{
		fakeAPIClient: fakeAPIClient{recent: recent, count: 120},
		countCalls:    &countCalls,
		recentCalls:   &recentCalls,
	}
	pl := &fakePlayerService{}
	m := NewBrowseModel(client, pl)

	// Initial load: page 0, needs 50, cache empty -> real fetch.
	updated, cmd := m.beginLoadRecentTracks(true)
	m, _ = updated.Update(cmd().(browseTracksMsg))
	if recentCalls != 1 {
		t.Fatalf("expected 1 GetRecentTracks call after initial load, got %d", recentCalls)
	}

	// Page 1 needs 100, cache only has 50 -> real fetch, cache grows to 100.
	m.page = 1
	updated, cmd = m.beginLoadRecentTracks(false)
	if cmd == nil {
		t.Fatalf("expected a fetch when page turn exceeds cached range")
	}
	m, _ = updated.Update(cmd().(browseTracksMsg))
	if recentCalls != 2 {
		t.Fatalf("expected 2 GetRecentTracks calls after growing to page 1, got %d", recentCalls)
	}

	// Back to page 0: needs 50, cache already has 100 -> no network call at all.
	m.page = 0
	updated, cmd = m.beginLoadRecentTracks(false)
	if cmd != nil {
		t.Fatalf("expected cached page turn to skip the network fetch entirely")
	}
	if recentCalls != 2 {
		t.Fatalf("expected no additional GetRecentTracks call, got %d", recentCalls)
	}
	if len(updated.tracks) != 50 {
		t.Fatalf("expected 50 tracks served from cache, got %d", len(updated.tracks))
	}
	if updated.tracks[0].ID != "1" {
		t.Fatalf("expected first track id 1, got %s", updated.tracks[0].ID)
	}
}

func TestBrowseForceRefetchesCount(t *testing.T) {
	recent := []models.Track{{ID: "1", StreamURL: "url"}}
	calls := 0
	client := countingAPIClient{fakeAPIClient: fakeAPIClient{recent: recent, count: 1}, countCalls: &calls}
	pl := &fakePlayerService{}
	m := NewBrowseModel(client, pl)

	updated, cmd := m.beginLoadRecentTracks(true)
	msg := cmd().(browseTracksMsg)
	m, _ = updated.Update(msg)

	updated2, cmd2 := m.beginLoadRecentTracks(true)
	msg2 := cmd2().(browseTracksMsg)
	_, _ = updated2.Update(msg2)
	if calls != 2 {
		t.Fatalf("expected forced refresh to refetch count, got %d calls", calls)
	}
}

func TestBrowsePaginationSlicesTracks(t *testing.T) {
	recent := make([]models.Track, 0, 120)
	for i := 0; i < 120; i++ {
		recent = append(recent, models.Track{ID: fmt.Sprintf("%d", i+1), Title: fmt.Sprintf("Song %d", i+1), StreamURL: "url"})
	}
	pl := &fakePlayerService{}
	m := NewBrowseModel(fakeAPIClient{recent: recent}, pl)

	updated, cmd := m.beginLoadRecentTracks(true)
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
