package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

type SearchState int

const (
	SearchInput SearchState = iota
	SearchResults
)

type SearchModel struct {
	apiClient  api.Client
	player     *player.Player
	input      textinput.Model
	spinner    spinner.Model
	state      SearchState
	results    models.SearchResult
	loading    bool
	err        error
	width      int
	height     int
	cursor     int
	resultType int
}

func NewSearchModel(client api.Client, pl *player.Player) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search for artists, albums, or tracks..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	s := spinner.New()
	s.Spinner = spinner.Dot

	return SearchModel{
		apiClient: client,
		player:    pl,
		input:     ti,
		spinner:   s,
		state:     SearchInput,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 4

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "p":
			if m.state == SearchInput {
				query := m.input.Value()
				if query != "" {
					m.loading = true
					m.state = SearchResults
					return m, m.search(query)
				}
			} else if m.state == SearchResults {
				if m.resultType == 0 && len(m.results.Tracks) > 0 && m.cursor < len(m.results.Tracks) {
					cur := m.player.CurrentTrack()
					selected := m.results.Tracks[m.cursor]
					if cur != nil && cur.ID == selected.ID && m.player.State() == models.StatePlaying {
						if err := m.player.Pause(); err != nil {
							m.err = fmt.Errorf("pause: %w", err)
						} else {
							m.err = nil
							m.input.Placeholder = "Paused"
						}
					} else if cur != nil && cur.ID == selected.ID && m.player.State() == models.StatePaused {
						if err := m.player.Resume(); err != nil {
							m.err = fmt.Errorf("resume: %w", err)
						} else {
							m.err = nil
							m.input.Placeholder = "Resumed"
						}
					} else {
						return m.handlePlay(), nil
					}
				}
			}
		case "esc":
			if m.state == SearchResults {
				m.state = SearchInput
				m.input.Focus()
				return m, textinput.Blink
			}
		case "tab":
			if m.state == SearchResults {
				m.resultType = (m.resultType + 1) % 3
				m.cursor = 0
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case models.SearchResult:
		m.results = msg
		m.loading = false
		m.state = SearchResults
		m.resultType = 0
		m.cursor = 0

	case error:
		m.err = msg
		m.loading = false
	}

	var cmd tea.Cmd
	if m.state == SearchInput {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				count := m.resultCount()
				if m.cursor < count-1 {
					m.cursor++
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SearchModel) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.state == SearchInput {
		return m.input.View()
	}

	if m.loading {
		return m.spinner.View() + " Searching..."
	}

	var content string
	types := []string{"Tracks", "Albums", "Artists"}

	for i, t := range types {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if i == m.resultType {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
		}
		content += style.Render(t) + "  "
	}
	content += "\n\n"

	switch m.resultType {
	case 0:
		content += m.renderTracks()
	case 1:
		content += m.renderAlbums()
	case 2:
		content += m.renderArtists()
	}

	return content
}

func (m SearchModel) search(query string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.apiClient.Search(context.Background(), query)
		if err != nil {
			return err
		}
		return result
	}
}

func (m SearchModel) handlePlay() SearchModel {
	switch m.resultType {
	case 0:
		if len(m.results.Tracks) > 0 && m.cursor < len(m.results.Tracks) {
			track := m.results.Tracks[m.cursor]
			if track.StreamURL == "" {
				m.err = fmt.Errorf("track has empty stream URL")
				return m
			}
			if err := m.player.Play(track); err != nil {
				m.err = fmt.Errorf("play: %w", err)
			} else {
				m.err = nil
				m.input.Placeholder = "Playing: " + track.Title
			}
		}
	}
	return m
}

func (m SearchModel) resultCount() int {
	switch m.resultType {
	case 0:
		return len(m.results.Tracks)
	case 1:
		return len(m.results.Albums)
	case 2:
		return len(m.results.Artists)
	}
	return 0
}

func (m SearchModel) renderTracks() string {
	if len(m.results.Tracks) == 0 {
		return "No tracks found"
	}

	var lines []string
	for i, t := range m.results.Tracks {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		min := t.Duration / 60
		sec := t.Duration % 60
		lines = append(lines, fmt.Sprintf("%s%s - %s (%d:%02d)", prefix, t.Title, t.Artist, min, sec))
	}

	visible := lines[:min(len(lines), m.height-6)]
	return lipgloss.NewStyle().Render(strings.Join(visible, "\n"))
}

func (m SearchModel) renderAlbums() string {
	if len(m.results.Albums) == 0 {
		return "No albums found"
	}

	var lines []string
	for i, a := range m.results.Albums {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s - %s", prefix, a.Name, a.Artist))
	}

	visible := lines[:min(len(lines), m.height-6)]
	return lipgloss.NewStyle().Render(strings.Join(visible, "\n"))
}

func (m SearchModel) renderArtists() string {
	if len(m.results.Artists) == 0 {
		return "No artists found"
	}

	var lines []string
	for i, a := range m.results.Artists {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s", prefix, a.Name))
	}

	visible := lines[:min(len(lines), m.height-6)]
	return lipgloss.NewStyle().Render(strings.Join(visible, "\n"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
