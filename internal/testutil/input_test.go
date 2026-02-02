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

// Completeness tests

func TestTestInputSource_MultipleKeysInSameFrame(t *testing.T) {
	source := NewTestInputSource()

	source.QueueKeyPress(ebiten.KeyA)
	source.QueueKeyPress(ebiten.KeyB)
	source.QueueKeyPress(ebiten.KeyC)
	source.AdvanceFrame()

	justPressed := source.JustPressedKeys()
	if len(justPressed) != 3 {
		t.Errorf("expected 3 just-pressed keys, got %d", len(justPressed))
	}

	for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyB, ebiten.KeyC} {
		if !source.IsKeyJustPressed(key) {
			t.Errorf("expected key %v to be just pressed", key)
		}
	}
}

func TestTestInputSource_InterleavedEvents(t *testing.T) {
	source := NewTestInputSource()

	source.QueueKeyPress(ebiten.KeyA)
	source.QueueMouseMove(50, 50)
	source.QueueKeyPress(ebiten.KeyB)
	source.QueueMouseClick(ebiten.MouseButtonLeft)
	source.QueueCharInput('x')
	source.AdvanceFrame()

	if !source.IsKeyJustPressed(ebiten.KeyA) || !source.IsKeyJustPressed(ebiten.KeyB) {
		t.Error("Keys should be pressed")
	}
	x, y := source.CursorPosition()
	if x != 50 || y != 50 {
		t.Errorf("cursor = (%d,%d), want (50,50)", x, y)
	}
	if !source.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		t.Error("left button should be pressed")
	}
	chars := source.AppendInputChars(nil)
	if len(chars) != 1 || chars[0] != 'x' {
		t.Errorf("chars = %v, want ['x']", chars)
	}
}

func TestTestInputSource_UnicodeCharInput(t *testing.T) {
	source := NewTestInputSource()

	testChars := []rune{'a', 'ñ', '日', '🎮', 'π', '€'}
	for _, c := range testChars {
		source.QueueCharInput(c)
	}
	source.AdvanceFrame()

	chars := source.AppendInputChars(nil)
	if len(chars) != len(testChars) {
		t.Errorf("expected %d chars, got %d", len(testChars), len(chars))
	}
	for i, expected := range testChars {
		if i < len(chars) && chars[i] != expected {
			t.Errorf("char[%d] = %q, want %q", i, chars[i], expected)
		}
	}
}

func TestTestInputSource_LargeEventQueue(t *testing.T) {
	source := NewTestInputSource()

	const numEvents = 1000
	for i := 0; i < numEvents; i++ {
		source.QueueCharInput(rune('a' + (i % 26)))
	}
	source.AdvanceFrame()

	chars := source.AppendInputChars(nil)
	if len(chars) != numEvents {
		t.Errorf("expected %d chars, got %d", numEvents, len(chars))
	}
}

func TestTestInputSource_WheelResetEachFrame(t *testing.T) {
	source := NewTestInputSource()

	source.QueueMouseWheel(1.0, 2.0)
	source.AdvanceFrame()

	x, y := source.Wheel()
	if x != 1.0 || y != 2.0 {
		t.Errorf("wheel = (%v,%v), want (1.0,2.0)", x, y)
	}

	source.AdvanceFrame()
	x, y = source.Wheel()
	if x != 0 || y != 0 {
		t.Errorf("wheel should reset to (0,0), got (%v,%v)", x, y)
	}
}

func TestTestInputSource_CharsResetEachFrame(t *testing.T) {
	source := NewTestInputSource()

	source.QueueTextInput("hello")
	source.AdvanceFrame()

	chars := source.AppendInputChars(nil)
	if string(chars) != "hello" {
		t.Errorf("chars = %q, want 'hello'", string(chars))
	}

	source.AdvanceFrame()
	chars = source.AppendInputChars(nil)
	if len(chars) != 0 {
		t.Errorf("chars should be empty, got %q", string(chars))
	}
}
