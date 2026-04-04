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
}

var (
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("93")).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("0")).
			Padding(0, 0)
	tapeTabRaisedStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("240")).
				Foreground(lipgloss.Color("248")).
				Background(lipgloss.Color("238")).
				Align(lipgloss.Center)
	tapeTabPressedStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("52")).
				Foreground(lipgloss.Color("223")).
				Background(lipgloss.Color("234")).
				Align(lipgloss.Center)
	tapeLedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	nowStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
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
		tabs:          []string{"Browse", "Search", "Queue"},
		browse:        views.NewBrowseModel(client, pl),
		search:        views.NewSearchModel(client, pl),
		queue:         views.NewQueueModel(pl),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.browse.Init(), m.search.Init(), m.queue.Init(), uiTickCmd())
}

func uiTickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return uiTickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.browse, _ = m.browse.Update(msg)
		m.search, _ = m.search.Update(msg)
		m.queue, _ = m.queue.Update(msg)
		return m, nil

	case uiTickMsg:
		m.blinkOn = !m.blinkOn
		return m, uiTickCmd()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
			if len(m.servers) > 1 {
				next := (m.currentServer + 1) % len(m.servers)
				m.status = fmt.Sprintf("Switching to %s...", m.servers[next].Name)
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
		if msg.err != nil {
			m.status = "Server switch failed: " + msg.err.Error()
			return m, nil
		}
		m.apiClient = msg.client
		m.currentServer = msg.index
		m.status = "Connected to " + m.servers[msg.index].Name
		_ = m.player.Stop()
		m.browse = views.NewBrowseModel(m.apiClient, m.player)
		m.search = views.NewSearchModel(m.apiClient, m.player)
		m.queue = views.NewQueueModel(m.player)
		return m, tea.Batch(m.browse.Init(), m.search.Init(), m.queue.Init())
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
	var content string
	switch m.activeTab {
	case TabBrowse:
		content = m.browse.View()
	case TabSearch:
		content = m.search.View()
	case TabQueue:
		content = m.queue.View()
	}

	tabBar := m.renderTabBar()
	footerLines := []string{hintStyle.Render("keys: tab shift+tab switch views | s switch server | ctrl+q quit")}
	if len(m.servers) > 0 {
		footerLines = append(footerLines, statusStyle.Render("source: "+m.servers[m.currentServer].Name))
		if track := m.player.CurrentTrack(); track != nil {
			state := "stopped"
			switch m.player.State() {
			case models.StatePlaying:
				state = "playing"
			case models.StatePaused:
				state = "paused"
			}
			footerLines = append(footerLines, nowStyle.Render(fmt.Sprintf("now spinning: %s - %s | state=%s", track.Title, track.Artist, state)))
		}
		if m.status != "" {
			footerLines = append(footerLines, hintStyle.Render("status: "+m.status))
		}
	}
	body := strings.Join([]string{tabBar, content, strings.Join(footerLines, "\n")}, "\n\n")

	containerStyle := frameStyle.Copy()
	hFrame := containerStyle.GetHorizontalFrameSize()
	vFrame := containerStyle.GetVerticalFrameSize()
	bodyStyle := lipgloss.NewStyle()

	if m.width > 0 {
		containerStyle = containerStyle.Width(m.width)
		innerWidth := m.width - hFrame
		if innerWidth < 1 {
			innerWidth = 1
		}
		bodyStyle = bodyStyle.Width(innerWidth).MaxWidth(innerWidth)
	}
	if m.height > 0 {
		containerStyle = containerStyle.Height(m.height)
		innerHeight := m.height - vFrame
		if innerHeight < 1 {
			innerHeight = 1
		}
		bodyStyle = bodyStyle.Height(innerHeight).MaxHeight(innerHeight)
	}

	body = bodyStyle.Render(body)
	return containerStyle.Render(body)
}

func (m Model) renderTabBar() string {
	header := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("MUSICA :: RETRO DECK")

	totalWidth := 48
	if m.width > 0 {
		totalWidth = m.width - frameStyle.GetHorizontalFrameSize()
	}
	if totalWidth < 18 {
		totalWidth = 18
	}

	tabCount := len(m.tabs)
	baseSegmentWidth := totalWidth / tabCount
	remainder := totalWidth % tabCount
	tabs := make([]string, tabCount)

	for i, t := range m.tabs {
		segmentWidth := baseSegmentWidth
		if i < remainder {
			segmentWidth++
		}
		innerWidth := segmentWidth - 2
		if innerWidth < 1 {
			innerWidth = 1
		}

		label := strings.ToUpper(t)
		if innerWidth <= 7 {
			short := []string{"BRW", "SRCH", "QUE"}
			if i < len(short) {
				label = short[i]
			}
		}
		if innerWidth <= 4 {
			label = string(label[0])
		}

		if Tab(i) == m.activeTab {
			text := "[" + label + "]"
			if innerWidth >= 6 {
				text = tapeLedStyle.Render("●") + " " + "[" + label + "]"
			}
			pressed := tapeTabPressedStyle.Copy().Width(innerWidth)
			if m.blinkOn {
				pressed = pressed.BorderForeground(lipgloss.Color("196")).Foreground(lipgloss.Color("203"))
			} else {
				pressed = pressed.BorderForeground(lipgloss.Color("88")).Foreground(lipgloss.Color("223"))
			}
			tabs[i] = pressed.Render(text)
		} else {
			raised := tapeTabRaisedStyle.Copy().Width(innerWidth)
			if m.blinkOn {
				raised = raised.BorderForeground(lipgloss.Color("160")).Foreground(lipgloss.Color("210"))
			}
			tabs[i] = raised.Render("[" + label + "]")
		}
	}

	return header + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) switchServerCmd(index int) tea.Cmd {
	return func() tea.Msg {
		if index < 0 || index >= len(m.servers) {
			return switchServerMsg{err: fmt.Errorf("invalid server index")}
		}

		serverCfg := m.servers[index]
		client, err := connectServer(serverCfg)
		if err != nil {
			return switchServerMsg{err: err}
		}

		return switchServerMsg{client: client, index: index}
	}
}

func connectServer(serverCfg config.ServerConfig) (api.Client, error) {
	ctx := context.Background()

	switch serverCfg.Type {
	case "navidrome":
		c := navidrome.New(serverCfg)
		if err := c.Ping(ctx); err != nil {
			return nil, err
		}
		return c, nil
	case "jellyfin":
		c := jellyfin.New(serverCfg)
		if err := c.Ping(ctx); err != nil {
			return nil, err
		}
		if err := c.Authenticate(ctx, serverCfg.Username, serverCfg.Password); err != nil {
			return nil, err
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unknown server type: %s", serverCfg.Type)
	}
}
