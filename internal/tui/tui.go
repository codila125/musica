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
	"github.com/codila125/musica/internal/api/jellyfin"
	"github.com/codila125/musica/internal/api/navidrome"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
	"github.com/codila125/musica/internal/tui/views"
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
	servers       []config.ServerConfig
	currentServer int
	status        string
	tabs          []string
	activeTab     Tab
	browse        views.BrowseModel
	search        views.SearchModel
	queue         views.QueueModel
	width         int
	height        int
	blinkOn       bool
	state         appState
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

func NewModel(client api.Client, pl *player.Player, servers []config.ServerConfig, currentServer int) Model {
	return Model{
		apiClient:     client,
		player:        pl,
		servers:       servers,
		currentServer: currentServer,
		tabs:          []string{"BROWSE", "SEARCH", "QUEUE"},
		browse:        views.NewBrowseModel(client, pl),
		search:        views.NewSearchModel(client, pl),
		queue:         views.NewQueueModel(pl),
		state:         stateBooting,
	}
}

func (m Model) Init() tea.Cmd {
	m.state = stateLoading
	return tea.Batch(m.browse.Init(), m.search.Init(), m.queue.Init(), uiTickCmd())
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
		m.browse, _ = m.browse.Update(childSize)
		m.search, _ = m.search.Update(childSize)
		m.queue, _ = m.queue.Update(childSize)
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
			if len(m.servers) > 1 {
				next := (m.currentServer + 1) % len(m.servers)
				m.status = fmt.Sprintf("Switching to %s...", m.servers[next].Name)
				m.state = stateSwitchingServer
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
		m.state = stateLoading
		if msg.err != nil {
			m.state = stateError
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
		_ = m.player.Stop()

		// Cancel in-flight async requests from previous server context.
		m.browse, _ = m.browse.Update(views.CancelInFlightCmd())
		m.search, _ = m.search.Update(views.CancelInFlightCmd())

		m.apiClient = msg.client
		if msg.index >= 0 && msg.index < len(m.servers) {
			m.currentServer = msg.index
			m.status = "Connected to " + m.servers[msg.index].Name
		} else {
			m.currentServer = 0
			m.status = "Connected"
		}
		m.browse = views.NewBrowseModel(m.apiClient, m.player)
		m.search = views.NewSearchModel(m.apiClient, m.player)
		m.queue = views.NewQueueModel(m.player)

		// Keep layout consistent after server switch by reapplying current size.
		childW, childH := m.childViewportDims(m.width, m.height)
		childSize := tea.WindowSizeMsg{Width: childW, Height: childH}
		m.browse, _ = m.browse.Update(childSize)
		m.search, _ = m.search.Update(childSize)
		m.queue, _ = m.queue.Update(childSize)

		m.state = stateReady
		return m, tea.Batch(m.browse.Init(), m.search.Init(), m.queue.Init())
	}

	// Route async/background messages to all child views so data loads
	// and updates aren't dropped when a tab isn't currently active.
	if _, ok := msg.(tea.KeyMsg); !ok {
		var cmdBrowse, cmdSearch, cmdQueue tea.Cmd
		m.browse, cmdBrowse = m.browse.Update(msg)
		m.search, cmdSearch = m.search.Update(msg)
		m.queue, cmdQueue = m.queue.Update(msg)
		return m, tea.Batch(cmdBrowse, cmdSearch, cmdQueue)
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case TabBrowse:
		m.browse, cmd = m.browse.Update(msg)
	case TabSearch:
		m.search, cmd = m.search.Update(msg)
	case TabQueue:
		m.queue, cmd = m.queue.Update(msg)
	}

	return m, cmd
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
	var content string
	switch m.activeTab {
	case TabBrowse:
		content = m.browse.View()
	case TabSearch:
		content = m.search.View()
	case TabQueue:
		content = m.queue.View()
	}

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

func (m Model) renderFooter(w int) string {
	line := lipgloss.NewStyle().Foreground(colorPurple).Render(strings.Repeat("═", w-2))

	// Server info
	serverInfo := ""
	if len(m.servers) > 0 {
		serverInfo = statusStyle.Render("◉ " + m.servers[m.currentServer].Name)
	}

	// Now playing
	nowPlaying := footerStyle.Render("No track playing")
	if track := m.player.CurrentTrack(); track != nil {
		stateIcon := "■"
		switch m.player.State() {
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

		if index < 0 || index >= len(m.servers) {
			return switchServerMsg{err: fmt.Errorf("invalid server index")}
		}

		serverCfg := m.servers[index]
		client, err := connectServer(ctx, serverCfg)
		if err != nil {
			return switchServerMsg{err: err}
		}

		return switchServerMsg{client: client, index: index}
	}
}

func connectServer(ctx context.Context, serverCfg config.ServerConfig) (api.Client, error) {
	switch serverCfg.Type {
	case "navidrome":
		c := navidrome.New(serverCfg)
		if err := c.Ping(ctx); err != nil {
			return nil, api.Wrap(api.ErrorKindNetwork, "navidrome.ping", err)
		}
		return c, nil
	case "jellyfin":
		c := jellyfin.New(serverCfg)
		if err := c.Ping(ctx); err != nil {
			return nil, api.Wrap(api.ErrorKindNetwork, "jellyfin.ping", err)
		}
		if err := c.Authenticate(ctx, serverCfg.Username, serverCfg.Password); err != nil {
			return nil, api.Wrap(api.ErrorKindAuth, "jellyfin.authenticate", err)
		}
		return c, nil
	default:
		return nil, api.Wrap(api.ErrorKindConfig, "connect.type", fmt.Errorf("unknown server type: %s", serverCfg.Type))
	}
}
