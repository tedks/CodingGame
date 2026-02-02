package input

import (
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/testutil"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler()

	if h.Mode() != ModeNormal {
		t.Errorf("initial Mode() = %v, want %v", h.Mode(), ModeNormal)
	}
	if h.Focus() != FocusMap {
		t.Errorf("initial Focus() = %v, want %v", h.Focus(), FocusMap)
	}
	if h.View() != ViewMap {
		t.Errorf("initial View() = %v, want %v", h.View(), ViewMap)
	}
	if h.TextBuffer() != "" {
		t.Errorf("initial TextBuffer() = %q, want empty string", h.TextBuffer())
	}
}

func TestHandler_SetMode(t *testing.T) {
	h := NewHandler()

	var receivedMode Mode
	h.OnModeChange(func(mode Mode) {
		receivedMode = mode
	})

	h.SetMode(ModeInsert)

	if h.Mode() != ModeInsert {
		t.Errorf("Mode() = %v, want %v", h.Mode(), ModeInsert)
	}
	if receivedMode != ModeInsert {
		t.Errorf("callback received mode %v, want %v", receivedMode, ModeInsert)
	}
}

func TestHandler_SetMode_NoCallbackOnSameMode(t *testing.T) {
	h := NewHandler()

	callCount := 0
	h.OnModeChange(func(mode Mode) {
		callCount++
	})

	// Setting to same mode shouldn't trigger callback
	h.SetMode(ModeNormal)

	if callCount != 0 {
		t.Errorf("callback called %d times, want 0 when mode unchanged", callCount)
	}
}

func TestHandler_SetFocus(t *testing.T) {
	h := NewHandler()

	var receivedFocus FocusArea
	h.OnFocusChange(func(focus FocusArea) {
		receivedFocus = focus
	})

	h.SetFocus(FocusPrompt)

	if h.Focus() != FocusPrompt {
		t.Errorf("Focus() = %v, want %v", h.Focus(), FocusPrompt)
	}
	if receivedFocus != FocusPrompt {
		t.Errorf("callback received focus %v, want %v", receivedFocus, FocusPrompt)
	}
}

func TestHandler_SetView(t *testing.T) {
	h := NewHandler()

	var receivedView ViewNumber
	h.OnViewChange(func(view ViewNumber) {
		receivedView = view
	})

	h.SetView(ViewBuilding)

	if h.View() != ViewBuilding {
		t.Errorf("View() = %v, want %v", h.View(), ViewBuilding)
	}
	if receivedView != ViewBuilding {
		t.Errorf("callback received view %v, want %v", receivedView, ViewBuilding)
	}
}

func TestHandler_SetTextBuffer(t *testing.T) {
	h := NewHandler()

	var receivedText string
	h.OnTextChange(func(text string) {
		receivedText = text
	})

	h.SetTextBuffer("hello")

	if h.TextBuffer() != "hello" {
		t.Errorf("TextBuffer() = %q, want %q", h.TextBuffer(), "hello")
	}
	if receivedText != "hello" {
		t.Errorf("callback received text %q, want %q", receivedText, "hello")
	}
}

func TestHandler_ClearTextBuffer(t *testing.T) {
	h := NewHandler()
	h.SetTextBuffer("hello")

	h.ClearTextBuffer()

	if h.TextBuffer() != "" {
		t.Errorf("TextBuffer() after clear = %q, want empty string", h.TextBuffer())
	}
}

func TestHandler_SetBindingStyle(t *testing.T) {
	h := NewHandler()

	// Initially vim
	if h.Bindings().Style() != StyleVim {
		t.Errorf("initial style = %v, want %v", h.Bindings().Style(), StyleVim)
	}

	// Switch to emacs
	h.SetBindingStyle(StyleEmacs)

	if h.Bindings().Style() != StyleEmacs {
		t.Errorf("style after SetBindingStyle = %v, want %v", h.Bindings().Style(), StyleEmacs)
	}
}

func TestHandler_OnAction(t *testing.T) {
	h := NewHandler()

	var receivedAction Action
	h.OnAction(func(action Action) {
		receivedAction = action
	})

	// Simulate receiving an action (we can't easily test Update() without mocking ebiten)
	// But we can verify the callback is set correctly by checking it's not nil
	// The actual input handling is tested indirectly through integration tests

	// For now, just verify the handler accepts the callback
	if h.onAction == nil {
		t.Error("expected onAction callback to be set")
	}

	// Suppress unused variable warning
	_ = receivedAction
}

func TestHandler_HandleBuiltInAction_ModeChanges(t *testing.T) {
	h := NewHandler()

	// Test entering insert mode
	h.handleBuiltInAction(ActionEnterInsert)
	if h.Mode() != ModeInsert {
		t.Errorf("after ActionEnterInsert, Mode() = %v, want %v", h.Mode(), ModeInsert)
	}

	// Test exiting to normal mode
	h.handleBuiltInAction(ActionExitMode)
	if h.Mode() != ModeNormal {
		t.Errorf("after ActionExitMode, Mode() = %v, want %v", h.Mode(), ModeNormal)
	}

	// Test entering visual mode
	h.handleBuiltInAction(ActionEnterVisual)
	if h.Mode() != ModeVisual {
		t.Errorf("after ActionEnterVisual, Mode() = %v, want %v", h.Mode(), ModeVisual)
	}
}

func TestHandler_HandleBuiltInAction_FocusChanges(t *testing.T) {
	h := NewHandler()

	// Test focus next
	h.handleBuiltInAction(ActionFocusNext)
	if h.Focus() != FocusPrompt {
		t.Errorf("after ActionFocusNext from Map, Focus() = %v, want %v", h.Focus(), FocusPrompt)
	}

	// Test focus prev
	h.handleBuiltInAction(ActionFocusPrev)
	if h.Focus() != FocusMap {
		t.Errorf("after ActionFocusPrev from Prompt, Focus() = %v, want %v", h.Focus(), FocusMap)
	}

	// Test focus prompt (also enters insert mode)
	h.handleBuiltInAction(ActionFocusPrompt)
	if h.Focus() != FocusPrompt {
		t.Errorf("after ActionFocusPrompt, Focus() = %v, want %v", h.Focus(), FocusPrompt)
	}
	if h.Mode() != ModeInsert {
		t.Errorf("after ActionFocusPrompt, Mode() = %v, want %v", h.Mode(), ModeInsert)
	}

	// Test focus map (also enters normal mode)
	h.handleBuiltInAction(ActionFocusMap)
	if h.Focus() != FocusMap {
		t.Errorf("after ActionFocusMap, Focus() = %v, want %v", h.Focus(), FocusMap)
	}
	if h.Mode() != ModeNormal {
		t.Errorf("after ActionFocusMap, Mode() = %v, want %v", h.Mode(), ModeNormal)
	}
}

func TestHandler_HandleBuiltInAction_ViewChanges(t *testing.T) {
	h := NewHandler()

	viewActions := []struct {
		action   Action
		expected ViewNumber
	}{
		{ActionView1, ViewMap},
		{ActionView2, ViewBuilding},
		{ActionView3, ViewUnit},
		{ActionView4, ViewTech},
		{ActionView5, ViewMission},
	}

	for _, tc := range viewActions {
		h.handleBuiltInAction(tc.action)
		if h.View() != tc.expected {
			t.Errorf("after %v, View() = %v, want %v", tc.action, h.View(), tc.expected)
		}
	}
}

func TestHandler_CancelPrompt(t *testing.T) {
	h := NewHandler()

	// Enter insert mode
	h.SetMode(ModeInsert)
	h.SetFocus(FocusPrompt)

	// Cancel should exit to normal mode
	h.handleBuiltInAction(ActionCancelPrompt)

	if h.Mode() != ModeNormal {
		t.Errorf("after ActionCancelPrompt, Mode() = %v, want %v", h.Mode(), ModeNormal)
	}
}

func TestHandler_Bindings_NotNil(t *testing.T) {
	h := NewHandler()

	if h.Bindings() == nil {
		t.Error("Bindings() should not return nil")
	}
}

// ============================================================================
// P0 Tests: Ship Blockers
// ============================================================================

// TestHandler_TextBuffer_UTF8Backspace verifies that backspace correctly removes
// runes (not bytes) from the text buffer, handling multi-byte characters properly.
func TestHandler_TextBuffer_UTF8Backspace(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		expected string
	}{
		{"ascii", "hello", "hell"},
		{"accented", "café", "caf"},  // é is 2 bytes
		{"emoji", "hi👋", "hi"},      // wave emoji is 4 bytes
		{"chinese", "你好", "你"},      // Chinese characters are 3 bytes each
		{"mixed", "a🚀b", "a🚀"},     // mixed ASCII and emoji
		{"empty", "", ""},            // edge case: empty buffer
		{"single_ascii", "a", ""},    // edge case: single ASCII char
		{"single_emoji", "😀", ""},   // edge case: single emoji
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler()
			source := testutil.NewTestInputSource()
			h.SetInputSource(source)
			h.SetMode(ModeInsert)
			h.SetTextBuffer(tc.initial)

			// Queue backspace and advance
			source.QueueKeyPress(ebiten.KeyBackspace)
			source.AdvanceFrame()
			h.Update()

			if h.TextBuffer() != tc.expected {
				t.Errorf("got %q (len %d bytes), want %q (len %d bytes)",
					h.TextBuffer(), len(h.TextBuffer()),
					tc.expected, len(tc.expected))
			}
		})
	}
}

// TestHandler_Update_KeyPressTriggersAction verifies that pressing a key
// triggers the corresponding action callback.
func TestHandler_Update_KeyPressTriggersAction(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	var received []Action
	h.OnAction(func(a Action) {
		received = append(received, a)
	})

	// Press 'j' in Normal mode - should trigger MoveDown
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	h.Update()

	if len(received) != 1 {
		t.Fatalf("got %d actions, want 1", len(received))
	}
	if received[0] != ActionMoveDown {
		t.Errorf("got %v, want ActionMoveDown", received[0])
	}
}

// TestHandler_Update_ModeChangeFromInput verifies that pressing 'i' enters insert mode.
func TestHandler_Update_ModeChangeFromInput(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	// Press 'i' to enter insert mode
	source.QueueKeyPress(ebiten.KeyI)
	source.AdvanceFrame()
	h.Update()

	if h.Mode() != ModeInsert {
		t.Errorf("got mode %v, want ModeInsert", h.Mode())
	}
}

// TestHandler_Update_ModifierDetection verifies that modifier keys are detected.
func TestHandler_Update_ModifierDetection(t *testing.T) {
	h := NewHandler()
	h.SetBindingStyle(StyleEmacs)
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	var received []Action
	h.OnAction(func(a Action) {
		received = append(received, a)
	})

	// Ctrl+P should move up in emacs mode
	source.QueueKeyHold(ebiten.KeyControl, 2)
	source.QueueKeyPress(ebiten.KeyP)
	source.AdvanceFrame()
	h.Update()

	if len(received) != 1 {
		t.Fatalf("got %d actions, want 1", len(received))
	}
	if received[0] != ActionMoveUp {
		t.Errorf("got %v, want ActionMoveUp", received[0])
	}
}

// TestHandler_CallbackReentrancy_NoDeadlock verifies that calling SetMode
// from within a mode change callback doesn't cause a deadlock.
func TestHandler_CallbackReentrancy_NoDeadlock(t *testing.T) {
	h := NewHandler()

	calls := 0
	h.OnModeChange(func(m Mode) {
		calls++
		if m == ModeInsert && calls < 3 {
			h.SetMode(ModeNormal) // Re-entrant call
		}
	})

	done := make(chan bool, 1)
	go func() {
		h.SetMode(ModeInsert)
		done <- true
	}()

	select {
	case <-done:
		// OK - didn't deadlock
	case <-time.After(time.Second):
		t.Fatal("callback re-entrancy caused deadlock")
	}
}

// TestHandler_CallbackReentrancy_StateConsistent verifies that re-entrant
// state changes result in consistent final state.
func TestHandler_CallbackReentrancy_StateConsistent(t *testing.T) {
	h := NewHandler()

	var finalMode Mode
	h.OnModeChange(func(m Mode) {
		if m == ModeInsert {
			h.SetMode(ModeVisual) // Re-entrant change
		}
		finalMode = h.Mode()
	})

	h.SetMode(ModeInsert)

	// Final state should be Visual (last SetMode wins)
	if finalMode != ModeVisual {
		t.Errorf("final mode %v, want ModeVisual", finalMode)
	}
}

// ============================================================================
// P1 Tests: Likely Bugs
// ============================================================================

// TestHandler_IsAction verifies that IsAction correctly detects triggered actions.
func TestHandler_IsAction(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()

	if !h.IsAction(ActionMoveDown) {
		t.Error("IsAction(MoveDown) should be true after 'j' press")
	}
	if h.IsAction(ActionMoveUp) {
		t.Error("IsAction(MoveUp) should be false")
	}
}

// TestHandler_IsActionHeld verifies that IsActionHeld correctly detects held keys.
func TestHandler_IsActionHeld(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	// Hold 'j' for 3 frames
	source.QueueKeyHold(ebiten.KeyJ, 3)
	source.AdvanceFrame()

	// Check that key is held for the duration
	for i := 0; i < 3; i++ {
		if !h.IsActionHeld(ActionMoveDown) {
			t.Errorf("frame %d: IsActionHeld should be true", i)
		}
		source.AdvanceFrame()
	}

	// After hold duration, should be released
	if h.IsActionHeld(ActionMoveDown) {
		t.Error("IsActionHeld should be false after release")
	}
}

// TestHandler_Update_MultipleKeysSameFrame verifies that multiple keys
// pressed in the same frame all trigger their respective actions.
func TestHandler_Update_MultipleKeysSameFrame(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	var received []Action
	h.OnAction(func(a Action) {
		received = append(received, a)
	})

	// Press both 'j' and 'k' in same frame
	source.QueueEvents(
		testutil.InputEvent{Type: testutil.KeyPress, Key: ebiten.KeyJ},
		testutil.InputEvent{Type: testutil.KeyPress, Key: ebiten.KeyK},
	)
	source.AdvanceFrame()
	h.Update()

	// Both actions should fire
	if len(received) != 2 {
		t.Errorf("got %d actions, want 2", len(received))
	}

	// Check that both MoveDown and MoveUp are present (order may vary)
	hasDown := false
	hasUp := false
	for _, a := range received {
		if a == ActionMoveDown {
			hasDown = true
		}
		if a == ActionMoveUp {
			hasUp = true
		}
	}
	if !hasDown {
		t.Error("expected ActionMoveDown in received actions")
	}
	if !hasUp {
		t.Error("expected ActionMoveUp in received actions")
	}
}

// ============================================================================
// P2 Tests: Robustness
// ============================================================================

// TestHandler_ConcurrentAccess verifies that concurrent reads and writes
// to the handler don't cause race conditions.
func TestHandler_ConcurrentAccess(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	var wg sync.WaitGroup

	// Concurrent mode changes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.SetMode(ModeInsert)
				_ = h.Mode()
				h.SetMode(ModeNormal)
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = h.Mode()
				_ = h.Focus()
				_ = h.View()
				_ = h.TextBuffer()
			}
		}()
	}

	wg.Wait()
	// Test passes if no race detector complaints
}

// TestHandler_KeySpam verifies that rapidly pressing a key produces
// the expected number of action events.
func TestHandler_KeySpam(t *testing.T) {
	h := NewHandler()
	source := testutil.NewTestInputSource()
	h.SetInputSource(source)

	actionCount := 0
	h.OnAction(func(a Action) {
		actionCount++
	})

	// Spam 'j' key across many frames
	for i := 0; i < 100; i++ {
		source.QueueKeyPress(ebiten.KeyJ)
		source.AdvanceFrame()
		h.Update()
	}

	if actionCount != 100 {
		t.Errorf("got %d actions, want 100", actionCount)
	}
}
