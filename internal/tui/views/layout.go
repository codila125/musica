package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minViewWidth      = 40
	minViewHeight     = 10
	minVisibleRows    = 3
	artistColWidth    = 35
	albumColWidth     = 40
	durationColWidth  = 8
	trackFixedColSize = 91
)

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
		Width(w - 4).
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

func trackNameWidth(innerW int) int {
	nameW := innerW - trackFixedColSize
	if nameW < 10 {
		nameW = 10
	}
	return nameW
}

func trackTableHeader(nameW int) string {
	return retroSubtleStyle.Render("  # ") +
		retroColumnHeaderStyle.Render(padRight("NAME", nameW)) +
		retroSubtleStyle.Render(" ") +
		retroColumnHeaderStyle.Render(padRight("ARTIST", artistColWidth)) +
		retroSubtleStyle.Render(" ") +
		retroColumnHeaderStyle.Render(padRight("ALBUM", albumColWidth)) +
		retroSubtleStyle.Render(" ") +
		retroColumnHeaderStyle.Render(padRight("DURATION", durationColWidth))
}
