package views

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func solidImage(w, h int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

func TestRenderCoverArtDimensions(t *testing.T) {
	img := solidImage(100, 100, color.RGBA{R: 255, A: 255})
	got := renderCoverArt(img, 20, 10)

	lines := strings.Split(got, "\n")
	if len(lines) != 10 {
		t.Fatalf("lines = %d, want 10", len(lines))
	}
	if cells := strings.Count(lines[0], "▀"); cells != 20 {
		t.Fatalf("cells in first line = %d, want 20", cells)
	}
}

func TestRenderCoverArtUsesTruecolor(t *testing.T) {
	img := solidImage(4, 4, color.RGBA{R: 255, A: 255})
	got := renderCoverArt(img, 2, 2)

	if !strings.Contains(got, "38;2;255;0;0") || !strings.Contains(got, "48;2;255;0;0") {
		t.Fatalf("want truecolor fg+bg red codes, got %q", got)
	}
}

func TestRenderCoverArtEmpty(t *testing.T) {
	if got := renderCoverArt(nil, 10, 5); got != "" {
		t.Fatalf("nil image must render empty, got %q", got)
	}
}
