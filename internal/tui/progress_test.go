package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderProgressBar(t *testing.T) {
	cases := []struct {
		name       string
		pos, dur   int
		w          int
		wantFilled int
	}{
		{"halfway", 150, 300, 20, 10},
		{"start", 0, 300, 20, 0},
		{"end", 300, 300, 20, 20},
		{"past end clamps", 400, 300, 20, 20},
		{"zero duration", 10, 0, 20, 0},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := renderProgressBar(tt.pos, tt.dur, tt.w)
			if lipgloss.Width(got) != tt.w {
				t.Fatalf("width = %d, want %d", lipgloss.Width(got), tt.w)
			}
			if filled := strings.Count(got, "█"); filled != tt.wantFilled {
				t.Fatalf("filled cells = %d, want %d (bar %q)", filled, tt.wantFilled, got)
			}
		})
	}
}
