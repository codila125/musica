package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
			if m.cursor < len(queue)-1 {
				m.cursor++
			}
		case "enter":
			queue := m.player.Queue()
			if m.cursor < len(queue) {
				_ = m.player.PlayQueue(queue, m.cursor)
			}
		}
	}

	return m, nil
}

func (m QueueModel) View() string {
	queue := m.player.Queue()
	current := m.player.CurrentIndex()

	if len(queue) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Queue is empty")
	}

	var lines []string
	for i, t := range queue {
		prefix := "  "
		if i == current {
			prefix = "▶ "
		} else if i == m.cursor {
			prefix = "> "
		}

		min := t.Duration / 60
		sec := t.Duration % 60
		line := fmt.Sprintf("%s%d. %s - %s (%d:%02d)", prefix, i+1, t.Title, t.Artist, min, sec)
		lines = append(lines, line)
	}

	start := 0
	visibleRows := m.height - 3
	if visibleRows < 1 {
		visibleRows = len(lines)
	}

	if m.cursor > visibleRows-1 {
		start = m.cursor - (visibleRows - 1)
	}
	if start < 0 {
		start = 0
	}
	if start > len(lines) {
		start = len(lines)
	}

	end := start + visibleRows
	if end > len(lines) {
		end = len(lines)
	}
	if end < start {
		end = start
	}

	return lipgloss.NewStyle().Render(strings.Join(lines[start:end], "\n"))
}
