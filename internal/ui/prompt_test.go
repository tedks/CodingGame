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

// Tests for conversation panel features

func TestPromptPanel_MessageHistory(t *testing.T) {
	p := NewPromptPanel(800)

	// Initially no messages
	if p.MessageCount() != 0 {
		t.Errorf("MessageCount() = %d, want 0", p.MessageCount())
	}

	// Add user message
	p.AddUserMessage("Hello Claude")
	if p.MessageCount() != 1 {
		t.Errorf("MessageCount() = %d, want 1", p.MessageCount())
	}

	// Add assistant response
	p.SetResponseText("Hello! How can I help?")
	if p.MessageCount() != 2 {
		t.Errorf("MessageCount() = %d, want 2", p.MessageCount())
	}

	// Empty messages should not be added
	p.AddUserMessage("")
	p.SetResponseText("")
	if p.MessageCount() != 2 {
		t.Errorf("MessageCount() = %d, want 2 (empty messages ignored)", p.MessageCount())
	}
}

func TestPromptPanel_ClearHistory(t *testing.T) {
	p := NewPromptPanel(800)

	p.AddUserMessage("msg1")
	p.SetResponseText("response1")
	p.ClearHistory()

	if p.MessageCount() != 0 {
		t.Errorf("MessageCount() after ClearHistory = %d, want 0", p.MessageCount())
	}
}

func TestPromptPanel_MinimizeMaximize(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)

	// Initially not minimized
	if p.IsMinimized() {
		t.Error("expected not minimized initially")
	}

	// Add messages to grow panel
	p.AddUserMessage("test message")
	p.SetResponseText("test response")

	// Toggle minimize
	p.ToggleMinimized()
	if !p.IsMinimized() {
		t.Error("expected minimized after ToggleMinimized()")
	}

	// Run update to apply height change
	for i := 0; i < 100; i++ {
		p.Update()
	}
	if p.Height() != MinPanelHeight {
		t.Errorf("Height() when minimized = %d, want %d", p.Height(), MinPanelHeight)
	}

	// Toggle back
	p.ToggleMinimized()
	if p.IsMinimized() {
		t.Error("expected not minimized after second ToggleMinimized()")
	}
}

func TestPromptPanel_HeightGrowsWithMessages(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)

	initialHeight := p.Height()

	// Add several messages
	for i := 0; i < 5; i++ {
		p.AddUserMessage("User message that should increase panel height")
		p.SetResponseText("Assistant response with enough text to grow")
	}

	// Run updates to animate height
	for i := 0; i < 100; i++ {
		p.Update()
	}

	if p.Height() <= initialHeight {
		t.Errorf("Height() = %d, expected > %d after adding messages", p.Height(), initialHeight)
	}
}

func TestPromptPanel_HeightClamped(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)

	// Add many messages
	for i := 0; i < 50; i++ {
		p.AddUserMessage("Message " + string(rune('A'+i%26)))
		p.SetResponseText("Response with lots of text to fill space")
	}

	// Run updates
	for i := 0; i < 200; i++ {
		p.Update()
	}

	if p.Height() > MaxPanelHeight {
		t.Errorf("Height() = %d, should not exceed MaxPanelHeight %d", p.Height(), MaxPanelHeight)
	}
}

func TestPromptPanel_ContainsPoint(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.Update() // Set Y position

	// Point inside panel
	if !p.ContainsPoint(400, p.Y+30) {
		t.Error("ContainsPoint should return true for point inside panel")
	}

	// Point above panel
	if p.ContainsPoint(400, p.Y-10) {
		t.Error("ContainsPoint should return false for point above panel")
	}

	// Point to the left of panel
	if p.ContainsPoint(-10, p.Y+30) {
		t.Error("ContainsPoint should return false for point left of panel")
	}
}

func TestPromptPanel_DragHandle(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.Update()

	// Point on drag handle (top edge)
	if !p.IsOnDragHandle(400, p.Y+2) {
		t.Error("IsOnDragHandle should return true for point on drag handle")
	}

	// Point below drag handle
	if p.IsOnDragHandle(400, p.Y+20) {
		t.Error("IsOnDragHandle should return false for point below drag handle")
	}
}

func TestPromptPanel_DragResize(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.Update()

	initialHeight := p.Height()

	// Start drag
	p.StartDrag(p.Y)
	if !p.IsDragging() {
		t.Error("expected IsDragging() = true after StartDrag()")
	}

	// Drag upward (should increase height)
	p.UpdateDrag(p.Y - 100)

	if p.Height() <= initialHeight {
		t.Errorf("Height() = %d, expected > %d after dragging up", p.Height(), initialHeight)
	}

	// End drag
	p.EndDrag()
	if p.IsDragging() {
		t.Error("expected IsDragging() = false after EndDrag()")
	}
}

func TestPromptPanel_ScrollUpDown(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)

	// Add enough messages to enable scrolling
	for i := 0; i < 20; i++ {
		p.AddUserMessage("Long message to create scrollable content")
		p.SetResponseText("Response text that adds more content")
	}

	// Maximize panel height
	p.StartDrag(0)
	p.UpdateDrag(-500)
	p.EndDrag()

	// Scroll up
	p.ScrollUp()
	// ScrollOffset should increase (we're scrolling up in history)

	// Scroll down
	p.ScrollDown()
	// ScrollOffset should decrease
}

func TestPromptPanel_HandleClick(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.AddUserMessage("test") // Add message so panel isn't minimal
	p.Update()

	// Click outside panel should not be handled
	if p.HandleClick(400, 0) {
		t.Error("HandleClick should return false for click outside panel")
	}

	// Click inside panel should be handled
	if !p.HandleClick(400, p.Y+30) {
		t.Error("HandleClick should return true for click inside panel")
	}
}

func TestPromptPanel_HandleScroll(t *testing.T) {
	p := NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.Update()

	// Scroll outside panel
	if p.HandleScroll(400, 0, 0, 1) {
		t.Error("HandleScroll should return false for scroll outside panel")
	}

	// Scroll inside panel
	if !p.HandleScroll(400, p.Y+30, 0, 1) {
		t.Error("HandleScroll should return true for scroll inside panel")
	}
}
