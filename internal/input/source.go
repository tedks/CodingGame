package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputSource abstracts keyboard and mouse input for testing.
// Production code uses EbitenInputSource (the default), while tests
// can inject TestInputSource to simulate user input.
type InputSource interface {
	// Keyboard methods
	JustPressedKeys() []ebiten.Key
	IsKeyPressed(key ebiten.Key) bool
	IsKeyJustPressed(key ebiten.Key) bool
	AppendInputChars(chars []rune) []rune

	// Mouse methods
	CursorPosition() (x, y int)
	IsMouseButtonPressed(button ebiten.MouseButton) bool
	IsMouseButtonJustPressed(button ebiten.MouseButton) bool
	Wheel() (xoff, yoff float64)
}

// EbitenInputSource is the real input implementation using Ebitengine.
type EbitenInputSource struct{}

// JustPressedKeys returns all keys that were just pressed this frame.
func (e *EbitenInputSource) JustPressedKeys() []ebiten.Key {
	return inpututil.AppendJustPressedKeys(nil)
}

// IsKeyPressed returns true if the key is currently held down.
func (e *EbitenInputSource) IsKeyPressed(key ebiten.Key) bool {
	return ebiten.IsKeyPressed(key)
}

// IsKeyJustPressed returns true if the key was just pressed this frame.
func (e *EbitenInputSource) IsKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

// AppendInputChars appends typed characters to the slice and returns it.
func (e *EbitenInputSource) AppendInputChars(chars []rune) []rune {
	return ebiten.AppendInputChars(chars)
}

// CursorPosition returns the current mouse cursor position.
func (e *EbitenInputSource) CursorPosition() (x, y int) {
	return ebiten.CursorPosition()
}

// IsMouseButtonPressed returns true if the mouse button is currently held down.
func (e *EbitenInputSource) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(button)
}

// IsMouseButtonJustPressed returns true if the mouse button was just pressed this frame.
func (e *EbitenInputSource) IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	return inpututil.IsMouseButtonJustPressed(button)
}

// Wheel returns the mouse wheel scroll amounts.
func (e *EbitenInputSource) Wheel() (xoff, yoff float64) {
	return ebiten.Wheel()
}

// DefaultSource is the package-level default input source.
// Use this in production code; tests can override it via SetInputSource.
var DefaultSource InputSource = &EbitenInputSource{}
