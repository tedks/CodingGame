package ui

import (
	"testing"
)

func TestNewPromptPanel(t *testing.T) {
	p := NewPromptPanel(800)

	if p.Width != 800 {
		t.Errorf("Width = %d, want 800", p.Width)
	}
	if p.Height() != MinPanelHeight {
		t.Errorf("Height() = %d, want %d", p.Height(), MinPanelHeight)
	}
	if p.State != PromptStateIdle {
		t.Errorf("initial State = %v, want PromptStateIdle", p.State)
	}
	if p.Mode != "NORMAL" {
		t.Errorf("initial Mode = %q, want %q", p.Mode, "NORMAL")
	}
	if p.Text != "" {
		t.Errorf("initial Text = %q, want empty", p.Text)
	}
}

func TestPromptPanel_SetPosition(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.SetPosition(100, 200) // Y is now calculated dynamically

	if p.X != 100 {
		t.Errorf("X = %d, want 100", p.X)
	}
	// Y is calculated based on screen height after Update()
	p.Update()
	expectedY := 600 - p.Height() // screenHeight - currentHeight
	if p.Y != expectedY {
		t.Errorf("Y = %d, want %d (calculated from screen height)", p.Y, expectedY)
	}
}

func TestPromptPanel_SetSize(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetSize(1024, 80) // Height parameter is ignored (dynamic)

	if p.Width != 1024 {
		t.Errorf("Width = %d, want 1024", p.Width)
	}
	// Height is dynamic and not set by SetSize
	if p.Height() != MinPanelHeight {
		t.Errorf("Height() = %d, want %d (height is dynamic)", p.Height(), MinPanelHeight)
	}
}

func TestPromptPanel_SetText(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetText("hello world")

	if p.Text != "hello world" {
		t.Errorf("Text = %q, want %q", p.Text, "hello world")
	}
}

func TestPromptPanel_SetMode(t *testing.T) {
	p := NewPromptPanel(800)

	tests := []struct {
		mode        string
		expectColor bool // Just check that color is set
	}{
		{"NORMAL", true},
		{"INSERT", true},
		{"VISUAL", true},
	}

	for _, tc := range tests {
		p.SetMode(tc.mode)
		if p.Mode != tc.mode {
			t.Errorf("Mode = %q, want %q", p.Mode, tc.mode)
		}
		// Just verify ModeColor is non-zero (colors are set)
		if p.ModeColor.A == 0 {
			t.Errorf("ModeColor alpha is 0 for mode %q", tc.mode)
		}
	}
}

func TestPromptPanel_Focus(t *testing.T) {
	p := NewPromptPanel(800)

	// Initially idle
	if p.IsFocused() {
		t.Error("expected unfocused initially")
	}

	// Focus
	p.Focus()
	if !p.IsFocused() {
		t.Error("expected focused after Focus()")
	}
	if p.State != PromptStateActive {
		t.Errorf("State = %v, want PromptStateActive", p.State)
	}

	// Unfocus
	p.Unfocus()
	if p.IsFocused() {
		t.Error("expected unfocused after Unfocus()")
	}
	if p.State != PromptStateIdle {
		t.Errorf("State = %v, want PromptStateIdle", p.State)
	}
}

func TestPromptPanel_Clear(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetText("some text")
	p.Clear()

	if p.Text != "" {
		t.Errorf("Text after Clear() = %q, want empty", p.Text)
	}
}

func TestPromptPanel_Submit(t *testing.T) {
	p := NewPromptPanel(800)

	var submittedText string
	p.OnSubmit = func(text string) {
		submittedText = text
	}

	// Submit with text
	p.SetText("test command")
	p.Submit()

	if submittedText != "test command" {
		t.Errorf("submitted text = %q, want %q", submittedText, "test command")
	}
	if p.State != PromptStateProcessing {
		t.Errorf("State after Submit = %v, want PromptStateProcessing", p.State)
	}
}

func TestPromptPanel_Submit_EmptyText(t *testing.T) {
	p := NewPromptPanel(800)

	callbackCalled := false
	p.OnSubmit = func(text string) {
		callbackCalled = true
	}

	// Set initial state to Active
	p.SetState(PromptStateActive)

	// Submit with empty text should not call callback or change state
	p.SetText("")
	p.Submit()

	if callbackCalled {
		t.Error("OnSubmit should not be called with empty text")
	}
	if p.State != PromptStateActive {
		t.Errorf("State should remain Active with empty text, got %v", p.State)
	}
}

func TestPromptPanel_Cancel(t *testing.T) {
	p := NewPromptPanel(800)

	cancelCalled := false
	p.OnCancel = func() {
		cancelCalled = true
	}

	p.SetText("some text")
	p.Focus()
	p.Cancel()

	if !cancelCalled {
		t.Error("OnCancel should be called")
	}
	if p.Text != "" {
		t.Errorf("Text after Cancel = %q, want empty", p.Text)
	}
	if p.State != PromptStateIdle {
		t.Errorf("State after Cancel = %v, want PromptStateIdle", p.State)
	}
}

func TestPromptPanel_SetState(t *testing.T) {
	p := NewPromptPanel(800)

	tests := []struct {
		state PromptState
	}{
		{PromptStateIdle},
		{PromptStateActive},
		{PromptStateProcessing},
	}

	for _, tc := range tests {
		p.SetState(tc.state)
		if p.State != tc.state {
			t.Errorf("State = %v, want %v", p.State, tc.state)
		}
	}
}

func TestPromptState_Constants(t *testing.T) {
	// Verify states are distinct
	states := []PromptState{PromptStateIdle, PromptStateActive, PromptStateProcessing}
	seen := make(map[PromptState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate PromptState value: %d", s)
		}
		seen[s] = true
	}
}
