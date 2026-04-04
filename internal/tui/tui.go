package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/api"
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
	apiClient api.Client
	player    *player.Player
	tabs      []string
	activeTab Tab
	browse    views.BrowseModel
	search    views.SearchModel
	queue     views.QueueModel
	width     int
	height    int
}

var (
	tabStyle       = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("240"))
	activeTabStyle = lipgloss.NewStyle().Padding(0, 2).Foreground(lipgloss.Color("255")).Bold(true)
)

func NewModel(client api.Client, pl *player.Player) Model {
	return Model{
		apiClient: client,
		player:    pl,
		tabs:      []string{"Browse", "Search", "Queue"},
		browse:    views.NewBrowseModel(client, pl),
		search:    views.NewSearchModel(client, pl),
		queue:     views.NewQueueModel(pl),
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			m.activeTab = (m.activeTab + 1) % Tab(len(m.tabs))
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			m.activeTab = (m.activeTab - 1 + Tab(len(m.tabs))) % Tab(len(m.tabs))
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q"))):
			return m, tea.Quit
		}
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
	return tabBar + "\n" + content
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
