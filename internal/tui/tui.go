package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/api/jellyfin"
	"github.com/codila125/musica/internal/api/navidrome"
	"github.com/codila125/musica/internal/config"
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
}

var (
	tabStyle       = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("240"))
	activeTabStyle = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("255")).Bold(true)
)

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
	return tea.Batch(m.browse.Init(), m.search.Init(), m.queue.Init())
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))):
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
	footer := ""
	if len(m.servers) > 0 {
		footer = fmt.Sprintf("\n[%s] tab: switch views, s: switch server, q: quit", m.servers[m.currentServer].Name)
		if m.status != "" {
			footer += "\n" + m.status
		}
	}
	return tabBar + "\n" + content + footer
}

func (m Model) renderTabBar() string {
	tabs := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		if Tab(i) == m.activeTab {
			tabs[i] = activeTabStyle.Render("> " + t)
		} else {
			tabs[i] = tabStyle.Render(t)
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
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
