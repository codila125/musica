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
	w, h := normalizeViewSize(m.width, m.height)

	queue := m.player.Queue()
	current := m.player.CurrentIndex()

	boxStyle := listBoxStyle(w, h)

	title := retroTitleStyle.Render("◎ TAPE QUEUE")
	divider := listDivider(w - 8)
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
	visibleRows := calcVisibleRows(h, 8)

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
	innerW := w - 8
	nameW := trackNameWidth(innerW)
	header := trackTableHeader(nameW)
	lines = append(lines, header)

	for i := start; i < end; i++ {
		t := queue[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(t.Title, nameW)
		artist := truncateStr(t.Artist, artistColWidth)
		album := truncateStr(t.Album, albumColWidth)
		dur := formatDuration(t.Duration)

		var line string
		if i == current {
			line = retroCurrentStyle.Render(fmt.Sprintf("▶ %s ", num)) +
				retroCurrentStyle.Render(padRight(name, nameW)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(artist, artistColWidth)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(album, albumColWidth)) +
				retroSubtleStyle.Render(" ") +
				lipgloss.NewStyle().Foreground(colorGreenSelect).Render(padRight(dur, durationColWidth))
		} else if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("► %s ", num)) +
				retroSelectedStyle.Render(padRight(name, nameW)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(artist, artistColWidth)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(album, albumColWidth)) +
				retroSubtleStyle.Render(" ") +
				lipgloss.NewStyle().Foreground(colorAmber).Render(padRight(dur, durationColWidth))
		} else {
			line = retroSubtleStyle.Render(fmt.Sprintf("  %s ", num)) +
				lipgloss.NewStyle().Foreground(colorLightText).Render(padRight(name, nameW)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(artist, artistColWidth)) +
				retroSubtleStyle.Render(" ") +
				retroSubtleStyle.Render(padRight(album, albumColWidth)) +
				retroSubtleStyle.Render(" ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(padRight(dur, durationColWidth))
		}
		lines = append(lines, line)
	}

	lines = append(lines, divider)
	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Track %d of %d", m.cursor+1, len(queue))))
	lines = append(lines, keys)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(content)
}
