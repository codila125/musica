package views

import (
	"context"
	"fmt"
	"strings"

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
				if err := m.player.PlayQueue(m.tracks, m.cursor); err != nil {
					m.err = fmt.Errorf("play: %w", err)
				} else {
					m.err = nil
					m.debug = "Playing: " + m.tracks[m.cursor].Title
				}
			}
		case "r":
			m.loading = true
			return m, m.loadRecentTracks
		}

	case []models.Track:
		m.tracks = msg
		m.cursor = 0
		m.loading = false
		m.err = nil
		m.debug = fmt.Sprintf("Loaded %d recent tracks", len(msg))

	case error:
		m.loading = false
		m.err = msg
		m.debug = msg.Error()
	}

	return m, nil
}

func (m BrowseModel) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("Error: " + m.err.Error())
	}
	if m.loading {
		return "Loading recent additions..."
	}

	head := "Recent Additions (enter/p: play, r: refresh)"
	lines := []string{head, strings.Repeat("-", len(head))}

	if len(m.tracks) == 0 {
		lines = append(lines, "No tracks found", "", m.debug)
		return strings.Join(lines, "\n")
	}

	maxRows := m.height - 6
	if maxRows < 8 {
		maxRows = 20
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.tracks) {
		end = len(m.tracks)
	}

	for i := start; i < end; i++ {
		t := m.tracks[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s - %s", prefix, t.Title, t.Artist))
	}

	if m.debug != "" {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(m.debug))
	}

	return strings.Join(lines, "\n")
}

func (m BrowseModel) loadRecentTracks() tea.Msg {
	tracks, err := m.apiClient.GetRecentTracks(context.Background(), 200)
	if err != nil {
		return err
	}
	return tracks
}
