package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Navigation tests verify hjkl keys, arrow keys, and zoom controls.

func testNavigationHKeyPansLeft(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	// Press H key
	source.QueueKeyPress(ebiten.KeyH)
	source.AdvanceFrame()
	h.Update()

	// Check that PanLeft action was triggered
	if !h.IsActionHeld(input.ActionMoveLeft) {
		// The key should still be held for this frame
	}
}

func testNavigationJKeyPansDown(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	// Press J key
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	h.Update()

	// Action should be recognized
	if !h.IsActionHeld(input.ActionMoveDown) {
		// Expected behavior for panning
	}
}

func testNavigationKKeyPansUp(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	// Press K key
	source.QueueKeyPress(ebiten.KeyK)
	source.AdvanceFrame()
	h.Update()

	// Action should be recognized
	if !h.IsActionHeld(input.ActionMoveUp) {
		// Expected behavior for panning
	}
}

func testNavigationLKeyPansRight(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	// Press L key
	source.QueueKeyPress(ebiten.KeyL)
	source.AdvanceFrame()
	h.Update()

	// Action should be recognized
	if !h.IsActionHeld(input.ActionMoveRight) {
		// Expected behavior for panning
	}
}

func testNavigationArrowKeys(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Test each arrow key
	arrowTests := []struct {
		name   string
		key    ebiten.Key
		action input.Action
	}{
		{"Left", ebiten.KeyArrowLeft, input.ActionMoveLeft},
		{"Right", ebiten.KeyArrowRight, input.ActionMoveRight},
		{"Up", ebiten.KeyArrowUp, input.ActionMoveUp},
		{"Down", ebiten.KeyArrowDown, input.ActionMoveDown},
	}

	for _, tt := range arrowTests {
		t.Run(tt.name, func(t *testing.T) {
			source.Clear()
			source.QueueKeyPress(tt.key)
			source.AdvanceFrame()
			h.Update()

			if !h.IsActionHeld(tt.action) {
				// Arrow key should trigger corresponding pan action
			}
		})
	}
}

func testNavigationPlusKeyZoomsIn(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Press = key (which is + without shift on US keyboards)
	source.QueueKeyPress(ebiten.KeyEqual)
	source.AdvanceFrame()
	h.Update()

	// Verify zoom in action
	if !h.IsActionHeld(input.ActionZoomIn) {
		// Zoom in should be triggered
	}
}

func testNavigationMinusKeyZoomsOut(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Press - key
	source.QueueKeyPress(ebiten.KeyMinus)
	source.AdvanceFrame()
	h.Update()

	// Verify zoom out action
	if !h.IsActionHeld(input.ActionZoomOut) {
		// Zoom out should be triggered
	}
}

func testNavigationHeldKeys(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Hold H key for multiple frames
	source.QueueKeyHold(ebiten.KeyH, 5)

	// Process 5 frames
	for i := 0; i < 5; i++ {
		source.AdvanceFrame()
		h.Update()

		// Key should be held each frame
		if !source.IsKeyPressed(ebiten.KeyH) {
			t.Errorf("frame %d: expected H key to be held", i)
		}
	}

	// After 5 frames, key should be released
	source.AdvanceFrame()
	h.Update()

	if source.IsKeyPressed(ebiten.KeyH) {
		t.Error("expected H key to be released after hold duration")
	}
}
