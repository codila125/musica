package views

import "testing"

func TestNormalizeViewSize(t *testing.T) {
	w, h := normalizeViewSize(1, 1)
	if w != minViewWidth {
		t.Fatalf("expected width %d, got %d", minViewWidth, w)
	}
	if h != minViewHeight {
		t.Fatalf("expected height %d, got %d", minViewHeight, h)
	}
}

func TestCalcVisibleRows(t *testing.T) {
	rows := calcVisibleRows(5, 10)
	if rows != minVisibleRows {
		t.Fatalf("expected min rows %d, got %d", minVisibleRows, rows)
	}

	rows = calcVisibleRows(30, 8)
	if rows != 22 {
		t.Fatalf("expected 22 rows, got %d", rows)
	}
}

func TestComputeTrackColumnsPriority(t *testing.T) {
	wide := computeTrackColumns(120)
	if !wide.showArtist || !wide.showAlbum || !wide.showDuration {
		t.Fatalf("expected all columns visible in wide layout")
	}

	narrow := computeTrackColumns(40)
	if narrow.nameW < 10 {
		t.Fatalf("expected minimum name width, got %d", narrow.nameW)
	}
	if narrow.showAlbum && narrow.albumW > albumColWidth {
		t.Fatalf("expected album to shrink or hide")
	}

	veryNarrow := computeTrackColumns(24)
	if veryNarrow.showAlbum {
		t.Fatalf("expected album hidden in very narrow layout")
	}
}
