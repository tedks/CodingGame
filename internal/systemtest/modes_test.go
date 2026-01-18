package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Mode tests verify vim-style mode transitions (Normal, Insert, Visual).

func testModesIEntersInsertMode(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state is Normal
	assertMode(t, h, input.ModeNormal)

	// Press I key
	source.QueueKeyPress(ebiten.KeyI)
	source.AdvanceFrame()
	h.Update()

	// Should now be in Insert mode
	assertMode(t, h, input.ModeInsert)
}

func testModesVEntersVisualMode(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state is Normal
	assertMode(t, h, input.ModeNormal)

	// Press V key
	source.QueueKeyPress(ebiten.KeyV)
	source.AdvanceFrame()
	h.Update()

	// Should now be in Visual mode
	assertMode(t, h, input.ModeVisual)
}

func testModesEscapeReturnsToNormal(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode first
	source.QueueKeyPress(ebiten.KeyI)
	source.AdvanceFrame()
	h.Update()
	assertMode(t, h, input.ModeInsert)

	// Press Escape
	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	h.Update()

	// Should now be back in Normal mode
	assertMode(t, h, input.ModeNormal)
}

func testModeTransitions(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Test: Normal -> Insert -> Normal
	assertMode(t, h, input.ModeNormal)

	source.QueueKeyPress(ebiten.KeyI)
	source.AdvanceFrame()
	h.Update()
	assertMode(t, h, input.ModeInsert)

	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	h.Update()
	assertMode(t, h, input.ModeNormal)

	// Test: Normal -> Visual -> Normal
	source.QueueKeyPress(ebiten.KeyV)
	source.AdvanceFrame()
	h.Update()
	assertMode(t, h, input.ModeVisual)

	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	h.Update()
	assertMode(t, h, input.ModeNormal)
}

func testModesColonBehavior(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state is Normal
	assertMode(t, h, input.ModeNormal)

	// Press : (colon) - should focus prompt and enter insert mode
	source.QueueKeyPress(ebiten.KeySemicolon) // Shift+; = :
	// Note: This might need modifier handling
	source.AdvanceFrame()
	h.Update()

	// The behavior depends on bindings configuration
	// By default, : should focus the prompt area
}

func testModeIndicatorUpdates(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Track mode changes
	modeChanges := []input.Mode{}
	h.OnModeChange(func(mode input.Mode) {
		modeChanges = append(modeChanges, mode)
	})

	// Change modes and verify callbacks fire
	source.QueueKeyPress(ebiten.KeyI)
	source.AdvanceFrame()
	h.Update()

	if len(modeChanges) != 1 || modeChanges[0] != input.ModeInsert {
		t.Errorf("expected mode change to Insert, got %v", modeChanges)
	}

	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	h.Update()

	if len(modeChanges) != 2 || modeChanges[1] != input.ModeNormal {
		t.Errorf("expected mode change to Normal, got %v", modeChanges)
	}
}
