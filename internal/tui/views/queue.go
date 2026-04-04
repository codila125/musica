package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type QueueModel struct {
	playback PlaybackService
	width    int
	height   int
	cursor   int
}

func NewQueueModel(pl PlaybackService) QueueModel {
	return QueueModel{
		playback: pl,
	}
}

func NewQueueModelWithService(pl PlaybackService) QueueModel {
	return QueueModel{playback: pl}
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
			queue := m.playback.Queue()
			if queue != nil && m.cursor < len(queue)-1 {
				m.cursor++
			}
		case "enter", "p":
			queue := m.playback.Queue()
			if queue != nil && m.cursor < len(queue) {
				_ = m.playback.ToggleQueueTrack(queue, m.cursor)
			}
		}
	}

	return m, nil
}

func (m QueueModel) View() string {
	w, h := normalizeViewSize(m.width, m.height)

	queue := m.playback.Queue()
	current := m.playback.CurrentIndex()

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
	cols := computeTrackColumns(innerW)
	header := trackTableHeader(cols)
	lines = append(lines, header)

	for i := start; i < end; i++ {
		t := queue[i]
		num := fmt.Sprintf("%02d", i+1)
		name := truncateStr(t.Title, cols.nameW)
		artist := truncateStr(t.Artist, cols.artistW)
		album := truncateStr(t.Album, cols.albumW)
		dur := formatDuration(t.Duration)

		var line string
		if i == current {
			line = retroCurrentStyle.Render(fmt.Sprintf("▶ %s ", num)) +
				retroCurrentStyle.Render(padRight(name, cols.nameW))
			if cols.showArtist {
				line += retroSubtleStyle.Render(" ") + retroSubtleStyle.Render(padRight(artist, cols.artistW))
			}
			if cols.showAlbum {
				line += retroSubtleStyle.Render(" ") + retroSubtleStyle.Render(padRight(album, cols.albumW))
			}
			if cols.showDuration {
				line += retroSubtleStyle.Render(" ") + lipgloss.NewStyle().Foreground(colorGreenSelect).Render(padRight(dur, cols.durationW))
			}
		} else if i == m.cursor {
			line = retroSelectedStyle.Render(fmt.Sprintf("► %s ", num)) +
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
	lines = append(lines, retroSubtleStyle.Render(fmt.Sprintf("  Track %d of %d", m.cursor+1, len(queue))))
	lines = append(lines, keys)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(content)
}
