package testutil

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
)

// InputEventType represents the type of input event.
type InputEventType int

const (
	// KeyPress indicates a key just pressed this frame
	KeyPress InputEventType = iota
	// KeyRelease indicates a key just released this frame
	KeyRelease
	// KeyHold indicates a key held down for multiple frames
	KeyHold
	// CharInput indicates character input (for text fields)
	CharInput
	// MouseMove indicates mouse cursor movement
	MouseMove
	// MouseButtonPress indicates a mouse button just pressed
	MouseButtonPress
	// MouseButtonRelease indicates a mouse button just released
	MouseButtonRelease
	// MouseWheel indicates mouse wheel scroll
	MouseWheel
)

// InputEvent represents a single input event to be played back.
type InputEvent struct {
	Type     InputEventType
	Key      ebiten.Key
	Char     rune
	Duration int // frames to hold (for KeyHold)
	X, Y     int // mouse position (for MouseMove)
	Button   ebiten.MouseButton
	WheelX   float64
	WheelY   float64
}

// TestInputSource implements InputSource for testing.
// It allows queuing input events that are played back frame-by-frame.
type TestInputSource struct {
	mu sync.Mutex

	// Event queue
	events []InputEvent

	// Current frame state
	justPressedKeys     map[ebiten.Key]bool
	pressedKeys         map[ebiten.Key]bool
	justReleasedKeys    map[ebiten.Key]bool
	inputChars          []rune
	cursorX, cursorY    int
	pressedButtons      map[ebiten.MouseButton]bool
	justPressedButtons  map[ebiten.MouseButton]bool
	justReleasedButtons map[ebiten.MouseButton]bool
	wheelX, wheelY      float64

	// Held key tracking (key -> frames remaining)
	heldKeys map[ebiten.Key]int
}

// NewTestInputSource creates a new test input source.
func NewTestInputSource() *TestInputSource {
	return &TestInputSource{
		justPressedKeys:     make(map[ebiten.Key]bool),
		pressedKeys:         make(map[ebiten.Key]bool),
		justReleasedKeys:    make(map[ebiten.Key]bool),
		pressedButtons:      make(map[ebiten.MouseButton]bool),
		justPressedButtons:  make(map[ebiten.MouseButton]bool),
		justReleasedButtons: make(map[ebiten.MouseButton]bool),
		heldKeys:            make(map[ebiten.Key]int),
	}
}

// QueueEvents adds events to the queue.
func (t *TestInputSource) QueueEvents(events ...InputEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, events...)
}

// QueueKeyPress queues a single key press event.
func (t *TestInputSource) QueueKeyPress(key ebiten.Key) {
	t.QueueEvents(InputEvent{Type: KeyPress, Key: key})
}

// QueueKeyHold queues a key hold event for the specified number of frames.
func (t *TestInputSource) QueueKeyHold(key ebiten.Key, frames int) {
	t.QueueEvents(InputEvent{Type: KeyHold, Key: key, Duration: frames})
}

// QueueCharInput queues character input.
func (t *TestInputSource) QueueCharInput(chars ...rune) {
	for _, c := range chars {
		t.QueueEvents(InputEvent{Type: CharInput, Char: c})
	}
}

// QueueTextInput queues a string as character input.
func (t *TestInputSource) QueueTextInput(text string) {
	t.QueueCharInput([]rune(text)...)
}

// QueueMouseMove queues a mouse movement event.
func (t *TestInputSource) QueueMouseMove(x, y int) {
	t.QueueEvents(InputEvent{Type: MouseMove, X: x, Y: y})
}

// QueueMouseClick queues a mouse button click.
func (t *TestInputSource) QueueMouseClick(button ebiten.MouseButton) {
	t.QueueEvents(InputEvent{Type: MouseButtonPress, Button: button})
}

// QueueMouseRelease queues a mouse button release.
func (t *TestInputSource) QueueMouseRelease(button ebiten.MouseButton) {
	t.QueueEvents(InputEvent{Type: MouseButtonRelease, Button: button})
}

// QueueMouseClickAndRelease queues a complete mouse click (press then release).
// This simulates a full click cycle that MapView needs to detect tile selection.
func (t *TestInputSource) QueueMouseClickAndRelease(button ebiten.MouseButton) {
	t.QueueEvents(
		InputEvent{Type: MouseButtonPress, Button: button},
		InputEvent{Type: MouseButtonRelease, Button: button},
	)
}

// QueueMouseWheel queues a mouse wheel scroll.
func (t *TestInputSource) QueueMouseWheel(xoff, yoff float64) {
	t.QueueEvents(InputEvent{Type: MouseWheel, WheelX: xoff, WheelY: yoff})
}

// AdvanceFrame processes events for the current frame and advances to the next.
// This should be called at the beginning of each Update() call.
func (t *TestInputSource) AdvanceFrame() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear frame-specific state
	t.justPressedKeys = make(map[ebiten.Key]bool)
	t.justReleasedKeys = make(map[ebiten.Key]bool)
	t.inputChars = nil
	t.justPressedButtons = make(map[ebiten.MouseButton]bool)
	t.justReleasedButtons = make(map[ebiten.MouseButton]bool)
	t.wheelX = 0
	t.wheelY = 0

	// Process held keys - decrement duration and release if expired
	for key, framesLeft := range t.heldKeys {
		if framesLeft <= 1 {
			delete(t.heldKeys, key)
			delete(t.pressedKeys, key)
			t.justReleasedKeys[key] = true
		} else {
			t.heldKeys[key] = framesLeft - 1
		}
	}

	// Process events for this frame
	// Collect all events that should be processed this frame
	var remainingEvents []InputEvent
	for _, event := range t.events {
		switch event.Type {
		case KeyPress:
			if !t.pressedKeys[event.Key] {
				t.justPressedKeys[event.Key] = true
			}
			t.pressedKeys[event.Key] = true
			// Auto-release next frame
			t.heldKeys[event.Key] = 1

		case KeyRelease:
			delete(t.pressedKeys, event.Key)
			delete(t.heldKeys, event.Key)
			t.justReleasedKeys[event.Key] = true

		case KeyHold:
			if !t.pressedKeys[event.Key] {
				t.justPressedKeys[event.Key] = true
			}
			t.pressedKeys[event.Key] = true
			t.heldKeys[event.Key] = event.Duration

		case CharInput:
			t.inputChars = append(t.inputChars, event.Char)

		case MouseMove:
			t.cursorX = event.X
			t.cursorY = event.Y

		case MouseButtonPress:
			if !t.pressedButtons[event.Button] {
				t.justPressedButtons[event.Button] = true
			}
			t.pressedButtons[event.Button] = true

		case MouseButtonRelease:
			delete(t.pressedButtons, event.Button)
			t.justReleasedButtons[event.Button] = true

		case MouseWheel:
			t.wheelX = event.WheelX
			t.wheelY = event.WheelY

		default:
			// Unknown event type, keep it
			remainingEvents = append(remainingEvents, event)
		}
	}

	t.events = remainingEvents
}

// HasPendingEvents returns true if there are events waiting to be processed.
func (t *TestInputSource) HasPendingEvents() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.events) > 0 || len(t.heldKeys) > 0
}

// Clear removes all events and resets state.
func (t *TestInputSource) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = nil
	t.justPressedKeys = make(map[ebiten.Key]bool)
	t.pressedKeys = make(map[ebiten.Key]bool)
	t.justReleasedKeys = make(map[ebiten.Key]bool)
	t.inputChars = nil
	t.cursorX = 0
	t.cursorY = 0
	t.pressedButtons = make(map[ebiten.MouseButton]bool)
	t.justPressedButtons = make(map[ebiten.MouseButton]bool)
	t.justReleasedButtons = make(map[ebiten.MouseButton]bool)
	t.heldKeys = make(map[ebiten.Key]int)
	t.wheelX = 0
	t.wheelY = 0
}

// InputSource interface implementation

// JustPressedKeys returns all keys that were just pressed this frame.
func (t *TestInputSource) JustPressedKeys() []ebiten.Key {
	t.mu.Lock()
	defer t.mu.Unlock()
	keys := make([]ebiten.Key, 0, len(t.justPressedKeys))
	for key := range t.justPressedKeys {
		keys = append(keys, key)
	}
	return keys
}

// IsKeyPressed returns true if the key is currently held down.
func (t *TestInputSource) IsKeyPressed(key ebiten.Key) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pressedKeys[key]
}

// IsKeyJustPressed returns true if the key was just pressed this frame.
func (t *TestInputSource) IsKeyJustPressed(key ebiten.Key) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.justPressedKeys[key]
}

// AppendInputChars appends typed characters to the slice and returns it.
func (t *TestInputSource) AppendInputChars(chars []rune) []rune {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append(chars, t.inputChars...)
}

// CursorPosition returns the current mouse cursor position.
func (t *TestInputSource) CursorPosition() (x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cursorX, t.cursorY
}

// IsMouseButtonPressed returns true if the mouse button is currently held down.
func (t *TestInputSource) IsMouseButtonPressed(button ebiten.MouseButton) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pressedButtons[button]
}

// IsMouseButtonJustPressed returns true if the mouse button was just pressed.
func (t *TestInputSource) IsMouseButtonJustPressed(button ebiten.MouseButton) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.justPressedButtons[button]
}

// Wheel returns the mouse wheel scroll amounts.
func (t *TestInputSource) Wheel() (xoff, yoff float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.wheelX, t.wheelY
}

// Ensure TestInputSource implements InputSource
var _ input.InputSource = (*TestInputSource)(nil)
