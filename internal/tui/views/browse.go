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
	"github.com/codila125/musica/internal/player"
)

type BrowseModel struct {
	apiClient api.Client
	player    *player.Player

	tracks  []models.Track
	cursor  int
	width   int
	height  int
	loading bool
	err     error
	debug   string
}

func NewBrowseModel(client api.Client, pl *player.Player) BrowseModel {
	return BrowseModel{
		apiClient: client,
		player:    pl,
		loading:   true,
	}
}

func (m BrowseModel) Init() tea.Cmd {
	return m.loadRecentTracks
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
		case "enter", "p":
			if len(m.tracks) > 0 {
				cur := m.player.CurrentTrack()
				if cur != nil && cur.ID == m.tracks[m.cursor].ID && m.player.State() == models.StatePlaying {
					if err := m.player.Pause(); err != nil {
						m.err = fmt.Errorf("pause: %w", err)
					} else {
						m.err = nil
						m.debug = "Paused"
					}
				} else if cur != nil && cur.ID == m.tracks[m.cursor].ID && m.player.State() == models.StatePaused {
					if err := m.player.Resume(); err != nil {
						m.err = fmt.Errorf("resume: %w", err)
					} else {
						m.err = nil
						m.debug = "Resumed"
					}
				} else {
					if err := m.player.Play(m.tracks[m.cursor]); err != nil {
						m.err = fmt.Errorf("play: %w", err)
					} else {
						m.err = nil
						m.debug = "Playing: " + m.tracks[m.cursor].Title
					}
				}
			}
		case "r":
			m.loading = true
			return m, m.loadRecentTracks
		case "q":
			if len(m.tracks) > 0 {
				if err := m.player.AppendToQueue(m.tracks[m.cursor]); err != nil {
					m.err = fmt.Errorf("queue: %w", err)
				} else {
					m.err = nil
					m.debug = "Queued: " + m.tracks[m.cursor].Title
				}
			}
		}

	case []models.Track:
		m.tracks = msg
		m.cursor = 0
		m.loading = false
		m.err = nil
		m.debug = fmt.Sprintf("Loaded %d tracks", len(msg))

	case error:
		m.loading = false
		m.err = msg
		m.debug = msg.Error()
	}

	return m, nil
}

func (m BrowseModel) View() string {
	w := m.width
	h := m.height
	if w < 40 {
		w = 40
	}
	if h < 10 {
		h = 10
	}

	// Single box like queue view
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurpleBorder).
		Padding(0, 1).
		Width(w - 4).
		Height(h - 2)

	if m.err != nil {
		content := lipgloss.JoinVertical(lipgloss.Left,
			retroTitleStyle.Render("◎ TRACK LIBRARY"),
			retroCassetteStyle.Render(strings.Repeat("─", w-8)),
			"",
			retroErrorStyle.Render("ERROR: "+m.err.Error()),
		)
		return boxStyle.Render(content)
	}

	if m.loading {
		content := lipgloss.JoinVertical(lipgloss.Left,
			retroTitleStyle.Render("◎ TRACK LIBRARY"),
			retroCassetteStyle.Render(strings.Repeat("─", w-8)),
			"",
			retroLoadingStyle.Render("Loading tracks..."),
		)
		return boxStyle.Render(content)
	}

	innerW := w - 8
	title := retroTitleStyle.Render("◎ TRACK LIBRARY")
	divider := retroCassetteStyle.Render(strings.Repeat("─", innerW))
	keys := retroSubtleStyle.Render("[p]lay/pause  [q]ueue  [r]efresh  [j/k]scroll")

	if len(m.tracks) == 0 {
		content := lipgloss.JoinVertical(lipgloss.Left,
			title,
			divider,
			"",
			retroSubtleStyle.Render("  No tracks found"),
			retroSubtleStyle.Render("  Press [r] to refresh"),
			"",
			divider,
			keys,
		)
		return boxStyle.Render(content)
	}

	// Calculate visible rows
	visibleRows := h - 8
	if visibleRows < 3 {
		visibleRows = 3
	}

	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(m.tracks) {
		end = len(m.tracks)
	}

	lines := []string{title, divider}

	// Cassette animation indicator (compact, inline)
	cassetteStatus := m.renderCompactCassette()
	lines = append(lines, cassetteStatus)

	for i := start; i < end; i++ {
		t := m.tracks[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(t.Title, innerW-20)
		artist := truncateStr(t.Artist, 12)
		dur := formatDuration(t.Duration)

		var line string
		if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("▶ %s %s", num, name)) +
				retroSubtleStyle.Render(" "+artist+" ") +
				lipgloss.NewStyle().Foreground(colorAmber).Render(dur)
		} else {
			line = retroSubtleStyle.Render(fmt.Sprintf("  %s ", num)) +
				lipgloss.NewStyle().Foreground(colorLightText).Render(name) +
				retroSubtleStyle.Render(" "+artist+" ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(dur)
		}
		lines = append(lines, line)
	}

	lines = append(lines, divider)
	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Track %d of %d", m.cursor+1, len(m.tracks))))
	lines = append(lines, keys)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(content)
}

func (m BrowseModel) renderCompactCassette() string {
	state := m.player.State()
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

func (m BrowseModel) loadRecentTracks() tea.Msg {
	tracks, err := m.apiClient.GetRecentTracks(context.Background(), 50)
	if err != nil {
		return err
	}
	return tracks
}

// Helper functions
func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func formatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
