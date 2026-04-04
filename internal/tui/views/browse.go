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
		return retroPanelForWidth(m.width).Render(retroErrorStyle.Render("Error: " + m.err.Error()))
	}
	if m.loading {
		return retroPanelForWidth(m.width).Render(retroLoadingStyle.Render("Loading recent additions..."))
	}

	title := retroTitleStyle.Render("RETRO TAPE PLAYER") + " " + retroSubtleStyle.Render("(enter/p play, q queue, r refresh)")
	divider := retroSubtleStyle.Render(strings.Repeat("=", 68))

	contentWidth := 72
	if m.width > 0 {
		contentWidth = m.width - 12
	}
	if contentWidth < 40 {
		contentWidth = 40
	}

	leftWidth := (contentWidth * 58) / 100
	if leftWidth < 24 {
		leftWidth = 24
	}
	rightWidth := contentWidth - leftWidth - 1
	if rightWidth < 15 {
		rightWidth = 15
		leftWidth = contentWidth - rightWidth - 1
	}

	listPanel := lipgloss.NewStyle().Width(leftWidth).MaxWidth(leftWidth).Render(m.renderTrackList())
	tapePanel := lipgloss.NewStyle().Width(rightWidth).MaxWidth(rightWidth).Render(m.renderTapeDeck())
	content := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", tapePanel)

	return retroPanelForWidth(m.width).Render(title + "\n" + divider + "\n" + content)
}

func (m BrowseModel) renderTrackList() string {
	lines := []string{retroTitleStyle.Render("TRACK LIST")}

	if len(m.tracks) == 0 {
		lines = append(lines, retroSubtleStyle.Render("No tracks found"))
		return strings.Join(lines, "\n")
	}

	visibleRows := 10
	start := 0
	if m.cursor >= visibleRows {
		start = m.cursor - visibleRows + 1
	}
	if start < 0 {
		start = 0
	}
	end := start + visibleRows
	if end > len(m.tracks) {
		end = len(m.tracks)
	}

	for i := start; i < end; i++ {
		t := m.tracks[i]
		prefix := retroSubtleStyle.Render("  ")
		if i == m.cursor {
			prefix = retroSelectedStyle.Render(">> ")
		}
		line := fmt.Sprintf("%s%02d %s %s", prefix, i+1, t.Title, retroSubtleStyle.Render("- "+t.Artist))
		lines = append(lines, line)
	}

	lines = append(lines, "", retroSubtleStyle.Render(fmt.Sprintf("showing 10/%d (j/k to scroll)", len(m.tracks))))
	if m.debug != "" {
		lines = append(lines, retroSubtleStyle.Render(m.debug))
	}

	return strings.Join(lines, "\n")
}

func (m BrowseModel) renderTapeDeck() string {
	frames := []string{"|", "/", "-", "\\", "|", "/", "-", "\\"}
	state := m.player.State()
	interval := 125 * time.Millisecond
	moving := true
	stateLabel := "STOP"

	switch state {
	case models.StatePlaying:
		interval = 90 * time.Millisecond
		stateLabel = "PLAY"
	case models.StatePaused:
		interval = 450 * time.Millisecond
		stateLabel = "PAUSE"
	default:
		moving = false
	}

	step := 0
	if moving {
		step = int((time.Now().UnixNano() / int64(interval)) % int64(len(frames)))
	}
	l1 := frames[step]
	l2 := frames[(step+2)%len(frames)]
	r1 := frames[(step+4)%len(frames)]
	r2 := frames[(step+6)%len(frames)]

	deck := []string{
		retroTitleStyle.Render("TAPE WHEELS"),
		retroSubtleStyle.Render("    ______________   ______________"),
		retroSubtleStyle.Render(fmt.Sprintf(`   /   %s   O   %s  \ /   %s   O   %s  \`, l1, l2, r1, r2)),
		retroSubtleStyle.Render(fmt.Sprintf("  |      %s(%s)%s      ||      %s(%s)%s      |", l2, l1, r2, r2, r1, l1)),
		retroSubtleStyle.Render(fmt.Sprintf("   \\__ %s ___ %s __// \\__ %s ___ %s __//", l2, l1, r2, r1)),
		retroSubtleStyle.Render("       |_____________________________|"),
		"",
		retroSubtleStyle.Render("          MODE: " + stateLabel),
		retroSubtleStyle.Render("       [PLAY] [STOP] [REW]"),
		retroSubtleStyle.Render("        [FF]  [PAUSE] [REC]"),
	}

	return strings.Join(deck, "\n")
}

func (m BrowseModel) loadRecentTracks() tea.Msg {
	tracks, err := m.apiClient.GetRecentTracks(context.Background(), 20)
	if err != nil {
		return err
	}
	return tracks
}
