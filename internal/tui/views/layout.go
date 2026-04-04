package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minViewWidth     = 40
	minViewHeight    = 10
	minVisibleRows   = 3
	artistColWidth   = 24
	albumColWidth    = 28
	durationColWidth = 8
	trackPrefixWidth = 4
)

type trackColumns struct {
	nameW        int
	artistW      int
	albumW       int
	durationW    int
	showArtist   bool
	showAlbum    bool
	showDuration bool
}

func normalizeViewSize(w, h int) (int, int) {
	if w < minViewWidth {
		w = minViewWidth
	}
	if h < minViewHeight {
		h = minViewHeight
	}
	return w, h
}

func listBoxStyle(w, h int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPurpleBorder).
		Padding(0, 1).
		Width(w - 6).
		Height(h - 2)
}

func listDivider(innerW int) string {
	return retroCassetteStyle.Render(strings.Repeat("─", innerW))
}

func calcVisibleRows(h, reserved int) int {
	rows := h - reserved
	if rows < minVisibleRows {
		rows = minVisibleRows
	}
	return rows
}

func computeTrackColumns(innerW int) trackColumns {
	cols := trackColumns{
		artistW:      artistColWidth,
		albumW:       albumColWidth,
		durationW:    durationColWidth,
		showArtist:   true,
		showAlbum:    true,
		showDuration: true,
	}

	calcName := func() int {
		w := innerW - trackPrefixWidth
		if cols.showArtist {
			w -= cols.artistW + 1
		}
		if cols.showAlbum {
			w -= cols.albumW + 1
		}
		if cols.showDuration {
			w -= cols.durationW + 1
		}
		return w
	}

	cols.nameW = calcName()
	for cols.nameW < 14 && (cols.albumW > 10 || cols.artistW > 10 || cols.durationW > 5) {
		if cols.showAlbum && cols.albumW > 10 {
			cols.albumW--
		}
		if cols.showArtist && cols.artistW > 10 {
			cols.artistW--
		}
		if cols.showDuration && cols.durationW > 5 {
			cols.durationW--
		}
		cols.nameW = calcName()
	}

	if cols.nameW < 12 && cols.showAlbum {
		cols.showAlbum = false
		cols.nameW = calcName()
	}
	if cols.nameW < 12 && cols.showArtist {
		cols.showArtist = false
		cols.nameW = calcName()
	}
	if cols.nameW < 10 && cols.showDuration {
		cols.showDuration = false
		cols.nameW = calcName()
	}

	if cols.nameW < 10 {
		cols.nameW = 10
	}

	return cols
}

func trackTableHeader(cols trackColumns) string {
	header := retroSubtleStyle.Render("  # ") +
		retroColumnHeaderStyle.Render(padRight("NAME", cols.nameW))

	if cols.showArtist {
		header += retroSubtleStyle.Render(" ") +
			retroColumnHeaderStyle.Render(padRight("ARTIST", cols.artistW))
	}
	if cols.showAlbum {
		header += retroSubtleStyle.Render(" ") +
			retroColumnHeaderStyle.Render(padRight("ALBUM", cols.albumW))
	}
	if cols.showDuration {
		header += retroSubtleStyle.Render(" ") +
			retroColumnHeaderStyle.Render(padRight("TIME", cols.durationW))
	}

	return header
}
