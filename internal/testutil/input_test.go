package testutil

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestTestInputSource_Basic(t *testing.T) {
	source := NewTestInputSource()

	// Initially, no keys should be pressed
	if len(source.JustPressedKeys()) != 0 {
		t.Error("expected no just-pressed keys initially")
	}

	// Queue a key press
	source.QueueKeyPress(ebiten.KeyA)

	// Advance frame to process the event
	source.AdvanceFrame()

	// Key should be just pressed
	if !source.IsKeyJustPressed(ebiten.KeyA) {
		t.Error("expected KeyA to be just pressed")
	}

	// Key should also be pressed (held)
	if !source.IsKeyPressed(ebiten.KeyA) {
		t.Error("expected KeyA to be pressed")
	}

	// Advance another frame
	source.AdvanceFrame()

	// Key should no longer be just pressed (auto-released after 1 frame)
	if source.IsKeyJustPressed(ebiten.KeyA) {
		t.Error("expected KeyA to not be just pressed after advance")
	}
}

func TestTestInputSource_KeyHold(t *testing.T) {
	source := NewTestInputSource()

	// Hold key for 3 frames
	source.QueueKeyHold(ebiten.KeyH, 3)
	source.AdvanceFrame()

	// Key should be just pressed on first frame
	if !source.IsKeyJustPressed(ebiten.KeyH) {
		t.Error("expected KeyH to be just pressed on first frame")
	}

	// Key should remain held for 3 frames
	for i := 0; i < 3; i++ {
		if !source.IsKeyPressed(ebiten.KeyH) {
			t.Errorf("frame %d: expected KeyH to be held", i)
		}
		source.AdvanceFrame()
	}

	// After duration, key should be released
	if source.IsKeyPressed(ebiten.KeyH) {
		t.Error("expected KeyH to be released after hold duration")
	}
}

func TestTestInputSource_CharInput(t *testing.T) {
	source := NewTestInputSource()

	// Queue characters
	source.QueueCharInput('a', 'b', 'c')
	source.AdvanceFrame()

	// Check input chars
	chars := source.AppendInputChars(nil)
	if len(chars) != 3 {
		t.Errorf("expected 3 chars, got %d", len(chars))
	}
	if string(chars) != "abc" {
		t.Errorf("expected 'abc', got %q", string(chars))
	}
}

func TestTestInputSource_TextInput(t *testing.T) {
	source := NewTestInputSource()

	// Queue text input
	source.QueueTextInput("hello")
	source.AdvanceFrame()

	// Check input chars
	chars := source.AppendInputChars(nil)
	if string(chars) != "hello" {
		t.Errorf("expected 'hello', got %q", string(chars))
	}
}

func TestTestInputSource_MouseMove(t *testing.T) {
	source := NewTestInputSource()

	// Initially at 0,0
	x, y := source.CursorPosition()
	if x != 0 || y != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", x, y)
	}

	// Queue mouse move
	source.QueueMouseMove(100, 200)
	source.AdvanceFrame()

	// Check position
	x, y = source.CursorPosition()
	if x != 100 || y != 200 {
		t.Errorf("expected (100,200), got (%d,%d)", x, y)
	}
}

func TestTestInputSource_MouseClick(t *testing.T) {
	source := NewTestInputSource()

	// Queue mouse click
	source.QueueMouseClick(ebiten.MouseButtonLeft)
	source.AdvanceFrame()

	// Check button state
	if !source.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		t.Error("expected left button to be just pressed")
	}
	if !source.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		t.Error("expected left button to be pressed")
	}
}

func TestTestInputSource_MouseWheel(t *testing.T) {
	source := NewTestInputSource()

	// Queue mouse wheel
	source.QueueMouseWheel(1.5, -2.0)
	source.AdvanceFrame()

	// Check wheel values
	xoff, yoff := source.Wheel()
	if xoff != 1.5 || yoff != -2.0 {
		t.Errorf("expected (1.5, -2.0), got (%v, %v)", xoff, yoff)
	}
}

func TestTestInputSource_Clear(t *testing.T) {
	source := NewTestInputSource()

	// Queue various events
	source.QueueKeyPress(ebiten.KeyA)
	source.QueueMouseMove(100, 100)
	source.AdvanceFrame()

	// Clear
	source.Clear()

	// All state should be reset
	if source.IsKeyPressed(ebiten.KeyA) {
		t.Error("expected KeyA to be cleared")
	}
	x, y := source.CursorPosition()
	if x != 0 || y != 0 {
		t.Errorf("expected (0,0) after clear, got (%d,%d)", x, y)
	}
}
