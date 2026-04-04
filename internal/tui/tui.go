package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/app"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/logger"
	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
)

type API = api.Client

type Tab int

const (
	TabBrowse Tab = iota
	TabSearch
	TabQueue
)

type Model struct {
	apiClient     api.Client
	player        *player.Player
	playback      *app.PlaybackController
	servers       []config.ServerConfig
	currentServer int
	status        string
	tabs          []string
	activeTab     Tab
	views         viewAdapter
	width         int
	height        int
	blinkOn       bool
	state         appState
	coordinator   switchCoordinator
}

// Colors
var (
	colorPurple    = lipgloss.Color("93")
	colorLightText = lipgloss.Color("230")
	colorDimText   = lipgloss.Color("244")
	colorYellow    = lipgloss.Color("226")
	colorRed       = lipgloss.Color("196")
	colorRedDim    = lipgloss.Color("88")
	colorGreen     = lipgloss.Color("46")
	colorCyan      = lipgloss.Color("51")
	colorAmber     = lipgloss.Color("214")
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
				Foreground(colorLightText).
				Align(lipgloss.Center).
				Padding(0, 1).
				Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorDimText)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	nowPlayingStyle = lipgloss.NewStyle().
			Foreground(colorCyan)
)

type uiTickMsg time.Time

type switchServerMsg struct {
	client api.Client
	index  int
	err    error
}

type switchCoordinator interface {
	NextIndex(current int) (int, bool)
	ConnectIndex(ctx context.Context, index int) app.SwitchResult
}

func NewModel(client api.Client, pl *player.Player, servers []config.ServerConfig, currentServer int) Model {
	playback := app.NewPlaybackController(pl)

	return Model{
		apiClient:     client,
		player:        pl,
		playback:      playback,
		servers:       servers,
		currentServer: currentServer,
		tabs:          []string{"BROWSE", "SEARCH", "QUEUE"},
		views:         newViewAdapter(client, playback),
		state:         stateBooting,
		coordinator:   app.NewCoordinator(servers, nil),
	}
}

func (m Model) Init() tea.Cmd {
	m = m.withState(stateLoading)
	return tea.Batch(m.views.Init(), uiTickCmd())
}

func uiTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return uiTickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		childW, childH := m.childViewportDims(msg.Width, msg.Height)
		childSize := tea.WindowSizeMsg{Width: childW, Height: childH}
		m.views.Resize(childSize)
		return m, nil

	case uiTickMsg:
		m.blinkOn = !m.blinkOn
		return m, uiTickCmd()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			if m.state == stateSwitchingServer {
				return m, nil
			}
			if next, ok := m.coordinator.NextIndex(m.currentServer); ok {
				m.status = fmt.Sprintf("Switching to %s...", m.servers[next].Name)
				m = m.withState(stateSwitchingServer)
				return m, m.switchServerCmd(next)
			}
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			m.activeTab = (m.activeTab + 1) % Tab(len(m.tabs))
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			m.activeTab = (m.activeTab - 1 + Tab(len(m.tabs))) % Tab(len(m.tabs))
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "ctrl+q"))):
			return m, tea.Quit
		}

	case switchServerMsg:
		m = m.withState(stateLoading)
		if msg.err != nil {
			m = m.withState(stateError)
			switch api.KindOf(msg.err) {
			case api.ErrorKindAuth:
				m.status = "Server switch failed: authentication error"
			case api.ErrorKindNetwork:
				m.status = "Server switch failed: network error"
			case api.ErrorKindConfig:
				m.status = "Server switch failed: configuration error"
			default:
				m.status = "Server switch failed: " + msg.err.Error()
			}
			return m, nil
		}
		// Keep one player instance to avoid race conditions while switching.
		// Reset playback/queue state before attaching new server data.
		_ = m.playback.Stop()

		// Cancel in-flight async requests from previous server context.
		m.views.CancelInFlight()

		m.apiClient = msg.client
		if msg.index >= 0 && msg.index < len(m.servers) {
			m.currentServer = msg.index
			m.status = "Connected to " + m.servers[msg.index].Name
		} else {
			m.currentServer = 0
			m.status = "Connected"
		}
		m.views = newViewAdapter(m.apiClient, m.playback)

		// Keep layout consistent after server switch by reapplying current size.
		childW, childH := m.childViewportDims(m.width, m.height)
		childSize := tea.WindowSizeMsg{Width: childW, Height: childH}
		m.views.Resize(childSize)

		m = m.withState(stateReady)
		return m, m.views.Init()
	}

	// Route async/background messages to all child views so data loads
	// and updates aren't dropped when a tab isn't currently active.
	if _, ok := msg.(tea.KeyMsg); !ok {
		return m, m.views.UpdateAll(msg)
	}

	return m, m.views.UpdateActive(m.activeTab, msg)
}

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
	frame := mainFrameStyle.Copy().
		Width(w).
		Height(h)

	return frame.Render(body)
}

func (m Model) renderHeader(w int) string {
	title := headerStyle.Render("╔══════════════════════════════════════════╗")
	subtitle := headerStyle.Render("║      MUSICA  ::  RETRO CASSETTE DECK     ║")
	bottom := headerStyle.Render("╚══════════════════════════════════════════╝")

	header := lipgloss.JoinVertical(lipgloss.Center, title, subtitle, bottom)
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
			style = tabButtonActiveStyle.Copy().Width(tabW)
			if m.blinkOn {
				style = style.BorderForeground(colorRed).Foreground(colorRed)
				label = "● " + name
			} else {
				style = style.BorderForeground(colorRedDim).Foreground(colorAmber)
				label = "○ " + name
			}
		} else {
			// Inactive tab is static
			style = tabButtonStyle.Copy().Width(tabW)
			label = "  " + name
		}

		tabs[i] = style.Render(label)
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Padding(0, 0, 1, 0).Render(tabBar)
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
	childH := totalH - mainFrameStyle.GetVerticalFrameSize() - 10
	if childW < 20 {
		childW = 20
	}
	if childH < 8 {
		childH = 8
	}
	return childW, childH
}

func (m Model) withState(next appState) Model {
	if m.state == next {
		return m
	}

	if !canTransition(m.state, next) {
		logger.Get().Error("invalid state transition: %s -> %s", m.state.String(), next.String())
		return m
	}

	logger.Get().Debug("state transition: %s -> %s", m.state.String(), next.String())
	m.state = next
	return m
}

func (m Model) renderFooter(w int) string {
	line := lipgloss.NewStyle().Foreground(colorPurple).Render(strings.Repeat("═", w-2))

	// Server info
	serverInfo := ""
	if len(m.servers) > 0 {
		serverInfo = statusStyle.Render("◉ " + m.servers[m.currentServer].Name)
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
		nowPlaying = nowPlayingStyle.Render(fmt.Sprintf("%s %s - %s", stateIcon, track.Title, track.Artist))
	}

	// Key hints
	hints := footerStyle.Render("[tab]switch [s]server [ctrl+q]quit")

	// Status line
	statusLine := ""
	if m.status != "" {
		statusLine = footerStyle.Render(m.status)
	}

	col1 := lipgloss.NewStyle().Width(w / 3).Align(lipgloss.Left).Render(serverInfo)
	col2 := lipgloss.NewStyle().Width(w / 3).Align(lipgloss.Center).Render(nowPlaying)
	col3 := lipgloss.NewStyle().Width(w / 3).Align(lipgloss.Right).Render(hints)
	infoLine := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	if statusLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left, line, infoLine, statusLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, line, infoLine)
}

func (m Model) switchServerCmd(index int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res := m.coordinator.ConnectIndex(ctx, index)
		return switchServerMsg{client: res.Client, index: res.Index, err: res.Err}
	}
}
