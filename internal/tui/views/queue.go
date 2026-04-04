package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

type QueueModel struct {
	player *player.Player
	width  int
	height int
	cursor int
}

func NewQueueModel(pl *player.Player) QueueModel {
	return QueueModel{
		player: pl,
	}
}

func (m QueueModel) Init() tea.Cmd {
	return nil
}

func (m QueueModel) Update(msg tea.Msg) (QueueModel, tea.Cmd) {
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
			queue := m.player.Queue()
			if queue != nil && m.cursor < len(queue)-1 {
				m.cursor++
			}
		case "enter", "p":
			queue := m.player.Queue()
			if queue != nil && m.cursor < len(queue) {
				cur := m.player.CurrentTrack()
				if cur != nil && cur.ID == queue[m.cursor].ID && m.player.State() == models.StatePlaying {
					_ = m.player.Pause()
				} else if cur != nil && cur.ID == queue[m.cursor].ID && m.player.State() == models.StatePaused {
					_ = m.player.Resume()
				} else {
					_ = m.player.PlayQueue(queue, m.cursor)
				}
			}
		}
	}

	return m, nil
}

func (m QueueModel) View() string {
	w := m.width
	h := m.height
	if w < 40 {
		w = 40
	}
	if h < 10 {
		h = 10
	}

	queue := m.player.Queue()
	current := m.player.CurrentIndex()

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurpleBorder).
		Padding(0, 1).
		Width(w - 4).
		Height(h - 2)

	title := retroTitleStyle.Render("◎ TAPE QUEUE")
	divider := retroCassetteStyle.Render(strings.Repeat("─", w-8))
	keys := retroSubtleStyle.Render("[p]lay/pause  [j/k]scroll")

	if queue == nil || len(queue) == 0 {
		content := lipgloss.JoinVertical(lipgloss.Left,
			title,
			divider,
			"",
			retroSubtleStyle.Render("  Queue is empty"),
			retroSubtleStyle.Render("  Add tracks from Browse tab"),
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
	if end > len(queue) {
		end = len(queue)
	}

	lines := []string{title, divider}

	// Column headers
	nameW := w - 91
	if nameW < 10 {
		nameW = 10
	}
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
		t := queue[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(t.Title, nameW)
		artist := truncateStr(t.Artist, 35)
		album := truncateStr(t.Album, 40)
		dur := formatDuration(t.Duration)

		var line string
		if i == current {
			line = retroCurrentStyle.Render(fmt.Sprintf("▶ %s ", num)) +
				retroCurrentStyle.Render(padRight(name, nameW)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(artist, 35)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(album, 40)) +
				retroSubtleStyle.Render(" ") +
				lipgloss.NewStyle().Foreground(colorGreenSelect).Render(padRight(dur, 8))
		} else if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("► %s ", num)) +
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

	lines = append(lines, divider)
	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Track %d of %d", m.cursor+1, len(queue))))
	lines = append(lines, keys)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(content)
}
