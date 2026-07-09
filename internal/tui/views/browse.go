package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/models"
)

const browsePageSize = 50

type BrowseModel struct {
	apiClient api.Client
	playback  PlaybackService

	tracks     []models.Track
	fetched    []models.Track // largest track set fetched so far, cached across page turns
	page       int
	total      int
	totalKnown bool
	cursor     int
	width      int
	height     int
	loading    bool
	err        error
	loadReqID  int64
	cancelLoad context.CancelFunc
}

type browseTracksMsg struct {
	id     int64
	tracks []models.Track
	count  int
	err    error
}

func NewBrowseModel(client api.Client, pl PlaybackService) BrowseModel {
	return BrowseModel{
		apiClient: client,
		playback:  pl,
		loading:   true,
		loadReqID: nextRequestID(),
	}
}

func (m BrowseModel) Init() tea.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	id := m.loadReqID
	return func() tea.Msg {
		defer cancel()
		return m.fetchTracksMsg(ctx, id, true)
	}
}

func (m BrowseModel) Update(msg tea.Msg) (BrowseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tracks)-1 {
				m.cursor++
			}
		case "left":
			if m.page > 0 {
				m.page--
				return m.beginLoadRecentTracks(false)
			}
		case "right":
			m.page++
			return m.beginLoadRecentTracks(false)
		case "enter", "p":
			if len(m.tracks) > 0 {
				m.err = m.playback.ToggleQueueTrack(m.tracks, m.cursor)
			}
		case "n":
			if len(m.tracks) > 0 {
				m.err = m.playback.ToggleQueueTrack(m.tracks, m.cursor)
				if m.err == nil {
					m.err = m.playback.Next()
				}
			}
		case "m":
			m.err = m.playback.Previous()
		case "r":
			m.err = m.playback.Replay()
		case "ctrl+r":
			return m.beginLoadRecentTracks(true)
		case "q":
			if len(m.tracks) > 0 {
				if err := m.playback.QueueTrack(m.tracks[m.cursor]); err != nil {
					m.err = fmt.Errorf("queue: %w", err)
				} else {
					m.err = nil
				}
			}
		}

	case cancelInFlightMsg:
		if m.cancelLoad != nil {
			m.cancelLoad()
			m.cancelLoad = nil
		}
		m.loadReqID = nextRequestID()
		m.loading = false

	case browseTracksMsg:
		if msg.id != m.loadReqID {
			return m, nil
		}
		m.cancelLoad = nil
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		countKnown := msg.count >= 0
		if msg.count == 0 && len(msg.tracks) > 0 {
			countKnown = false
		}
		m.fetched = msg.tracks
		return m.applyPage(msg.tracks, msg.count, countKnown), nil
	}

	return m, nil
}

// applyPage recomputes total/totalKnown and slices tracks down to the
// current page. Shared by a fresh server response and a cache hit that
// reuses a previously fetched track set.
func (m BrowseModel) applyPage(tracks []models.Track, count int, countKnown bool) BrowseModel {
	m.totalKnown = countKnown
	if countKnown {
		m.total = count
	} else {
		m.total = len(tracks)
	}
	pageSize := browsePageSize
	maxPage := 0
	if m.total > 0 {
		maxPage = (m.total - 1) / pageSize
	}
	if m.page > maxPage {
		m.page = maxPage
	}
	start := m.page * pageSize
	end := start + pageSize
	if start > m.total {
		start = m.total
	}
	if end > m.total {
		end = m.total
	}
	if start > len(tracks) {
		start = len(tracks)
	}
	if end > len(tracks) {
		end = len(tracks)
	}
	m.tracks = append([]models.Track(nil), tracks[start:end]...)
	if m.cursor >= len(m.tracks) {
		m.cursor = len(m.tracks) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.err = nil
	return m
}

func (m BrowseModel) View() string {
	w, h := normalizeViewSize(m.width, m.height)

	boxStyle := listBoxStyle(w, h)

	if m.err != nil {
		content := lipgloss.JoinVertical(lipgloss.Left,
			retroTitleStyle.Render("◎ TRACK LIBRARY"),
			listDivider(w-8),
			"",
			retroErrorStyle.Render("ERROR: "+m.err.Error()),
		)
		return boxStyle.Render(content)
	}

	if m.loading {
		content := lipgloss.JoinVertical(lipgloss.Left,
			retroTitleStyle.Render("◎ TRACK LIBRARY"),
			listDivider(w-8),
			"",
			retroLoadingStyle.Render("Loading tracks..."),
		)
		return boxStyle.Render(content)
	}

	innerW := w - 8
	title := retroTitleStyle.Render("◎ TRACK LIBRARY")
	divider := listDivider(innerW)

	if len(m.tracks) == 0 {
		content := lipgloss.JoinVertical(lipgloss.Left,
			title,
			divider,
			"",
			retroSubtleStyle.Render("  No tracks found"),
			"",
			divider,
		)
		return boxStyle.Render(content)
	}

	// Calculate visible rows
	visibleRows := calcVisibleRows(h, 8)
	start, end := scrollWindow(m.cursor, len(m.tracks), visibleRows)

	lines := []string{title, divider}

	// Cassette animation indicator (compact, inline)
	cassetteStatus := m.renderCompactCassette()
	lines = append(lines, cassetteStatus)
	lines = append(lines, listDivider(innerW))

	// Column headers
	cols := computeTrackColumns(innerW)
	header := trackTableHeader(cols)
	lines = append(lines, header)

	for i := start; i < end; i++ {
		t := m.tracks[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(t.Title, cols.nameW)
		artist := truncateStr(t.Artist, cols.artistW)
		album := truncateStr(t.Album, cols.albumW)
		dur := FormatDuration(t.Duration)

		var line string
		if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("▶ %s ", num)) +
				retroSelectedStyle.Render(padRight(name, cols.nameW))
			if cols.showArtist {
				line += retroSubtleStyle.Render(" ") + retroSubtleStyle.Render(padRight(artist, cols.artistW))
			}
			if cols.showAlbum {
				line += retroSubtleStyle.Render(" ") + retroSubtleStyle.Render(padRight(album, cols.albumW))
			}
			if cols.showDuration {
				line += retroSubtleStyle.Render(" ") + lipgloss.NewStyle().Foreground(colorAmber).Render(padRight(dur, cols.durationW))
			}
		} else {
			line = retroSubtleStyle.Render(fmt.Sprintf("  %s ", num)) +
				lipgloss.NewStyle().Foreground(colorLightText).Render(padRight(name, cols.nameW))
			if cols.showArtist {
				line += retroSubtleStyle.Render(" ") + retroSubtleStyle.Render(padRight(artist, cols.artistW))
			}
			if cols.showAlbum {
				line += retroSubtleStyle.Render(" ") + retroSubtleStyle.Render(padRight(album, cols.albumW))
			}
			if cols.showDuration {
				line += retroSubtleStyle.Render(" ") + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(padRight(dur, cols.durationW))
			}
		}
		lines = append(lines, line)
	}

	lines = append(lines, divider)
	pageLabel := ""
	if m.total > 0 {
		pageLabel = fmt.Sprintf("  Page %d of %d", m.page+1, (m.total-1)/browsePageSize+1)
	}
	totalLabel := ""
	if m.totalKnown && m.total > 0 {
		totalLabel = fmt.Sprintf(" (Total %d)", m.total)
	}
	lines = append(lines, retroSubtleStyle.Align(lipgloss.Center).Width(innerW).Render(fmt.Sprintf("Track %d of %d%s%s", m.cursor+1, len(m.tracks), totalLabel, pageLabel)))

	content := strings.Join(lines, "\n")
	return boxStyle.Render(content)
}

func (m BrowseModel) renderCompactCassette() string {
	state := m.playback.State()
	interval := 200 * time.Millisecond
	moving := false
	stateLabel := "■ STOP"
	ledColor := colorRedDim

	switch state {
	case models.StatePlaying:
		interval = 80 * time.Millisecond
		stateLabel = "▶ PLAY"
		moving = true
		ledColor = colorGreenSelect
	case models.StatePaused:
		interval = 300 * time.Millisecond
		stateLabel = "❚❚ PAUSE"
		moving = true
		ledColor = colorAmber
	}

	// Simple spinning reel animation
	reels := []string{"◐", "◓", "◑", "◒"}
	frame := 0
	if moving {
		frame = int((time.Now().UnixNano() / int64(interval)) % int64(len(reels)))
	}

	reel := reels[frame]
	ledStyle := lipgloss.NewStyle().Foreground(ledColor).Bold(true)

	// Compact cassette: [reel]====[reel] STATUS
	cassette := retroCassetteStyle.Render("  ╔══") +
		ledStyle.Render(reel) +
		retroCassetteStyle.Render("════════") +
		ledStyle.Render(reel) +
		retroCassetteStyle.Render("══╗ ") +
		ledStyle.Render("● "+stateLabel)

	return cassette
}

// beginLoadRecentTracks starts a page load. forceCount refetches the library
// total even if already known (used for an explicit user refresh) and skips
// the cache; otherwise a page turn that stays within the largest track set
// already fetched is served straight from that cache with no network call,
// since paging back and forth over recently seen tracks is common and the
// underlying data rarely changes mid-session.
func (m BrowseModel) beginLoadRecentTracks(forceCount bool) (BrowseModel, tea.Cmd) {
	required := (m.page + 1) * browsePageSize
	haveEnough := len(m.fetched) >= required || (m.totalKnown && len(m.fetched) >= m.total)
	if !forceCount && haveEnough {
		return m.applyPage(m.fetched, m.total, m.totalKnown), nil
	}

	if m.cancelLoad != nil {
		m.cancelLoad()
		m.cancelLoad = nil
	}

	m.loadReqID = nextRequestID()
	m.loading = true
	m.err = nil

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	m.cancelLoad = cancel
	id := m.loadReqID
	return m, func() tea.Msg {
		return m.fetchTracksMsg(ctx, id, forceCount)
	}
}

func (m BrowseModel) fetchTracksMsg(ctx context.Context, id int64, forceCount bool) tea.Msg {
	limit := (m.page + 1) * browsePageSize
	tracks, err := m.apiClient.GetRecentTracks(ctx, limit)

	count := m.total
	if forceCount || !m.totalKnown {
		c, countErr := m.apiClient.GetRecentTracksCount(ctx)
		if countErr != nil {
			count = -1
		} else {
			count = c
		}
	}

	return browseTracksMsg{id: id, tracks: tracks, count: count, err: err}
}

// Helper functions
func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	s = sanitizeDisplay(s)
	if runewidth.StringWidth(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return runewidth.Truncate(s, maxLen, "")
	}
	return runewidth.Truncate(s, maxLen, "...")
}

func FormatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func padRight(s string, w int) string {
	s = sanitizeDisplay(s)
	if runewidth.StringWidth(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-runewidth.StringWidth(s))
}
