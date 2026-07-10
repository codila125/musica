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
	"github.com/codila125/musica/internal/models"
	"github.com/codila125/musica/internal/player"
	"github.com/codila125/musica/internal/telemetry"
	"github.com/codila125/musica/internal/tui/views"
)

type API = api.Client

type Tab int

const (
	TabBrowse Tab = iota
	TabSearch
	TabQueue
	TabNowPlaying
)

type Model struct {
	apiClient        api.Client
	player           *player.Player
	playback         *app.PlaybackController
	servers          []config.ServerConfig
	currentServer    int
	status           string
	position         int
	duration         int
	positionAt       time.Time
	progressErr      error
	barFrame         int
	shineOffset      int
	lastProgressPoll time.Time
	lastScrobbledID  string
	tabs             []string
	activeTab        Tab
	views            viewAdapter
	width            int
	height           int
	blinkOn          bool
	state            appState
	coordinator      switchCoordinator
	helpVisible      bool
}

// headerHeight is the rows above the tab bar: 1 frame border + 5 header
// lines drawn by renderHeader.
const headerHeight = 6

// tabBarHeight covers the bordered tab buttons (3 rows).
const tabBarHeight = 3

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
		tabs:          []string{"BROWSE", "SEARCH", "QUEUE", "PLAYING"},
		views:         newViewAdapter(client, playback),
		state:         stateBooting,
		coordinator:   app.NewCoordinator(servers, nil),
	}
}

func (m Model) Init() tea.Cmd {
	m = m.withState(stateLoading)
	return tea.Batch(m.views.Init(), uiTickCmd(tickIntervalFor(m.playback.State())))
}

func uiTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return uiTickMsg(t)
	})
}

// tickIntervalFor keeps animations smooth while playing and drops the
// redraw rate ~5x when idle so an open-but-unused player stays cheap.
func tickIntervalFor(state models.PlayerState) time.Duration {
	if state == models.StatePlaying {
		return 80 * time.Millisecond
	}
	return 400 * time.Millisecond
}

// shouldPollProgress throttles mpv IPC to ~1/s; the footer interpolates
// between polls via positionAt, so faster polling buys nothing.
func shouldPollProgress(state models.PlayerState, lastPoll, now time.Time) bool {
	if state != models.StatePlaying && state != models.StatePaused {
		return false
	}
	return lastPoll.IsZero() || now.Sub(lastPoll) >= time.Second
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
		now := time.Time(msg)
		// Derive blink from the wall clock so it keeps a steady ~1s cadence
		// regardless of the current tick interval.
		m.blinkOn = now.UnixMilli()/960%2 == 0
		state := m.playback.State()
		if state == models.StatePlaying {
			m.barFrame = (m.barFrame + 1) % 64
			m.shineOffset = (m.shineOffset + 1) % 120
		}
		if shouldPollProgress(state, m.lastProgressPoll, now) {
			m.lastProgressPoll = now
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
		}
		// Feed position into child views (NOW PLAYING lyrics sync) without
		// them polling mpv on their own.
		progressCmd := m.views.UpdateAll(views.ProgressMsg{
			PositionMs: m.displayPosition() * 1000,
			DurationS:  m.duration,
		})
		// Scrobble on track change here so every start path (manual play,
		// next/previous, auto-advance at track end) is covered by one spot.
		if track := m.playback.CurrentTrack(); track != nil && track.ID != m.lastScrobbledID {
			m.lastScrobbledID = track.ID
			go func(client api.Client, id string) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := client.Scrobble(ctx, id); err != nil {
					logger.Get().Debug("scrobble %s failed: %v", id, err)
				}
			}(m.apiClient, track.ID)
		}
		return m, tea.Batch(uiTickCmd(tickIntervalFor(state)), progressCmd)

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
				if m.activeTab == TabSearch && m.views.SearchIsInInputMode() {
					return m, nil
				}
				r := 'k'
				if msg.Button == tea.MouseButtonWheelDown {
					r = 'j'
				}
				return m, m.views.UpdateActive(m.activeTab, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			case tea.MouseButtonLeft:
				if tab, ok := m.tabAt(msg.X, msg.Y); ok {
					m.activeTab = tab
				}
				return m, nil
			}
		}
		return m, nil

	case tea.KeyMsg:
		searchTyping := m.activeTab == TabSearch && m.views.SearchIsInInputMode()
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys(",", "."))) && !searchTyping:
			delta := 10
			if msg.String() == "," {
				delta = -10
			}
			if err := m.playback.SeekBy(delta); err == nil {
				// Reflect the jump immediately instead of waiting for the poll.
				m.lastProgressPoll = time.Time{}
			}
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("-", "=", "+"))) && !searchTyping:
			delta := 5
			if msg.String() == "-" {
				delta = -5
			}
			v := m.playback.VolumeBy(delta)
			m.status = fmt.Sprintf("Volume %d%%", v)
			return m, nil
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

// tabAt maps a click position to a tab, mirroring renderTabBar geometry:
// equal-width bordered buttons centered within the frame.
func (m Model) tabAt(x, y int) (Tab, bool) {
	if y < headerHeight || y >= headerHeight+tabBarHeight {
		return 0, false
	}
	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - mainFrameStyle.GetHorizontalFrameSize()
	tabCount := len(m.tabs)
	tabW := (innerW - 6) / tabCount
	if tabW < 12 {
		tabW = 12
	}
	cellW := tabW + 2 // rounded border adds one column per side
	totalW := tabCount * cellW
	startX := 1 + (innerW-totalW)/2
	if x < startX || x >= startX+totalW {
		return 0, false
	}
	return Tab((x - startX) / cellW), true
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
