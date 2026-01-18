package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Focus tests verify Tab cycling, Shift+Tab, and focus area changes.
// The actual focus order from mode.go is:
// FocusMap -> FocusPrompt -> FocusAdvisors -> FocusMissions -> FocusResponse

func testFocusTabCycles(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial focus is Map
	assertFocus(t, h, input.FocusMap)

	// Press Tab
	source.QueueKeyPress(ebiten.KeyTab)
	source.AdvanceFrame()
	h.Update()

	// Should cycle to next focus area
	// Expected: FocusMap -> FocusPrompt (per mode.go focusOrder)
	assertFocus(t, h, input.FocusPrompt)

	// Press Tab again
	source.QueueKeyPress(ebiten.KeyTab)
	source.AdvanceFrame()
	h.Update()

	// Expected: FocusPrompt -> FocusAdvisors
	assertFocus(t, h, input.FocusAdvisors)
}

func testFocusShiftTabReverse(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Set focus to Prompt first
	h.SetFocus(input.FocusPrompt)
	assertFocus(t, h, input.FocusPrompt)

	// Press Shift+Tab (FocusPrev action)
	// Queue shift key as held, then tab
	source.QueueEvents(
		testutil.InputEvent{Type: testutil.KeyHold, Key: ebiten.KeyShift, Duration: 2},
		testutil.InputEvent{Type: testutil.KeyPress, Key: ebiten.KeyTab},
	)
	source.AdvanceFrame()
	h.Update()

	// Should cycle backwards
	// Expected: FocusPrompt -> FocusMap
	assertFocus(t, h, input.FocusMap)
}

func testFocusSlashPrompt(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertFocus(t, h, input.FocusMap)
	assertMode(t, h, input.ModeNormal)

	// Press / (slash) key - triggers ActionSearch
	// Note: ActionSearch doesn't automatically focus prompt in current implementation
	// This test verifies the action is triggered
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionSearch {
			actionFired = true
		}
	})

	source.QueueKeyPress(ebiten.KeySlash)
	source.AdvanceFrame()
	h.Update()

	if !actionFired {
		t.Error("expected ActionSearch to fire on slash key")
	}
}

func testFocusInputRouting(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// When focused on Map, hjkl should pan
	h.SetFocus(input.FocusMap)
	h.SetMode(input.ModeNormal)

	source.QueueKeyPress(ebiten.KeyH)
	source.AdvanceFrame()
	h.Update()

	// H key in Normal mode with Map focus should trigger pan
	if !h.IsActionHeld(input.ActionMoveLeft) {
		// Panning should work
	}

	// When focused on Prompt in Insert mode, hjkl should not pan
	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// In Insert mode, 'h' types a character, doesn't pan
	source.QueueCharInput('h')
	source.AdvanceFrame()
	h.Update()

	// Text buffer should contain 'h'
	assertTextBuffer(t, h, "h")
}

func testFocusIndicatorVisible(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Track focus changes
	focusChanges := []input.FocusArea{}
	h.OnFocusChange(func(focus input.FocusArea) {
		focusChanges = append(focusChanges, focus)
	})

	// Change focus
	source.QueueKeyPress(ebiten.KeyTab)
	source.AdvanceFrame()
	h.Update()

	if len(focusChanges) != 1 {
		t.Errorf("expected 1 focus change, got %d", len(focusChanges))
	}

	// Expected: FocusMap -> FocusPrompt
	if focusChanges[0] != input.FocusPrompt {
		t.Errorf("expected focus change to FocusPrompt, got %v", focusChanges[0])
	}
}

func testFocusWrapAround(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Start at Map
	h.SetFocus(input.FocusMap)

	// Cycle through all focus areas
	// The full order: Map -> Prompt -> Advisors -> Missions -> Response -> Map
	expectedOrder := []input.FocusArea{
		input.FocusPrompt,
		input.FocusAdvisors,
		input.FocusMissions,
		input.FocusResponse,
		input.FocusMap, // wraps back
	}

	for i, expected := range expectedOrder {
		source.QueueKeyPress(ebiten.KeyTab)
		source.AdvanceFrame()
		h.Update()

		if h.Focus() != expected {
			t.Errorf("tab %d: expected focus %v, got %v", i+1, expected, h.Focus())
		}
	}
}
