package views

import (
	"testing"

	"github.com/codila125/musica/internal/models"
)

func TestCurrentLyricIndex(t *testing.T) {
	lines := []models.LyricLine{
		{StartMs: 0, Text: "one"},
		{StartMs: 5000, Text: "two"},
		{StartMs: 12000, Text: "three"},
	}
	cases := []struct {
		name  string
		posMs int
		want  int
	}{
		{"before all", -1, -1},
		{"first line", 0, 0},
		{"mid first", 4999, 0},
		{"exact second", 5000, 1},
		{"mid last", 20000, 2},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := currentLyricIndex(lines, tt.posMs); got != tt.want {
				t.Fatalf("currentLyricIndex(%d) = %d, want %d", tt.posMs, got, tt.want)
			}
		})
	}
	if got := currentLyricIndex(nil, 100); got != -1 {
		t.Fatalf("empty lines = %d, want -1", got)
	}
}
