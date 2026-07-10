package tui

import (
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

func TestPadCenterUsesDisplayWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		w    int
	}{
		{"ascii", "abc", 9},
		{"multibyte led", "◉ MUSICA ◉", 20},
		{"cjk title", "残酷な天使のテーゼ", 30},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := padCenter(tt.in, tt.w)
			if lipgloss.Width(got) != tt.w {
				t.Fatalf("padCenter(%q, %d) display width = %d, want %d", tt.in, tt.w, lipgloss.Width(got), tt.w)
			}
		})
	}
}

func TestTrimLabelUsesDisplayWidth(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
	}{
		{"ascii too long", "abcdefghij", 6},
		{"multibyte kept whole", "◉◉◉", 5},
		{"cjk trimmed", "残酷な天使のテーゼ", 8},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := trimLabel(tt.in, tt.max)
			if lipgloss.Width(got) > tt.max {
				t.Fatalf("trimLabel(%q, %d) display width = %d, want <= %d", tt.in, tt.max, lipgloss.Width(got), tt.max)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("trimLabel(%q, %d) = %q, invalid UTF-8", tt.in, tt.max, got)
			}
		})
	}
}
