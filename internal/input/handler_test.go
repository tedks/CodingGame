package input

import (
	"testing"
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
