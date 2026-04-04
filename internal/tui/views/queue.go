package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

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
	queue := m.player.Queue()
	current := m.player.CurrentIndex()

	if queue == nil || len(queue) == 0 {
		return retroPanelForWidth(m.width).Render(retroSubtleStyle.Render("Queue is empty"))
	}

	lines := []string{
		retroTitleStyle.Render("Tape Queue") + " " + retroSubtleStyle.Render("(enter/p: play-pause, j/k: move)"),
		retroSubtleStyle.Render(strings.Repeat("-", 56)),
	}
	for i, t := range queue {
		prefix := retroSubtleStyle.Render("  ")
		if i == current {
			prefix = retroCurrentStyle.Render("** ")
		} else if i == m.cursor && i != current {
			prefix = retroSelectedStyle.Render(">> ")
		} else {
			prefix = retroSubtleStyle.Render("  ")
		}

		min := t.Duration / 60
		sec := t.Duration % 60
		line := fmt.Sprintf("%s%02d %s %s %s", prefix, i+1, t.Title, retroSubtleStyle.Render("- "+t.Artist), retroSubtleStyle.Render(fmt.Sprintf("(%d:%02d)", min, sec)))
		lines = append(lines, line)
	}

	start := 0
	visibleRows := 20
	if m.height > 0 {
		availableRows := m.height - 10
		if availableRows < visibleRows {
			visibleRows = availableRows
		}
	}
	if visibleRows < 5 {
		visibleRows = 5
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

	return retroPanelForWidth(m.width).Render(strings.Join(lines[start:end], "\n"))
}
