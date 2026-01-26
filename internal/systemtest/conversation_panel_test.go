package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// Conversation panel tests verify the interactive conversation UI:
// - Message display and scrolling
// - Drag to resize
// - Minimize/maximize toggle
// - Input isolation (map doesn't scroll when panel is active)

// testConversationPanel creates a prompt panel configured for testing.
func testConversationPanel() *ui.PromptPanel {
	p := ui.NewPromptPanel(800)
	p.SetScreenHeight(600)
	p.Update() // Initialize Y position
	return p
}

// Tests for conversation panel interaction

func testConversationPanelMessageHistory(t *testing.T) {
	p := testConversationPanel()

	// Verify initial state
	if p.MessageCount() != 0 {
		t.Errorf("expected 0 messages initially, got %d", p.MessageCount())
	}

	// Submit a message (simulates user typing and pressing enter)
	p.SetText("Hello Claude")
	p.Submit()

	// Should have 1 message (user message added by Submit)
	if p.MessageCount() != 1 {
		t.Errorf("expected 1 message after submit, got %d", p.MessageCount())
	}

	// Add response (simulates Claude responding)
	p.SetResponseText("Hello! How can I help you today?")

	// Should have 2 messages
	if p.MessageCount() != 2 {
		t.Errorf("expected 2 messages after response, got %d", p.MessageCount())
	}
}

func testConversationPanelDragResize(t *testing.T) {
	p := testConversationPanel()
	p.AddUserMessage("test message")
	p.SetResponseText("test response")

	// Run a few updates to let height grow
	for i := 0; i < 50; i++ {
		p.Update()
	}

	initialHeight := p.Height()

	// Start drag at the top of the panel (drag handle)
	dragY := p.Y + ui.DragHandleHeight/2
	p.StartDrag(dragY)

	if !p.IsDragging() {
		t.Error("expected IsDragging() = true after StartDrag")
	}

	// Drag upward to increase height
	p.UpdateDrag(dragY - 100)

	newHeight := p.Height()
	if newHeight <= initialHeight {
		t.Errorf("expected height to increase when dragging up: was %d, now %d", initialHeight, newHeight)
	}

	// End drag
	p.EndDrag()

	if p.IsDragging() {
		t.Error("expected IsDragging() = false after EndDrag")
	}

	// Height should be preserved (user's preferred height)
	finalHeight := p.Height()
	if finalHeight != newHeight {
		t.Errorf("height should be preserved after EndDrag: was %d, now %d", newHeight, finalHeight)
	}
}

func testConversationPanelMinimizeRestore(t *testing.T) {
	p := testConversationPanel()

	// Add some messages to grow the panel
	p.AddUserMessage("message 1")
	p.SetResponseText("response 1")
	p.AddUserMessage("message 2")
	p.SetResponseText("response 2")

	// Let height grow
	for i := 0; i < 100; i++ {
		p.Update()
	}

	expandedHeight := p.Height()
	if expandedHeight <= ui.MinPanelHeight {
		t.Errorf("expected panel to grow with messages, got height %d", expandedHeight)
	}

	// Click minimize button
	minimizeX := p.X + p.Width - 50 // Approximate button position
	minimizeY := p.Y + 4
	if !p.HandleClick(minimizeX, minimizeY) {
		t.Error("expected HandleClick to return true for minimize button")
	}

	// Let animation complete
	for i := 0; i < 100; i++ {
		p.Update()
	}

	// Should be minimized
	if !p.IsMinimized() {
		t.Error("expected IsMinimized() = true after clicking minimize")
	}
	if p.Height() != ui.MinPanelHeight {
		t.Errorf("expected height = %d when minimized, got %d", ui.MinPanelHeight, p.Height())
	}

	// Click again to restore
	p.HandleClick(minimizeX, p.Y+4) // Y changed after minimize

	// Let animation complete
	for i := 0; i < 100; i++ {
		p.Update()
	}

	// Should be restored
	if p.IsMinimized() {
		t.Error("expected IsMinimized() = false after clicking again")
	}
}

func testConversationPanelScrolling(t *testing.T) {
	p := testConversationPanel()

	// Add many messages to create scrollable content
	for i := 0; i < 20; i++ {
		p.AddUserMessage("This is a long message that takes up space")
		p.SetResponseText("This is a detailed response with lots of text")
	}

	// Maximize the panel via drag
	p.StartDrag(p.Y)
	p.UpdateDrag(p.Y - 400)
	p.EndDrag()

	// Let height stabilize
	for i := 0; i < 50; i++ {
		p.Update()
	}

	// Mouse is inside panel
	mouseX := p.Width / 2
	mouseY := p.Y + p.Height()/2

	// Scroll up (positive deltaY)
	if !p.HandleScroll(mouseX, mouseY, 0, 1) {
		t.Error("expected HandleScroll to return true when inside panel")
	}

	// Scroll down (negative deltaY)
	if !p.HandleScroll(mouseX, mouseY, 0, -1) {
		t.Error("expected HandleScroll to return true for scroll down")
	}

	// Scroll outside panel should not be handled
	outsideY := p.Y - 50
	if p.HandleScroll(mouseX, outsideY, 0, 1) {
		t.Error("expected HandleScroll to return false when outside panel")
	}
}

func testConversationPanelInputIsolation(t *testing.T) {
	p := testConversationPanel()
	p.AddUserMessage("test")
	p.Update()

	// Point inside panel should be contained
	insideX := p.Width / 2
	insideY := p.Y + p.Height()/2

	if !p.ContainsPoint(insideX, insideY) {
		t.Errorf("ContainsPoint(%d, %d) = false, expected true (panel at Y=%d, height=%d)",
			insideX, insideY, p.Y, p.Height())
	}

	// Click inside should be consumed
	if !p.HandleClick(insideX, insideY) {
		t.Error("HandleClick inside panel should return true")
	}

	// Point outside panel should not be contained
	outsideY := p.Y - 20
	if p.ContainsPoint(insideX, outsideY) {
		t.Error("ContainsPoint should return false for point above panel")
	}

	// Click outside should not be consumed
	if p.HandleClick(insideX, outsideY) {
		t.Error("HandleClick outside panel should return false")
	}
}

func testConversationPanelDragHandle(t *testing.T) {
	p := testConversationPanel()
	p.Update()

	// Point on drag handle
	handleX := p.Width / 2
	handleY := p.Y + ui.DragHandleHeight/2

	if !p.IsOnDragHandle(handleX, handleY) {
		t.Error("IsOnDragHandle should return true for point on drag handle")
	}

	// Point below drag handle
	belowHandleY := p.Y + ui.DragHandleHeight + 20
	if p.IsOnDragHandle(handleX, belowHandleY) {
		t.Error("IsOnDragHandle should return false for point below drag handle")
	}
}

func testConversationPanelSubmitFlow(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)
	p := testConversationPanel()

	// Wire up handler to panel
	var submittedText string
	h.OnAction(func(action input.Action) {
		if action == input.ActionSubmitPrompt {
			submittedText = h.TextBuffer()
			// Simulate the game scene behavior
			p.SetText(submittedText)
			p.Submit()
			h.ClearTextBuffer()
		}
	})

	// Focus prompt and enter insert mode
	h.SetFocus(input.FocusPrompt)
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Type a message
	source.QueueCharInput('h', 'e', 'l', 'l', 'o')
	source.AdvanceFrame()
	h.Update()

	// Press Enter to submit
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	h.Update()

	// Verify message was added to panel
	if p.MessageCount() != 1 {
		t.Errorf("expected 1 message after submit, got %d", p.MessageCount())
	}

	// Add response
	p.SetResponseText("Hi there!")

	// Verify conversation has 2 messages
	if p.MessageCount() != 2 {
		t.Errorf("expected 2 messages after response, got %d", p.MessageCount())
	}

	// Panel should be in Processing state after submit
	if p.State != ui.PromptStateProcessing {
		t.Errorf("expected PromptStateProcessing, got %v", p.State)
	}

	// Complete the turn
	p.SetState(ui.PromptStateIdle)

	if p.State != ui.PromptStateIdle {
		t.Errorf("expected PromptStateIdle after SetState, got %v", p.State)
	}
}

func testConversationPanelHeightAnimation(t *testing.T) {
	p := testConversationPanel()

	// Initial height should be minimum
	if p.Height() != ui.MinPanelHeight {
		t.Errorf("initial height = %d, expected %d", p.Height(), ui.MinPanelHeight)
	}

	// Add messages to trigger height growth
	p.AddUserMessage("Message that should increase height")
	p.SetResponseText("Response that adds more height to the panel")

	// Height doesn't change immediately (animation)
	heightBefore := p.Height()

	// Run updates to animate
	for i := 0; i < 50; i++ {
		p.Update()
	}

	heightAfter := p.Height()

	if heightAfter <= heightBefore {
		t.Errorf("height should animate up: before=%d, after=%d", heightBefore, heightAfter)
	}
}
