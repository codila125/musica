package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/tui/views"
)

// Colors
var (
	colorPurple    = lipgloss.Color("130")
	colorLightText = lipgloss.Color("230")
	colorDimText   = lipgloss.Color("244")
	colorYellow    = lipgloss.Color("220")
	colorRed       = lipgloss.Color("196")
	colorRedDim    = lipgloss.Color("88")
	colorGreen     = lipgloss.Color("46")
	colorCyan      = lipgloss.Color("51")
	colorAmber     = lipgloss.Color("214")
	colorBrown     = lipgloss.Color("136")
)

// Styles
var (
	mainFrameStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorPurple).
			Foreground(colorLightText)

	tabButtonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorDimText).
			Foreground(colorDimText).
			Align(lipgloss.Center).
			Padding(0, 1)

	tabButtonActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorRed).
				Foreground(colorAmber).
				Align(lipgloss.Center).
				Padding(0, 1).
				Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorDimText)

	helpTitleStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	nowPlayingStyle = lipgloss.NewStyle().
			Foreground(colorCyan)
)

func (m Model) View() string {
	w := m.width
	h := m.height
	if w < 60 {
		w = 60
	}
	if h < 20 {
		h = 20
	}

	// Build layout parts
	innerW := w - mainFrameStyle.GetHorizontalFrameSize()
	childW, childH := m.childViewportDims(w, h)

	header := m.renderHeader(innerW)
	tabBar := m.renderTabBar(innerW)
	content := m.renderContent(innerW, childW, childH)
	footer := m.renderFooter(innerW)

	// Join vertically
	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabBar,
		content,
		footer,
	)

	// Apply main frame to fill terminal
	frame := mainFrameStyle.
		Width(w).
		Height(h)

	output := frame.Render(body)

	if m.helpVisible {
		help := m.renderHelp(w, h)
		output = lipgloss.JoinVertical(lipgloss.Left, output, help)
	}

	return output
}

func (m Model) renderHeader(w int) string {
	innerW := w
	if innerW < 38 {
		innerW = 38
	}
	contentW := innerW - 2
	label := trimLabel(" MUSICA  ::  RETRO CASSETTE DECK ", contentW)
	led := "◉"
	if !m.blinkOn {
		led = "◌"
	}
	labelLine := padCenter(""+led+label+led, contentW)
	shineLine := padCenter(m.renderDeckShine(contentW), contentW)
	controlLine := padCenter("[PLAY] [PAUSE] [REW] [FF]", contentW)

	top := "╔" + strings.Repeat("═", contentW) + "╗"
	mid := "║" + labelLine + "║"
	ctl := "║" + controlLine + "║"
	shn := "║" + shineLine + "║"
	bot := "╚" + strings.Repeat("═", contentW) + "╝"

	header := lipgloss.JoinVertical(lipgloss.Center,
		headerStyle.Render(top),
		headerStyle.Render(mid),
		footerStyle.Render(ctl),
		headerStyle.Render(shn),
		headerStyle.Render(bot),
	)
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(header)
}

func (m Model) renderTabBar(w int) string {
	tabCount := len(m.tabs)
	tabW := (w - 6) / tabCount
	if tabW < 12 {
		tabW = 12
	}

	tabs := make([]string, tabCount)
	for i, name := range m.tabs {
		var style lipgloss.Style
		var label string

		if Tab(i) == m.activeTab {
			// Active tab with blinking red LED
			style = tabButtonActiveStyle.Width(tabW)
			if m.blinkOn {
				style = style.BorderForeground(colorRed).Foreground(colorRed)
				label = "● " + name
			} else {
				style = style.BorderForeground(colorRedDim).Foreground(colorAmber)
				label = "○ " + name
			}
		} else {
			// Inactive tab is static
			style = tabButtonStyle.Width(tabW)
			label = "  " + name
		}

		tabs[i] = style.Render(label)
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Padding(0, 0, 1, 0).Render(tabBar)
}

func (m Model) renderDeckShine(w int) string {
	if w <= 0 {
		return ""
	}
	base := []rune(strings.Repeat("─", w))
	pos := m.shineOffset % (w + 10)
	var b strings.Builder
	for i := 0; i < w; i++ {
		ch := base[i]
		d := i - pos
		if d >= -2 && d <= 2 {
			ch = '═'
		}
		b.WriteRune(ch)
	}
	return lipgloss.NewStyle().Foreground(colorAmber).Render(b.String())
}

func (m Model) renderContent(w, childW, childH int) string {
	content := m.views.View(m.activeTab)

	return lipgloss.NewStyle().
		Width(w).
		Height(childH).
		Align(lipgloss.Center, lipgloss.Top).
		Render(content)
}

func (m Model) childViewportDims(totalW, totalH int) (int, int) {
	childW := totalW - mainFrameStyle.GetHorizontalFrameSize() - 2
	childH := totalH - mainFrameStyle.GetVerticalFrameSize() - 12
	if childW < 20 {
		childW = 20
	}
	if childH < 8 {
		childH = 8
	}
	return childW, childH
}

func (m Model) renderFooter(w int) string {
	line := lipgloss.NewStyle().Foreground(colorPurple).Render(strings.Repeat("═", w-2))

	// Server info
	serverInfo := ""
	if len(m.servers) > 0 {
		ledStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
		nameStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
		offStyle := lipgloss.NewStyle().Foreground(colorDimText)
		var led, name string
		if m.blinkOn {
			led = ledStyle.Render("◉")
			name = nameStyle.Render(m.servers[m.currentServer].Name)
		} else {
			led = offStyle.Render("◉")
			name = offStyle.Render(m.servers[m.currentServer].Name)
		}
		serverInfo = led + name
	}

	// Now playing
	nowPlaying := footerStyle.Render("No track playing")
	if track := m.playback.CurrentTrack(); track != nil {
		stateIcon := "■"
		switch m.playback.State() {
		case models.StatePlaying:
			stateIcon = "▶"
		case models.StatePaused:
			stateIcon = "❚❚"
		}
		progress := ""
		if m.playback.State() == models.StatePlaying || m.playback.State() == models.StatePaused {
			if m.progressErr == nil {
				displayPos := m.position
				if m.playback.State() == models.StatePlaying && !m.positionAt.IsZero() {
					displayPos += int(time.Since(m.positionAt).Seconds())
				}
				if m.duration > 0 && displayPos > m.duration {
					displayPos = m.duration
				}
				if m.duration > 0 {
					progress = fmt.Sprintf(" %s/%s", views.FormatDuration(displayPos), views.FormatDuration(m.duration))
				} else {
					progress = fmt.Sprintf(" %s", views.FormatDuration(displayPos))
				}
			}
		}
		nowPlaying = nowPlayingStyle.Render(fmt.Sprintf("%s %s - %s%s", stateIcon, track.Title, track.Artist, progress))
	}

	// Key hints
	hints := footerStyle.Render("[ctrl+h]help [tab]switch [ctrl+s]server [ctrl+q]quit")

	// Status line
	statusLine := ""
	if m.status != "" {
		statusLine = footerStyle.Render(m.status)
	}

	col1 := lipgloss.NewStyle().Width(w / 3).Align(lipgloss.Left).Render(serverInfo)
	col2 := lipgloss.NewStyle().Width(w / 3).Align(lipgloss.Center).Render(nowPlaying)
	col3 := lipgloss.NewStyle().Width(w / 3).Align(lipgloss.Right).Render(hints)
	infoLine := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	lines := []string{line, infoLine}
	if track := m.playback.CurrentTrack(); track != nil {
		if m.playback.State() == models.StatePlaying {
			bars := m.renderSoundBars(w)
			lines = append(lines, bars)
		}
	}
	lines = append(lines, m.renderTapeWindow(w))
	if statusLine != "" {
		lines = append(lines, statusLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderHelp(w, h int) string {
	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAmber).
		Width(w-6).
		Padding(1, 2)

	helpSectionStyle := lipgloss.NewStyle().
		Foreground(colorAmber).
		Bold(true)

	lines := []string{
		helpTitleStyle.Render(" ◎ KEYBOARD SHORTCUTS "),
		"",
		footerStyle.Render("  [ctrl+h]     Toggle this help"),
		footerStyle.Render("  [tab]        Switch between tabs"),
		footerStyle.Render("  [shift+tab]  Previous tab"),
		footerStyle.Render("  [ctrl+s]     Switch server"),
		footerStyle.Render("  [ctrl+r]     Refresh (Browse tab)"),
		footerStyle.Render("  [r]          Replay current track"),
		footerStyle.Render("  [n]          Next track"),
		footerStyle.Render("  [m]          Previous track"),
		footerStyle.Render("  [ctrl+q]     Quit application"),
		"",
		helpTitleStyle.Render(" ◎ TAB SPECIFIC "),
		"",
	}

	switch m.activeTab {
	case TabBrowse:
		lines = append(lines,
			helpSectionStyle.Render("  ─── TRACK LIBRARY ───"),
			footerStyle.Render("  [j/k] or [↓/↑]  Navigate up/down"),
			footerStyle.Render("  [←/→]        Previous/next page"),
			footerStyle.Render("  [p] or [enter]  Play/pause track"),
			footerStyle.Render("  [q]            Add to queue"),
			footerStyle.Render("  [ctrl+r]       Refresh tracks"),
		)
	case TabSearch:
		lines = append(lines,
			helpSectionStyle.Render("  ─── SEARCH DECK ───"),
			footerStyle.Render("  [type]           Enter search query"),
			footerStyle.Render("  [enter]          Start search"),
			footerStyle.Render("  [esc]            Back to input"),
			"",
			helpSectionStyle.Render("  ─── SEARCH RESULTS ───"),
			footerStyle.Render("  [j/k] or [↓/↑]  Navigate up/down"),
			footerStyle.Render("  [p] or [enter]  Play track"),
			footerStyle.Render("  [q]            Add to queue"),
			footerStyle.Render("  [←/→] or [l/r]  Switch category (Tracks/Albums/Artists)"),
		)
	case TabQueue:
		lines = append(lines,
			helpSectionStyle.Render("  ─── TAPE QUEUE ───"),
			footerStyle.Render("  [j/k] or [↓/↑]  Navigate up/down"),
			footerStyle.Render("  [p] or [enter]  Play/pause track"),
		)
	}

	lines = append(lines, "", footerStyle.Render("  Press [ctrl+h] to close"))

	return helpBox.Render(strings.Join(lines, "\n"))
}

func (m Model) renderSoundBars(w int) string {
	const (
		barCount  = 12
		maxHeight = 7
	)
	phaseBase := float64(m.barFrame) * 0.18
	var bars strings.Builder
	for i := 0; i < barCount; i++ {
		if i > 0 {
			bars.WriteByte(' ')
		}
		phase := phaseBase + float64(i)*0.55
		height := int((math.Sin(phase)+1.0)/2.0*float64(maxHeight-1)) + 1
		bars.WriteString(strings.Repeat("▌", height))
	}
	barStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(barStyle.Render(bars.String()))
}

func (m Model) renderTapeWindow(w int) string {
	if w <= 0 {
		return ""
	}
	inner := w - 12
	if inner < 20 {
		inner = w
	}
	leftReel := "◐"
	rightReel := "◑"
	if m.playback.State() == models.StatePlaying {
		if m.blinkOn {
			leftReel = "◓"
			rightReel = "◒"
		}
	}
	spoolLen := inner/2 - 4
	if spoolLen < 2 {
		spoolLen = 2
	}
	spool := strings.Repeat("━", spoolLen)
	window := fmt.Sprintf("%s%s%s%s", leftReel, spool, rightReel, spool)
	windowStyle := lipgloss.NewStyle().Foreground(colorBrown)
	frame := lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(windowStyle.Render(window))
	return frame
}

func trimLabel(label string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(label) <= max {
		return label
	}
	if max <= 3 {
		return label[:max]
	}
	return label[:max-3] + "..."
}

func padCenter(s string, w int) string {
	if len(s) >= w {
		return s
	}
	pad := w - len(s)
	left := pad / 2
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
