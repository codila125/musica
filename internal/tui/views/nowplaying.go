package views

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codila125/musica/internal/api"
	"github.com/codila125/musica/internal/models"
)

// ProgressMsg carries the parent's polled playback position so this view
// never issues its own mpv IPC.
type ProgressMsg struct {
	PositionMs int
	DurationS  int
}

type lyricsLoadedMsg struct {
	trackID string
	lyrics  models.Lyrics
}

type coverLoadedMsg struct {
	albumID  string
	rendered string
}

const (
	coverWCells = 26
	coverHCells = 13
)

type NowPlayingModel struct {
	client   api.Client
	playback PlaybackService
	width    int
	height   int

	posMs int
	durS  int

	lyrics    models.Lyrics
	lyricsFor string

	cover      string
	coverFor   string
	coverCache map[string]string
}

func NewNowPlayingModel(client api.Client, pl PlaybackService) NowPlayingModel {
	return NowPlayingModel{
		client:     client,
		playback:   pl,
		coverCache: map[string]string{},
	}
}

func (m NowPlayingModel) Init() tea.Cmd {
	return nil
}

func (m NowPlayingModel) Update(msg tea.Msg) (NowPlayingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			_ = m.playback.Next()
		case "m":
			_ = m.playback.Previous()
		case "r":
			_ = m.playback.Replay()
		case "p", "enter", " ":
			if cur := m.playback.CurrentTrack(); cur != nil {
				_ = m.playback.ToggleTrack(*cur)
			}
		}
		return m, m.refreshTrackCmd()

	case ProgressMsg:
		m.posMs = msg.PositionMs
		m.durS = msg.DurationS
		return m, m.refreshTrackCmd()

	case lyricsLoadedMsg:
		if cur := m.playback.CurrentTrack(); cur != nil && cur.ID == msg.trackID {
			m.lyrics = msg.lyrics
		}

	case coverLoadedMsg:
		m.coverCache[msg.albumID] = msg.rendered
		if cur := m.playback.CurrentTrack(); cur != nil && cur.AlbumID == msg.albumID {
			m.cover = msg.rendered
		}
	}
	return m, nil
}

// refreshTrackCmd kicks off lyrics/cover fetches when the current track
// changes; both are fetched at most once per track/album.
func (m *NowPlayingModel) refreshTrackCmd() tea.Cmd {
	cur := m.playback.CurrentTrack()
	if cur == nil {
		m.lyrics = models.Lyrics{}
		m.lyricsFor = ""
		m.cover = ""
		m.coverFor = ""
		return nil
	}

	var cmds []tea.Cmd
	if cur.ID != m.lyricsFor {
		m.lyricsFor = cur.ID
		m.lyrics = models.Lyrics{}
		track := *cur
		client := m.client
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			lyrics, err := client.GetLyrics(ctx, track)
			if err != nil {
				return lyricsLoadedMsg{trackID: track.ID}
			}
			return lyricsLoadedMsg{trackID: track.ID, lyrics: lyrics}
		})
	}
	if cur.AlbumID != m.coverFor {
		m.coverFor = cur.AlbumID
		if cached, ok := m.coverCache[cur.AlbumID]; ok {
			m.cover = cached
		} else {
			m.cover = ""
			if url := cur.CoverURL; url != "" {
				albumID := cur.AlbumID
				cmds = append(cmds, func() tea.Msg {
					return fetchCover(url, albumID)
				})
			}
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func fetchCover(url, albumID string) tea.Msg {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return coverLoadedMsg{albumID: albumID}
	}
	defer resp.Body.Close()
	// Covers are small; cap the read defensively anyway.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return coverLoadedMsg{albumID: albumID}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return coverLoadedMsg{albumID: albumID}
	}
	return coverLoadedMsg{albumID: albumID, rendered: renderCoverArt(img, coverWCells, coverHCells)}
}

func (m NowPlayingModel) View() string {
	w, h := normalizeViewSize(m.width, m.height)
	boxStyle := listBoxStyle(w, h)
	innerW := w - 8

	title := retroTitleStyle.Render("◎ NOW PLAYING")
	divider := listDivider(innerW)

	cur := m.playback.CurrentTrack()
	if cur == nil {
		content := lipgloss.JoinVertical(lipgloss.Left,
			title,
			divider,
			"",
			retroSubtleStyle.Render("  Nothing playing"),
			retroSubtleStyle.Render("  Start a track from Browse or Search"),
			"",
			divider,
		)
		return boxStyle.Render(content)
	}

	info := m.renderTrackInfo(cur, innerW)
	top := info
	if m.cover != "" && innerW >= coverWCells+30 {
		top = lipgloss.JoinHorizontal(lipgloss.Top, m.cover, "  ", info)
	}

	lines := []string{title, divider, top, divider}
	lines = append(lines, m.renderLyrics(innerW, h)...)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(content)
}

func (m NowPlayingModel) renderTrackInfo(cur *models.Track, innerW int) string {
	labelStyle := retroSubtleStyle
	valueStyle := lipgloss.NewStyle().Foreground(colorLightText)

	rows := []string{
		"",
		retroCurrentStyle.Render(truncateStr(cur.Title, innerW-coverWCells-8)),
		labelStyle.Render("Artist  ") + valueStyle.Render(truncateStr(cur.Artist, innerW-coverWCells-16)),
		labelStyle.Render("Album   ") + valueStyle.Render(truncateStr(cur.Album, innerW-coverWCells-16)),
	}
	if cur.Format != "" {
		rows = append(rows, labelStyle.Render("Format  ")+valueStyle.Render(strings.ToUpper(cur.Format)))
	}
	if cur.Duration > 0 {
		rows = append(rows, labelStyle.Render("Length  ")+valueStyle.Render(FormatDuration(cur.Duration)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m NowPlayingModel) renderLyrics(innerW, h int) []string {
	if len(m.lyrics.Lines) == 0 {
		return []string{retroSubtleStyle.Render("  No lyrics available")}
	}

	visible := calcVisibleRows(h, coverHCells+9)
	cur := -1
	if m.lyrics.Synced {
		cur = currentLyricIndex(m.lyrics.Lines, m.posMs)
	}

	// Keep the active line vertically centered in the lyric window.
	start := 0
	if cur > visible/2 {
		start = cur - visible/2
	}
	if start+visible > len(m.lyrics.Lines) {
		start = len(m.lyrics.Lines) - visible
	}
	if start < 0 {
		start = 0
	}
	end := start + visible
	if end > len(m.lyrics.Lines) {
		end = len(m.lyrics.Lines)
	}

	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		text := truncateStr(m.lyrics.Lines[i].Text, innerW-4)
		if i == cur {
			out = append(out, retroCurrentStyle.Render(fmt.Sprintf("▶ %s", text)))
		} else {
			out = append(out, retroSubtleStyle.Render(fmt.Sprintf("  %s", text)))
		}
	}
	return out
}
