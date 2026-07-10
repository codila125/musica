package views

import (
	"fmt"
	"image"
	"sort"
	"strings"

	"github.com/codila125/musica/internal/models"
)

// renderCoverArt draws an image as wCells x hCells terminal cells using
// half-block characters: the upper half of each cell is the foreground
// color, the lower half the background, giving 2 pixels per cell height.
func renderCoverArt(img image.Image, wCells, hCells int) string {
	if img == nil || wCells <= 0 || hCells <= 0 {
		return ""
	}
	bounds := img.Bounds()
	iw, ih := bounds.Dx(), bounds.Dy()
	if iw == 0 || ih == 0 {
		return ""
	}

	sample := func(px, py int) (r, g, b uint8) {
		c := img.At(bounds.Min.X+px, bounds.Min.Y+py)
		r16, g16, b16, _ := c.RGBA()
		return uint8(r16 >> 8), uint8(g16 >> 8), uint8(b16 >> 8)
	}

	pixelRows := hCells * 2
	var sb strings.Builder
	for row := 0; row < hCells; row++ {
		for col := 0; col < wCells; col++ {
			// Nearest-neighbor sampling keeps this stdlib-only and fast
			// at the tiny output sizes a terminal needs.
			topY := (row * 2) * ih / pixelRows
			botY := (row*2 + 1) * ih / pixelRows
			x := col * iw / wCells
			tr, tg, tb := sample(x, topY)
			br, bg, bb := sample(x, botY)
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", tr, tg, tb, br, bg, bb)
		}
		sb.WriteString("\x1b[0m")
		if row < hCells-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// currentLyricIndex returns the index of the lyric line active at posMs,
// or -1 when playback is before the first line.
func currentLyricIndex(lines []models.LyricLine, posMs int) int {
	if len(lines) == 0 {
		return -1
	}
	// First line with StartMs > posMs; active line is the one before it.
	i := sort.Search(len(lines), func(i int) bool {
		return lines[i].StartMs > posMs
	})
	return i - 1
}
