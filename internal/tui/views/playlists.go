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

type PlaylistsModel struct {
	client   api.Client
	playback PlaybackService

	playlists []models.Playlist
	tracks    []models.Track
	open      bool // false: playlist list, true: tracks of selected playlist
	selected  models.Playlist

	listCursor  int
	trackCursor int

	width      int
	height     int
	loading    bool
	err        error
	loadReqID  int64
	cancelLoad context.CancelFunc
}

type playlistsMsg struct {
	id        int64
	playlists []models.Playlist
	err       error
}

type playlistTracksMsg struct {
	id     int64
	tracks []models.Track
	err    error
}

func NewPlaylistsModel(client api.Client, pl PlaybackService) PlaylistsModel {
	return PlaylistsModel{
		client:    client,
		playback:  pl,
		loading:   true,
		loadReqID: nextRequestID(),
	}
}

func (m PlaylistsModel) Init() tea.Cmd {
	id := m.loadReqID
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		playlists, err := client.GetPlaylists(ctx)
		return playlistsMsg{id: id, playlists: playlists, err: err}
	}
}

func (m PlaylistsModel) Update(msg tea.Msg) (PlaylistsModel, tea.Cmd) {
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

	case playlistsMsg:
		if msg.id != m.loadReqID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.playlists = msg.playlists
			if m.listCursor >= len(m.playlists) {
				m.listCursor = 0
			}
		}

	case playlistTracksMsg:
		if msg.id != m.loadReqID {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.tracks = msg.tracks
			m.open = true
			m.trackCursor = 0
		}
	}

	return m, nil
}

func (m PlaylistsModel) handleKey(msg tea.KeyMsg) (PlaylistsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.open {
			if m.trackCursor > 0 {
				m.trackCursor--
			}
		} else if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.open {
			if m.trackCursor < len(m.tracks)-1 {
				m.trackCursor++
			}
		} else if m.listCursor < len(m.playlists)-1 {
			m.listCursor++
		}
	case "enter", "l", "right":
		if !m.open {
			return m.openSelected()
		}
		if len(m.tracks) > 0 {
			m.err = m.playback.ToggleQueueTrack(m.tracks, m.trackCursor)
		}
	case "esc", "h", "left":
		m.open = false
	case "p":
		if m.open && len(m.tracks) > 0 {
			m.err = m.playback.ToggleQueueTrack(m.tracks, m.trackCursor)
		} else if !m.open {
			return m.openSelected()
		}
	case "q":
		if m.open && len(m.tracks) > 0 {
			m.err = m.playback.QueueTrack(m.tracks[m.trackCursor])
		}
	case "n":
		_ = m.playback.Next()
	case "m":
		_ = m.playback.Previous()
	case "r":
		_ = m.playback.Replay()
	case "ctrl+r":
		if m.open {
			return m.openSelected()
		}
		m.loading = true
		m.loadReqID = nextRequestID()
		return m, m.Init()
	}
	return m, nil
}

func (m PlaylistsModel) openSelected() (PlaylistsModel, tea.Cmd) {
	if m.listCursor >= len(m.playlists) {
		return m, nil
	}
	m.selected = m.playlists[m.listCursor]
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
	playlistID := m.selected.ID
	return m, func() tea.Msg {
		defer cancel()
		tracks, err := client.GetPlaylistTracks(ctx, playlistID)
		return playlistTracksMsg{id: id, tracks: tracks, err: err}
	}
}

func (m PlaylistsModel) View() string {
	w, h := normalizeViewSize(m.width, m.height)
	boxStyle := listBoxStyle(w, h)
	innerW := w - 8
	divider := listDivider(innerW)

	titleText := "◎ PLAYLISTS"
	if m.open {
		titleText += "  »  " + truncateStr(m.selected.Name, 32)
	}
	title := retroTitleStyle.Render(titleText)

	if m.err != nil {
		return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			title, divider, "", retroErrorStyle.Render("ERROR: "+m.err.Error())))
	}
	if m.loading {
		return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
			title, divider, "", retroLoadingStyle.Render("Loading...")))
	}

	visibleRows := calcVisibleRows(h, 8)
	lines := []string{title, divider}
	if m.open {
		lines = append(lines, m.renderTrackRows(innerW, visibleRows)...)
	} else {
		lines = append(lines, m.renderPlaylistRows(innerW, visibleRows)...)
	}
	lines = append(lines, divider)
	lines = append(lines, retroSubtleStyle.Render("  [enter/l] open  [esc/h] back  [p] play  [q] queue"))
	return boxStyle.Render(strings.Join(lines, "\n"))
}

func (m PlaylistsModel) renderPlaylistRows(innerW, visibleRows int) []string {
	if len(m.playlists) == 0 {
		return []string{"", retroSubtleStyle.Render("  No playlists found")}
	}
	start, end := scrollWindow(m.listCursor, len(m.playlists), visibleRows)
	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		p := m.playlists[i]
		label := fmt.Sprintf("%s  %s", truncateStr(p.Name, innerW-24), retroSubtleStyle.Render(fmt.Sprintf("%d tracks", p.TrackCount)))
		out = append(out, m.renderRow(label, i == m.listCursor))
	}
	return out
}

func (m PlaylistsModel) renderTrackRows(innerW, visibleRows int) []string {
	if len(m.tracks) == 0 {
		return []string{"", retroSubtleStyle.Render("  Playlist is empty")}
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
		label := fmt.Sprintf("%02d  %s  %s", i+1, truncateStr(t.Title, innerW-20), retroSubtleStyle.Render(FormatDuration(t.Duration)))
		if i == current {
			out = append(out, retroCurrentStyle.Render("▶ ")+retroCurrentStyle.Render(label))
			continue
		}
		out = append(out, m.renderRow(label, i == m.trackCursor))
	}
	return out
}

func (m PlaylistsModel) renderRow(label string, selected bool) string {
	if selected {
		return retroSelectedStyle.Render("► ") + retroSelectedStyle.Render(label)
	}
	return retroSubtleStyle.Render("  ") + lipgloss.NewStyle().Foreground(colorLightText).Render(label)
}
