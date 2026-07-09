package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/app"
	"github.com/codila125/musica/internal/config"
	"github.com/codila125/musica/internal/logger"
	"github.com/codila125/musica/internal/player"
	"github.com/codila125/musica/internal/telemetry"
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
	position      int
	duration      int
	positionAt    time.Time
	progressErr   error
	barFrame      int
	blinkCounter  int
	shineOffset   int
	tabs          []string
	activeTab     Tab
	views         viewAdapter
	width         int
	height        int
	blinkOn       bool
	state         appState
	coordinator   switchCoordinator
	helpVisible   bool
}

type uiTickMsg time.Time

type switchServerMsg struct {
	traceID string
	started time.Time
	client  api.Client
	index   int
	err     error
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
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
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
		m.blinkCounter++
		if m.blinkCounter%12 == 0 {
			m.blinkOn = !m.blinkOn
		}
		m.barFrame = (m.barFrame + 1) % 64
		m.shineOffset = (m.shineOffset + 1) % 120
		pos, posErr := m.playback.Position()
		dur, durErr := m.playback.Duration()
		m.progressErr = nil
		if posErr != nil {
			m.progressErr = posErr
		} else {
			m.position = pos
			m.positionAt = time.Now()
		}
		if durErr != nil {
			m.progressErr = durErr
		} else {
			m.duration = dur
		}
		return m, uiTickCmd()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))):
			if m.activeTab == TabSearch && m.views.SearchIsInInputMode() {
				break
			}
			if m.state == stateSwitchingServer {
				telemetry.Count("switch.ignored.in_progress")
				return m, nil
			}
			if next, ok := m.coordinator.NextIndex(m.currentServer); ok {
				m.status = fmt.Sprintf("Switching to %s...", m.servers[next].Name)
				telemetry.Count("switch.requested")
				telemetry.Event("switch.start",
					telemetry.Field{Key: "from", Value: m.currentServer},
					telemetry.Field{Key: "to", Value: next},
				)
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
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+h"))):
			m.helpVisible = !m.helpVisible
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "ctrl+q"))):
			return m, tea.Quit
		}

	case switchServerMsg:
		durationMs := time.Since(msg.started).Milliseconds()
		m = m.withState(stateLoading)
		if msg.err != nil {
			m = m.withState(stateError)
			telemetry.Count("switch.failed")
			telemetry.Event("switch.complete",
				telemetry.Field{Key: "trace_id", Value: msg.traceID},
				telemetry.Field{Key: "to", Value: msg.index},
				telemetry.Field{Key: "ok", Value: false},
				telemetry.Field{Key: "err_kind", Value: api.KindOf(msg.err)},
				telemetry.Field{Key: "duration_ms", Value: durationMs},
			)
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
		telemetry.Count("switch.success")
		telemetry.Event("switch.complete",
			telemetry.Field{Key: "trace_id", Value: msg.traceID},
			telemetry.Field{Key: "to", Value: msg.index},
			telemetry.Field{Key: "ok", Value: true},
			telemetry.Field{Key: "duration_ms", Value: durationMs},
		)
		return m, m.views.Init()
	}

	// Route async/background messages to all child views so data loads
	// and updates aren't dropped when a tab isn't currently active.
	if _, ok := msg.(tea.KeyMsg); !ok {
		return m, m.views.UpdateAll(msg)
	}

	return m, m.views.UpdateActive(m.activeTab, msg)
}

func (m Model) withState(next appState) Model {
	if m.state == next {
		return m
	}

	if !canTransition(m.state, next) {
		logger.Get().Error("invalid state transition: %s -> %s", m.state.String(), next.String())
		telemetry.Count("state.invalid_transition")
		telemetry.Event("state.transition.invalid",
			telemetry.Field{Key: "from", Value: m.state.String()},
			telemetry.Field{Key: "to", Value: next.String()},
		)
		return m
	}

	logger.Get().Debug("state transition: %s -> %s", m.state.String(), next.String())
	telemetry.Count("state.transition")
	telemetry.Event("state.transition",
		telemetry.Field{Key: "from", Value: m.state.String()},
		telemetry.Field{Key: "to", Value: next.String()},
	)
	m.state = next
	return m
}

func (m Model) switchServerCmd(index int) tea.Cmd {
	traceID := fmt.Sprintf("sw-%d-%d", time.Now().UnixNano(), index)
	start := time.Now()
	return func() tea.Msg {
		endConnect := telemetry.Timed("switch.connect",
			telemetry.Field{Key: "trace_id", Value: traceID},
			telemetry.Field{Key: "to", Value: index},
		)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res := m.coordinator.ConnectIndex(ctx, index)
		if res.Err != nil {
			telemetry.Count("switch.connect.failed")
			endConnect(
				telemetry.Field{Key: "ok", Value: false},
				telemetry.Field{Key: "err_kind", Value: api.KindOf(res.Err)},
			)
		} else {
			telemetry.Count("switch.connect.success")
			endConnect(telemetry.Field{Key: "ok", Value: true})
		}
		return switchServerMsg{traceID: traceID, started: start, client: res.Client, index: res.Index, err: res.Err}
	}
}
