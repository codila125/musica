package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/models"
)

type libraryLevel int

const (
	levelArtists libraryLevel = iota
	levelAlbums
	levelTracks
)

type LibraryModel struct {
	client   api.Client
	playback PlaybackService

	level   libraryLevel
	artists []models.Artist
	albums  []models.Album
	tracks  []models.Track

	artistCursor int
	albumCursor  int
	trackCursor  int

	selArtist models.Artist
	selAlbum  models.Album

	width      int
	height     int
	loading    bool
	err        error
	loadReqID  int64
	cancelLoad context.CancelFunc
}

type libraryArtistsMsg struct {
	id      int64
	artists []models.Artist
	err     error
}

type libraryAlbumsMsg struct {
	id     int64
	albums []models.Album
	err    error
}

type libraryTracksMsg struct {
	id     int64
	tracks []models.Track
	err    error
}

func NewLibraryModel(client api.Client, pl PlaybackService) LibraryModel {
	return LibraryModel{
		client:    client,
		playback:  pl,
		loading:   true,
		loadReqID: nextRequestID(),
	}
}

func (m LibraryModel) Init() tea.Cmd {
	id := m.loadReqID
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		artists, err := client.GetArtists(ctx)
		return libraryArtistsMsg{id: id, artists: artists, err: err}
	}
}

func (m LibraryModel) Update(msg tea.Msg) (LibraryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case cancelInFlightMsg:
		if m.cancelLoad != nil {
			m.cancelLoad()
			m.cancelLoad = nil
		}
		m.loadReqID = nextRequestID()
		m.loading = false

	case libraryArtistsMsg:
		if msg.id != m.loadReqID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.artists = msg.artists
			m.clampCursors()
		}

	case libraryAlbumsMsg:
		if msg.id != m.loadReqID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.albums = msg.albums
			m.level = levelAlbums
			m.albumCursor = 0
		}

	case libraryTracksMsg:
		if msg.id != m.loadReqID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.tracks = msg.tracks
			m.level = levelTracks
			m.trackCursor = 0
		}
	}

	return m, nil
}

func (m LibraryModel) handleKey(msg tea.KeyMsg) (LibraryModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		switch m.level {
		case levelArtists:
			if m.artistCursor > 0 {
				m.artistCursor--
			}
		case levelAlbums:
			if m.albumCursor > 0 {
				m.albumCursor--
			}
		case levelTracks:
			if m.trackCursor > 0 {
				m.trackCursor--
			}
		}
	case "down", "j":
		switch m.level {
		case levelArtists:
			if m.artistCursor < len(m.artists)-1 {
				m.artistCursor++
			}
		case levelAlbums:
			if m.albumCursor < len(m.albums)-1 {
				m.albumCursor++
			}
		case levelTracks:
			if m.trackCursor < len(m.tracks)-1 {
				m.trackCursor++
			}
		}
	case "enter", "l", "right":
		return m.drillDown()
	case "esc", "h", "left":
		switch m.level {
		case levelTracks:
			m.level = levelAlbums
		case levelAlbums:
			m.level = levelArtists
		}
	case "p":
		if m.level == levelTracks && len(m.tracks) > 0 {
			m.err = m.playback.ToggleQueueTrack(m.tracks, m.trackCursor)
		} else {
			return m.drillDown()
		}
	case "q":
		if m.level == levelTracks && len(m.tracks) > 0 {
			m.err = m.playback.QueueTrack(m.tracks[m.trackCursor])
		}
	case "n":
		_ = m.playback.Next()
	case "m":
		_ = m.playback.Previous()
	case "r":
		_ = m.playback.Replay()
	case "ctrl+r":
		return m.reloadLevel()
	}
	return m, nil
}

func (m LibraryModel) drillDown() (LibraryModel, tea.Cmd) {
	switch m.level {
	case levelArtists:
		if m.artistCursor >= len(m.artists) {
			return m, nil
		}
		m.selArtist = m.artists[m.artistCursor]
		return m.beginLoad(func(ctx context.Context, client api.Client, id int64) tea.Msg {
			albums, err := client.GetAlbums(ctx, m.selArtist.ID)
			return libraryAlbumsMsg{id: id, albums: albums, err: err}
		})
	case levelAlbums:
		if m.albumCursor >= len(m.albums) {
			return m, nil
		}
		m.selAlbum = m.albums[m.albumCursor]
		return m.beginLoad(func(ctx context.Context, client api.Client, id int64) tea.Msg {
			tracks, err := client.GetTracks(ctx, m.selAlbum.ID)
			return libraryTracksMsg{id: id, tracks: tracks, err: err}
		})
	}
	return m, nil
}

func (m LibraryModel) reloadLevel() (LibraryModel, tea.Cmd) {
	switch m.level {
	case levelArtists:
		m.loading = true
		m.loadReqID = nextRequestID()
		return m, m.Init()
	case levelAlbums:
		m.level = levelArtists
		return m.drillDown()
	case levelTracks:
		m.level = levelAlbums
		return m.drillDown()
	}
	return m, nil
}

func (m LibraryModel) beginLoad(fetch func(ctx context.Context, client api.Client, id int64) tea.Msg) (LibraryModel, tea.Cmd) {
	if m.cancelLoad != nil {
		m.cancelLoad()
	}
	m.loadReqID = nextRequestID()
	m.loading = true
	m.err = nil

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	m.cancelLoad = cancel
	id := m.loadReqID
	client := m.client
	return m, func() tea.Msg {
		defer cancel()
		return fetch(ctx, client, id)
	}
}

func (m *LibraryModel) clampCursors() {
	if m.artistCursor >= len(m.artists) {
		m.artistCursor = len(m.artists) - 1
	}
	if m.artistCursor < 0 {
		m.artistCursor = 0
	}
}

func (m LibraryModel) View() string {
	w, h := normalizeViewSize(m.width, m.height)
	boxStyle := listBoxStyle(w, h)
	innerW := w - 8
	divider := listDivider(innerW)

	title := retroTitleStyle.Render("◎ MUSIC LIBRARY" + m.breadcrumb())

	if m.err != nil {
		return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			title, divider, "", retroErrorStyle.Render("ERROR: "+m.err.Error())))
	}
	if m.loading {
		return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			title, divider, "", retroLoadingStyle.Render("Loading...")))
	}

	visibleRows := calcVisibleRows(h, 8)
	var lines []string
	lines = append(lines, title, divider)

	switch m.level {
	case levelArtists:
		lines = append(lines, m.renderArtistRows(innerW, visibleRows)...)
	case levelAlbums:
		lines = append(lines, m.renderAlbumRows(innerW, visibleRows)...)
	case levelTracks:
		lines = append(lines, m.renderTrackRows(innerW, visibleRows)...)
	}

	lines = append(lines, divider)
	lines = append(lines, retroSubtleStyle.Render("  [enter/l] open  [esc/h] back  [p] play  [q] queue"))
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (m LibraryModel) breadcrumb() string {
	switch m.level {
	case levelAlbums:
		return "  »  " + truncateStr(m.selArtist.Name, 30)
	case levelTracks:
		return "  »  " + truncateStr(m.selArtist.Name, 20) + "  »  " + truncateStr(m.selAlbum.Name, 25)
	}
	return ""
}

func (m LibraryModel) renderArtistRows(innerW, visibleRows int) []string {
	if len(m.artists) == 0 {
		return []string{"", retroSubtleStyle.Render("  No artists found")}
	}
	start, end := scrollWindow(m.artistCursor, len(m.artists), visibleRows)
	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		a := m.artists[i]
		label := fmt.Sprintf("%s  %s", truncateStr(a.Name, innerW-24), retroSubtleStyle.Render(fmt.Sprintf("%d albums", a.AlbumCount)))
		out = append(out, m.renderRow(label, i == m.artistCursor))
	}
	return out
}

func (m LibraryModel) renderAlbumRows(innerW, visibleRows int) []string {
	if len(m.albums) == 0 {
		return []string{"", retroSubtleStyle.Render("  No albums found")}
	}
	start, end := scrollWindow(m.albumCursor, len(m.albums), visibleRows)
	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		a := m.albums[i]
		year := ""
		if a.Year > 0 {
			year = fmt.Sprintf(" (%d)", a.Year)
		}
		label := fmt.Sprintf("%s%s  %s", truncateStr(a.Name, innerW-28), year, retroSubtleStyle.Render(fmt.Sprintf("%d tracks", a.TrackCount)))
		out = append(out, m.renderRow(label, i == m.albumCursor))
	}
	return out
}

func (m LibraryModel) renderTrackRows(innerW, visibleRows int) []string {
	if len(m.tracks) == 0 {
		return []string{"", retroSubtleStyle.Render("  No tracks found")}
	}
	start, end := scrollWindow(m.trackCursor, len(m.tracks), visibleRows)
	current := -1
	if cur := m.playback.CurrentTrack(); cur != nil {
		for i, t := range m.tracks {
			if t.ID == cur.ID {
				current = i
				break
			}
		}
	}
	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		t := m.tracks[i]
		dur := FormatDuration(t.Duration)
		label := fmt.Sprintf("%02d  %s  %s", i+1, truncateStr(t.Title, innerW-20), retroSubtleStyle.Render(dur))
		if i == current {
			out = append(out, retroCurrentStyle.Render("▶ ")+retroCurrentStyle.Render(label))
			continue
		}
		out = append(out, m.renderRow(label, i == m.trackCursor))
	}
	return out
}

func (m LibraryModel) renderRow(label string, selected bool) string {
	if selected {
		return retroSelectedStyle.Render("► ") + retroSelectedStyle.Render(label)
	}
	return retroSubtleStyle.Render("  ") + lipgloss.NewStyle().Foreground(colorLightText).Render(label)
}
