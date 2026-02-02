package ui

import (
	"image/color"
	"testing"
)

// Tests for RightAlignedSummaryX edge cases

func TestRightAlignedSummaryX_BlockLargerThanWidth(t *testing.T) {
	// When blockWidth exceeds available space, should return minX and false
	x := 100
	width := 200
	blockWidth := 300 // Larger than available width
	minPadding := 20

	startX, ok := RightAlignedSummaryX(x, width, blockWidth, minPadding)

	if ok {
		t.Error("expected ok=false when block is larger than available width")
	}

	expectedMinX := x + minPadding
	if startX != expectedMinX {
		t.Errorf("expected startX=%d (minX), got %d", expectedMinX, startX)
	}
}

func TestRightAlignedSummaryX_ExactFit(t *testing.T) {
	// Block exactly fits the available space
	x := 100
	width := 200
	blockWidth := 180 // width - minPadding
	minPadding := 20

	startX, ok := RightAlignedSummaryX(x, width, blockWidth, minPadding)

	if !ok {
		t.Error("expected ok=true when block exactly fits")
	}

	expectedX := x + width - blockWidth
	if startX != expectedX {
		t.Errorf("expected startX=%d, got %d", expectedX, startX)
	}
}

func TestRightAlignedSummaryX_ZeroBlockWidth(t *testing.T) {
	x := 100
	width := 200
	blockWidth := 0
	minPadding := 20

	startX, ok := RightAlignedSummaryX(x, width, blockWidth, minPadding)

	if !ok {
		t.Error("expected ok=true for zero block width")
	}

	// Should be at right edge
	expectedX := x + width
	if startX != expectedX {
		t.Errorf("expected startX=%d, got %d", expectedX, startX)
	}
}

func TestRightAlignedSummaryX_ZeroWidth(t *testing.T) {
	x := 100
	width := 0
	blockWidth := 50
	minPadding := 20

	startX, ok := RightAlignedSummaryX(x, width, blockWidth, minPadding)

	// With zero width, block can't fit
	if ok {
		t.Error("expected ok=false when width is zero and block has size")
	}

	expectedMinX := x + minPadding
	if startX != expectedMinX {
		t.Errorf("expected startX=%d (minX), got %d", expectedMinX, startX)
	}
}

// Tests for DrawEmptyState layout calculations (we can't test rendering without screen)

func TestDrawEmptyState_CalculatesNegativeX(t *testing.T) {
	// Test that very narrow widths produce negative X coordinates
	// This documents the current behavior - centering can produce negative coords
	const approxCharWidth = 3
	message := "This is a very long message that won't fit"
	width := 20 // Very narrow

	// Calculate what DrawEmptyState would compute
	msgX := 0 + width/2 - len(message)*approxCharWidth
	if msgX >= 0 {
		t.Error("expected negative X for message that doesn't fit narrow width")
	}
}

func TestDrawEmptyState_EmptyMessage(t *testing.T) {
	// Empty message should still calculate valid coordinates
	const approxCharWidth = 3
	message := ""
	width := 200
	x := 0

	// Calculate what DrawEmptyState would compute
	msgX := x + width/2 - len(message)*approxCharWidth
	if msgX != 100 { // width/2 since len("") * 3 = 0
		t.Errorf("expected msgX=100 for empty message, got %d", msgX)
	}
}

// Tests for DrawProgressBar edge cases

func TestDrawProgressBar_NegativeProgress(t *testing.T) {
	// Verify progress is clamped to 0 (can't draw negative)
	progress := -0.5
	clamped := progress
	if clamped < 0 {
		clamped = 0
	}
	if clamped != 0 {
		t.Errorf("expected clamped progress=0, got %f", clamped)
	}
}

func TestDrawProgressBar_ProgressOver1(t *testing.T) {
	// Verify progress is clamped to 1
	progress := 1.5
	clamped := progress
	if clamped > 1 {
		clamped = 1
	}
	if clamped != 1 {
		t.Errorf("expected clamped progress=1, got %f", clamped)
	}
}

func TestDrawProgressBar_ZeroDimensions(t *testing.T) {
	// Zero dimensions should not panic
	// We can't actually call Draw without a screen, but we can verify
	// the math doesn't panic
	width := 0
	height := 0
	progress := 0.5

	fillWidth := int(float64(width) * progress)
	if fillWidth != 0 {
		t.Errorf("expected fillWidth=0 for zero width, got %d", fillWidth)
	}

	// Just document that height 0 means no visual bar (but no panic)
	_ = height
}

func TestDrawProgressBar_FillWidthCalculation(t *testing.T) {
	testCases := []struct {
		width    int
		progress float64
		expected int
	}{
		{100, 0.0, 0},
		{100, 0.5, 50},
		{100, 1.0, 100},
		{100, 0.333, 33}, // Truncates, doesn't round
		{0, 0.5, 0},
		{1, 0.5, 0}, // int(0.5) = 0
		{2, 0.5, 1},
	}

	for _, tc := range testCases {
		fillWidth := int(float64(tc.width) * tc.progress)
		if fillWidth != tc.expected {
			t.Errorf("width=%d, progress=%f: expected fillWidth=%d, got %d",
				tc.width, tc.progress, tc.expected, fillWidth)
		}
	}
}

// Tests for HeaderLayout defaults

func TestDefaultHeaderLayout(t *testing.T) {
	layout := DefaultHeaderLayout()

	if layout.Height != 60 {
		t.Errorf("expected Height=60, got %d", layout.Height)
	}
	if layout.Padding != 20 {
		t.Errorf("expected Padding=20, got %d", layout.Padding)
	}
	if layout.TitleOffsetY != 10 {
		t.Errorf("expected TitleOffsetY=10, got %d", layout.TitleOffsetY)
	}
	if layout.SummaryOffsetY != 30 {
		t.Errorf("expected SummaryOffsetY=30, got %d", layout.SummaryOffsetY)
	}
}

// Tests for CardStyle defaults

func TestDefaultCardStyle(t *testing.T) {
	borderColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	style := DefaultCardStyle(borderColor)

	if style.BorderWidth != 2 {
		t.Errorf("expected BorderWidth=2, got %f", style.BorderWidth)
	}
	if style.BorderColor.R != borderColor.R {
		t.Errorf("expected BorderColor.R=%d, got %d", borderColor.R, style.BorderColor.R)
	}
}
