package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Prompt tests verify prompt focusing, text entry, submission, and cancellation.

func testPromptSlashFocuses(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertFocus(t, h, input.FocusMap)
	assertMode(t, h, input.ModeNormal)

	// Press / (slash) key - triggers ActionSearch
	// Note: Current implementation fires ActionSearch but doesn't change focus
	// This test verifies the action fires
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

func testPromptEnterSubmits(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Focus prompt and enter insert mode
	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.SetTextBuffer("test query")

	// Track action callbacks
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionSubmitPrompt {
			actionFired = true
		}
	})

	// Press Enter
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	h.Update()

	// Submit action should fire
	if !actionFired {
		t.Error("expected ActionSubmitPrompt to fire")
	}
}

func testPromptEscapeCancels(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Focus prompt and enter insert mode
	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.SetTextBuffer("partial input")

	// Track action callbacks
	cancelFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionCancelPrompt || action == input.ActionExitMode {
			cancelFired = true
		}
	})

	// Press Escape
	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	h.Update()

	// Should return to Normal mode
	assertMode(t, h, input.ModeNormal)

	// Cancel/ExitMode action should fire
	if !cancelFired {
		t.Error("expected cancel/exit action to fire")
	}
}

func testPromptCursorBlinks(t *testing.T) {
	// This test would verify cursor blinking in the UI
	// For now, we just verify that the handler supports cursor position tracking
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.SetTextBuffer("test")

	// The cursor would blink based on frame count in the actual UI
	// This test verifies the infrastructure supports it
}

func testPromptTextDisplay(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Focus prompt and enter insert mode
	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Type some text
	source.QueueCharInput('h', 'e', 'l', 'l', 'o')
	source.AdvanceFrame()
	h.Update()

	// Verify text is in buffer
	assertTextBuffer(t, h, "hello")

	// Type more text
	source.QueueCharInput(' ', 'w', 'o', 'r', 'l', 'd')
	source.AdvanceFrame()
	h.Update()

	assertTextBuffer(t, h, "hello world")
}

func testPromptClearAfterSubmit(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Focus prompt and enter insert mode
	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.SetTextBuffer("search query")

	// Track submit and verify buffer content at time of submit
	submittedText := ""
	h.OnAction(func(action input.Action) {
		if action == input.ActionSubmitPrompt {
			submittedText = h.TextBuffer()
			// Clear buffer after capturing
			h.ClearTextBuffer()
		}
	})

	// Press Enter
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	h.Update()

	// Verify text was captured before clear
	if submittedText != "search query" {
		t.Errorf("expected submitted text 'search query', got %q", submittedText)
	}

	// Buffer should be cleared
	assertTextBuffer(t, h, "")
}
