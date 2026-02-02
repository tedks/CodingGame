package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Emacs tests verify Emacs-style keybindings (Ctrl combinations).

func testEmacsCtrlPMovesUp(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	h.SetBindingStyle(input.StyleEmacs)

	// Track if MoveUp action fires
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionMoveUp {
			actionFired = true
		}
	})

	// Hold Ctrl and press P
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyP)
	source.AdvanceFrame()
	h.Update()

	if !actionFired {
		t.Error("expected ActionMoveUp to fire on Ctrl+P")
	}
}

func testEmacsCtrlNMovesDown(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	h.SetBindingStyle(input.StyleEmacs)

	// Track if MoveDown action fires
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionMoveDown {
			actionFired = true
		}
	})

	// Hold Ctrl and press N
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyN)
	source.AdvanceFrame()
	h.Update()

	if !actionFired {
		t.Error("expected ActionMoveDown to fire on Ctrl+N")
	}
}

func testEmacsCtrlBMovesLeft(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	h.SetBindingStyle(input.StyleEmacs)

	// Track if MoveLeft action fires
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionMoveLeft {
			actionFired = true
		}
	})

	// Hold Ctrl and press B
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyB)
	source.AdvanceFrame()
	h.Update()

	if !actionFired {
		t.Error("expected ActionMoveLeft to fire on Ctrl+B")
	}
}

func testEmacsCtrlFMovesRight(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	h.SetBindingStyle(input.StyleEmacs)

	// Track if MoveRight action fires
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionMoveRight {
			actionFired = true
		}
	})

	// Hold Ctrl and press F
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyF)
	source.AdvanceFrame()
	h.Update()

	if !actionFired {
		t.Error("expected ActionMoveRight to fire on Ctrl+F")
	}
}

func testEmacsCtrlGExitsMode(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	h.SetBindingStyle(input.StyleEmacs)

	// Start in Insert mode
	h.SetMode(input.ModeInsert)
	assertMode(t, h, input.ModeInsert)

	// Hold Ctrl and press G
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyG)
	source.AdvanceFrame()
	h.Update()

	// Should exit to Normal mode
	assertMode(t, h, input.ModeNormal)
}

func testEmacsCtrlSSearches(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	h.SetBindingStyle(input.StyleEmacs)

	// Track if Search action fires
	actionFired := false
	h.OnAction(func(action input.Action) {
		if action == input.ActionSearch {
			actionFired = true
		}
	})

	// Hold Ctrl and press S
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyS)
	source.AdvanceFrame()
	h.Update()

	if !actionFired {
		t.Error("expected ActionSearch to fire on Ctrl+S")
	}
}
