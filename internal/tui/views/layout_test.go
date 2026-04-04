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

func TestTrackNameWidth(t *testing.T) {
	nameW := trackNameWidth(20)
	if nameW != 10 {
		t.Fatalf("expected minimum name width 10, got %d", nameW)
	}

	nameW = trackNameWidth(120)
	if nameW != 29 {
		t.Fatalf("expected name width 29, got %d", nameW)
	}
}
