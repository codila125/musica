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
	ti.Placeholder = "Type to search..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

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
		m.input.Width = msg.Width - 10
		if m.input.Width < 20 {
			m.input.Width = 20
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.state == SearchInput {
				query := m.input.Value()
				if query != "" {
					m.loading = true
					m.state = SearchResults
					return m, m.search(query)
				}
			} else if m.state == SearchResults {
				return m.handlePlay(), nil
			}
		case "p":
			if m.state == SearchResults {
				if m.resultType == 0 && len(m.results.Tracks) > 0 && m.cursor < len(m.results.Tracks) {
					cur := m.player.CurrentTrack()
					selected := m.results.Tracks[m.cursor]
					if cur != nil && cur.ID == selected.ID && m.player.State() == models.StatePlaying {
						_ = m.player.Pause()
					} else if cur != nil && cur.ID == selected.ID && m.player.State() == models.StatePaused {
						_ = m.player.Resume()
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
			case "left", "h":
				m.resultType = (m.resultType - 1 + 3) % 3
				m.cursor = 0
			case "right", "l":
				m.resultType = (m.resultType + 1) % 3
				m.cursor = 0
			case "q":
				if m.resultType == 0 && len(m.results.Tracks) > 0 && m.cursor < len(m.results.Tracks) {
					track := m.results.Tracks[m.cursor]
					if err := m.player.AppendToQueue(track); err != nil {
						m.err = fmt.Errorf("queue: %w", err)
					} else {
						m.err = nil
						m.input.Placeholder = "Queued: " + track.Title
					}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SearchModel) View() string {
	w := m.width
	h := m.height
	if w < 40 {
		w = 40
	}
	if h < 10 {
		h = 10
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurpleBorder).
		Padding(0, 1).
		Width(w - 4).
		Height(h - 2)

	if m.err != nil {
		content := lipgloss.JoinVertical(lipgloss.Left,
			retroTitleStyle.Render("◎ SEARCH"),
			retroCassetteStyle.Render(strings.Repeat("─", w-8)),
			"",
			retroErrorStyle.Render("ERROR: "+m.err.Error()),
		)
		return boxStyle.Render(content)
	}

	if m.state == SearchInput {
		return m.renderInputView(boxStyle, w)
	}

	if m.loading {
		content := lipgloss.JoinVertical(lipgloss.Left,
			retroTitleStyle.Render("◎ SEARCH"),
			retroCassetteStyle.Render(strings.Repeat("─", w-8)),
			"",
			retroLoadingStyle.Render(m.spinner.View()+" Searching..."),
		)
		return boxStyle.Render(content)
	}

	return m.renderResultsView(boxStyle, w, h)
}

func (m SearchModel) renderInputView(boxStyle lipgloss.Style, w int) string {
	title := retroTitleStyle.Render("◎ SEARCH DECK")
	divider := retroCassetteStyle.Render(strings.Repeat("─", w-8))
	keys := retroSubtleStyle.Render("[enter]search  [esc]back")

	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAmber).
		Padding(0, 1).
		Render(m.input.View())

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		divider,
		"",
		retroSubtleStyle.Render("  Enter search query:"),
		"",
		"  "+searchBox,
		"",
		divider,
		keys,
	)
	return boxStyle.Render(content)
}

func (m SearchModel) renderResultsView(boxStyle lipgloss.Style, w, h int) string {
	title := retroTitleStyle.Render("◎ SEARCH RESULTS")
	divider := retroCassetteStyle.Render(strings.Repeat("─", w-8))
	keys := retroSubtleStyle.Render("[tab]category [p]lay [q]ueue [esc]back")

	// Category tabs
	types := []string{"TRACKS", "ALBUMS", "ARTISTS"}
	var tabs []string
	for i, t := range types {
		if i == m.resultType {
			tabs = append(tabs, retroSelectedStyle.Render("["+t+"]"))
		} else {
			tabs = append(tabs, retroSubtleStyle.Render(" "+t+" "))
		}
	}
	tabBar := strings.Join(tabs, " ")

	// Results content
	var resultContent string
	switch m.resultType {
	case 0:
		resultContent = m.renderTracks(w, h)
	case 1:
		resultContent = m.renderAlbums(w, h)
	case 2:
		resultContent = m.renderArtists(w, h)
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		divider,
		tabBar,
		divider,
		resultContent,
		divider,
		keys,
	)
	return boxStyle.Render(content)
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

func (m SearchModel) renderTracks(w, h int) string {
	if len(m.results.Tracks) == 0 {
		return retroSubtleStyle.Render("  No tracks found")
	}

	visibleRows := h - 12
	if visibleRows < 3 {
		visibleRows = 3
	}

	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(m.results.Tracks) {
		end = len(m.results.Tracks)
	}

	nameW := w - 91
	if nameW < 10 {
		nameW = 10
	}

	var lines []string

	// Column headers
	header := retroSubtleStyle.Render("  # ") +
		retroColumnHeaderStyle.Render(padRight("NAME", nameW)) +
		retroSubtleStyle.Render(" ") +
		retroColumnHeaderStyle.Render(padRight("ARTIST", 35)) +
		retroSubtleStyle.Render(" ") +
		retroColumnHeaderStyle.Render(padRight("ALBUM", 40)) +
		retroSubtleStyle.Render(" ") +
		retroColumnHeaderStyle.Render(padRight("DURATION", 8))
	lines = append(lines, header)

	for i := start; i < end; i++ {
		t := m.results.Tracks[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(t.Title, nameW)
		artist := truncateStr(t.Artist, 35)
		album := truncateStr(t.Album, 40)
		dur := formatDuration(t.Duration)

		var line string
		if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("▶ %s ", num)) +
				retroSelectedStyle.Render(padRight(name, nameW)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(artist, 35)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(album, 40)) +
				retroSubtleStyle.Render(" ") +
				lipgloss.NewStyle().Foreground(colorAmber).Render(padRight(dur, 8))
		} else {
			line = retroSubtleStyle.Render(fmt.Sprintf("  %s ", num)) +
				lipgloss.NewStyle().Foreground(colorLightText).Render(padRight(name, nameW)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(artist, 35)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(album, 40)) +
				retroSubtleStyle.Render(" ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(padRight(dur, 8))
		}
		lines = append(lines, line)
	}

	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Track %d of %d", m.cursor+1, len(m.results.Tracks))))
	return strings.Join(lines, "\n")
}

func (m SearchModel) renderAlbums(w, h int) string {
	if len(m.results.Albums) == 0 {
		return retroSubtleStyle.Render("  No albums found")
	}

	visibleRows := h - 12
	if visibleRows < 3 {
		visibleRows = 3
	}

	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(m.results.Albums) {
		end = len(m.results.Albums)
	}

	var lines []string
	for i := start; i < end; i++ {
		a := m.results.Albums[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(a.Name, w-20)
		artist := truncateStr(a.Artist, 15)

		var line string
		if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("▶ %s %s", num, name)) +
				retroSubtleStyle.Render(" - "+artist)
		} else {
			line = retroSubtleStyle.Render(fmt.Sprintf("  %s ", num)) +
				lipgloss.NewStyle().Foreground(colorLightText).Render(name) +
				retroSubtleStyle.Render(" - "+artist)
		}
		lines = append(lines, line)
	}

	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Album %d of %d", m.cursor+1, len(m.results.Albums))))
	return strings.Join(lines, "\n")
}

func (m SearchModel) renderArtists(w, h int) string {
	if len(m.results.Artists) == 0 {
		return retroSubtleStyle.Render("  No artists found")
	}

	visibleRows := h - 12
	if visibleRows < 3 {
		visibleRows = 3
	}

	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	end := start + visibleRows
	if end > len(m.results.Artists) {
		end = len(m.results.Artists)
	}

	var lines []string
	for i := start; i < end; i++ {
		a := m.results.Artists[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(a.Name, w-12)

		var line string
		if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("▶ %s %s", num, name))
		} else {
			line = retroSubtleStyle.Render(fmt.Sprintf("  %s ", num)) +
				lipgloss.NewStyle().Foreground(colorLightText).Render(name)
		}
		lines = append(lines, line)
	}

	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Artist %d of %d", m.cursor+1, len(m.results.Artists))))
	return strings.Join(lines, "\n")
}
