// Package ui provides user interface components for the CodingGame application.
package ui

import (
	"image/color"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PromptState represents the current state of prompt interaction.
type PromptState int

const (
	// PromptStateIdle - prompt is unfocused, showing placeholder
	PromptStateIdle PromptState = iota
	// PromptStateActive - prompt is focused and receiving input
	PromptStateActive
	// PromptStateProcessing - prompt submitted, waiting for response
	PromptStateProcessing
)

// MessageRole indicates who sent a message.
type MessageRole int

const (
	// RoleUser is a message from the user.
	RoleUser MessageRole = iota
	// RoleAssistant is a message from Claude.
	RoleAssistant
)

// Message represents a single message in the conversation.
type Message struct {
	Role MessageRole
	Text string
	Time time.Time
}

// Panel size constants
const (
	MinPanelHeight    = 60  // Minimized height (just input bar)
	MaxPanelHeight    = 500 // Maximum expanded height
	LineHeight        = 16  // Height per line of text
	InputAreaHeight   = 50  // Height of the input area at bottom
	MessagePadding    = 8   // Padding around messages
	PanelGrowthPerMsg = 40  // How much to grow per message (approximate)
	AnimationSpeed    = 8   // Pixels per frame for smooth animation
	DragHandleHeight  = 8   // Height of the drag handle area at top
)

// PromptPanel provides a conversation interface with Claude.
type PromptPanel struct {
	// Position and size
	X, Y         int
	Width        int
	screenHeight int // Full screen height for calculating position

	// Size state
	minimized       bool
	targetHeight    int
	currentHeight   int
	userHeight      int  // User's preferred height (set by dragging)
	dragging        bool // Whether user is currently dragging to resize
	dragStartY      int  // Y position where drag started
	dragStartHeight int  // Height when drag started

	// Conversation history
	messages []Message

	// Input state
	State       PromptState
	Text        string
	Mode        string // Current mode indicator (NORMAL, INSERT, etc.)
	Placeholder string

	// Scrolling
	scrollOffset int

	// Styling
	BackgroundColor    color.RGBA
	BorderColor        color.RGBA
	FocusedBorderColor color.RGBA
	TextColor          color.RGBA
	UserTextColor      color.RGBA
	AssistantTextColor color.RGBA
	PlaceholderColor   color.RGBA
	ModeColor          color.RGBA
	CursorColor        color.RGBA

	// Cursor animation
	cursorVisible bool
	cursorTimer   time.Time

	// Callbacks
	OnSubmit func(text string) // Called when prompt is submitted
	OnCancel func()            // Called when prompt is cancelled
}

// NewPromptPanel creates a new prompt panel with default styling.
func NewPromptPanel(width int) *PromptPanel {
	return &PromptPanel{
		Width:        width,
		screenHeight: 720, // Default, will be updated

		minimized:     false,
		targetHeight:  MinPanelHeight,
		currentHeight: MinPanelHeight,
		userHeight:    200, // Default user height when expanded

		messages: make([]Message, 0),

		State:       PromptStateIdle,
		Mode:        "NORMAL",
		Placeholder: "Press Enter to chat with Claude...",

		// Semi-transparent dark background
		BackgroundColor:    color.RGBA{15, 15, 25, 200},
		BorderColor:        color.RGBA{60, 60, 80, 255},
		FocusedBorderColor: color.RGBA{100, 150, 255, 255},
		TextColor:          color.RGBA{255, 255, 255, 255},
		UserTextColor:      color.RGBA{150, 200, 255, 255}, // Light blue for user
		AssistantTextColor: color.RGBA{200, 255, 200, 255}, // Light green for Claude
		PlaceholderColor:   color.RGBA{100, 100, 120, 255},
		ModeColor:          color.RGBA{100, 200, 255, 255},
		CursorColor:        color.RGBA{255, 255, 255, 255},

		cursorVisible: true,
		cursorTimer:   time.Now(),
	}
}

// SetPosition sets the prompt panel's position.
// Note: Y position is calculated dynamically based on height.
func (p *PromptPanel) SetPosition(x, y int) {
	p.X = x
	// Y is set dynamically based on screen height and panel height
}

// SetScreenHeight sets the screen height for position calculations.
func (p *PromptPanel) SetScreenHeight(height int) {
	p.screenHeight = height
}

// SetSize sets the prompt panel's width (height is dynamic).
func (p *PromptPanel) SetSize(width, height int) {
	p.Width = width
	// Height is ignored - it's calculated dynamically
}

// SetText sets the current text in the prompt.
func (p *PromptPanel) SetText(text string) {
	p.Text = text
}

// SetMode sets the mode indicator text.
func (p *PromptPanel) SetMode(mode string) {
	p.Mode = mode
	// Update mode color based on mode
	switch mode {
	case "INSERT":
		p.ModeColor = color.RGBA{100, 255, 100, 255} // Green
	case "VISUAL":
		p.ModeColor = color.RGBA{255, 150, 100, 255} // Orange
	default: // NORMAL
		p.ModeColor = color.RGBA{100, 200, 255, 255} // Blue
	}
}

// SetState sets the prompt state.
func (p *PromptPanel) SetState(state PromptState) {
	p.State = state
	if state == PromptStateActive {
		p.cursorTimer = time.Now()
		p.cursorVisible = true
	}
}

// Focus activates the prompt for input.
func (p *PromptPanel) Focus() {
	p.State = PromptStateActive
	p.cursorTimer = time.Now()
	p.cursorVisible = true
}

// Unfocus deactivates the prompt.
func (p *PromptPanel) Unfocus() {
	p.State = PromptStateIdle
}

// IsFocused returns whether the prompt is currently focused.
func (p *PromptPanel) IsFocused() bool {
	return p.State == PromptStateActive
}

// Clear clears the prompt text.
func (p *PromptPanel) Clear() {
	p.Text = ""
}

// SetResponseText adds a response from Claude to the conversation.
func (p *PromptPanel) SetResponseText(text string) {
	if text == "" {
		return
	}
	p.messages = append(p.messages, Message{
		Role: RoleAssistant,
		Text: text,
		Time: time.Now(),
	})
	p.updateTargetHeight()
}

// AddUserMessage adds a user message to the conversation.
func (p *PromptPanel) AddUserMessage(text string) {
	if text == "" {
		return
	}
	p.messages = append(p.messages, Message{
		Role: RoleUser,
		Text: text,
		Time: time.Now(),
	})
	p.updateTargetHeight()
}

// updateTargetHeight calculates the target height based on user preference and messages.
func (p *PromptPanel) updateTargetHeight() {
	if p.minimized {
		p.targetHeight = MinPanelHeight
		return
	}

	// Use user's preferred height, but grow if content exceeds it
	contentHeight := InputAreaHeight
	for _, msg := range p.messages {
		lines := p.countLines(msg.Text)
		contentHeight += lines*LineHeight + MessagePadding*2
	}

	// Target is the larger of user height or content height
	targetHeight := p.userHeight
	if contentHeight > targetHeight {
		targetHeight = contentHeight
	}

	// Clamp to min/max
	if targetHeight < MinPanelHeight {
		targetHeight = MinPanelHeight
	}
	if targetHeight > MaxPanelHeight {
		targetHeight = MaxPanelHeight
	}

	p.targetHeight = targetHeight
}

// countLines estimates how many lines a message will take.
func (p *PromptPanel) countLines(text string) int {
	if text == "" {
		return 1
	}

	// Available width for text (accounting for padding and role indicator)
	availableWidth := p.Width - 100 // rough estimate
	charsPerLine := availableWidth / 7

	if charsPerLine < 20 {
		charsPerLine = 20
	}

	// Count newlines and wrap points
	lines := 1
	currentLineLen := 0

	for _, r := range text {
		if r == '\n' {
			lines++
			currentLineLen = 0
		} else {
			currentLineLen++
			if currentLineLen >= charsPerLine {
				lines++
				currentLineLen = 0
			}
		}
	}

	return lines
}

// ToggleMinimized toggles between minimized and expanded states.
func (p *PromptPanel) ToggleMinimized() {
	p.minimized = !p.minimized
	p.updateTargetHeight()
}

// IsMinimized returns whether the panel is minimized.
func (p *PromptPanel) IsMinimized() bool {
	return p.minimized
}

// Minimize minimizes the panel.
func (p *PromptPanel) Minimize() {
	p.minimized = true
	p.updateTargetHeight()
}

// Maximize expands the panel.
func (p *PromptPanel) Maximize() {
	p.minimized = false
	p.updateTargetHeight()
}

// IsOnDragHandle returns true if the given coordinates are on the drag handle.
func (p *PromptPanel) IsOnDragHandle(x, y int) bool {
	return x >= p.X && x <= p.X+p.Width &&
		y >= p.Y && y <= p.Y+DragHandleHeight
}

// StartDrag begins a drag operation.
func (p *PromptPanel) StartDrag(y int) {
	p.dragging = true
	p.dragStartY = y
	p.dragStartHeight = p.currentHeight
}

// UpdateDrag updates the panel height during a drag.
func (p *PromptPanel) UpdateDrag(y int) {
	if !p.dragging {
		return
	}

	// Calculate new height based on drag delta
	// Moving mouse up (negative delta) increases height
	delta := p.dragStartY - y
	newHeight := p.dragStartHeight + delta

	// Clamp to min/max
	if newHeight < MinPanelHeight {
		newHeight = MinPanelHeight
	}
	if newHeight > MaxPanelHeight {
		newHeight = MaxPanelHeight
	}

	// Update both current and user height
	p.currentHeight = newHeight
	p.userHeight = newHeight
	p.targetHeight = newHeight
	p.Y = p.screenHeight - p.currentHeight

	// If we're at minimum height, consider it minimized
	p.minimized = newHeight <= MinPanelHeight+10
}

// EndDrag ends a drag operation.
func (p *PromptPanel) EndDrag() {
	if !p.dragging {
		return
	}
	p.dragging = false
	p.updateTargetHeight()
}

// IsDragging returns whether the user is currently dragging.
func (p *PromptPanel) IsDragging() bool {
	return p.dragging
}

// ContainsPoint returns true if the given coordinates are within the panel.
func (p *PromptPanel) ContainsPoint(x, y int) bool {
	return x >= p.X && x <= p.X+p.Width &&
		y >= p.Y && y <= p.Y+p.currentHeight
}

// IsOnMinimizeButton returns true if the given coordinates are on the minimize button.
func (p *PromptPanel) IsOnMinimizeButton(x, y int) bool {
	// Minimize button is in top-right corner
	buttonWidth := 100
	buttonHeight := DragHandleHeight + 4
	buttonX := p.X + p.Width - buttonWidth - 10
	buttonY := p.Y
	return x >= buttonX && x <= buttonX+buttonWidth &&
		y >= buttonY && y <= buttonY+buttonHeight
}

// HandleClick processes a click at the given coordinates.
// Returns true if the click was handled.
func (p *PromptPanel) HandleClick(x, y int) bool {
	if !p.ContainsPoint(x, y) {
		return false
	}

	// Check minimize button
	if p.IsOnMinimizeButton(x, y) {
		p.ToggleMinimized()
		return true
	}

	return true // Consumed the click even if no specific action
}

// HandleScroll processes mouse wheel scrolling.
// delta is positive for scroll up, negative for scroll down.
func (p *PromptPanel) HandleScroll(x, y int, deltaX, deltaY float64) bool {
	if !p.ContainsPoint(x, y) {
		return false
	}

	// Scroll the conversation
	if deltaY > 0 {
		p.ScrollUp()
	} else if deltaY < 0 {
		p.ScrollDown()
	}

	return true
}

// ScrollUp scrolls the conversation up.
func (p *PromptPanel) ScrollUp() {
	p.scrollOffset += LineHeight * 2
	maxScroll := p.maxScrollOffset()
	if p.scrollOffset > maxScroll {
		p.scrollOffset = maxScroll
	}
}

// ScrollDown scrolls the conversation down.
func (p *PromptPanel) ScrollDown() {
	p.scrollOffset -= LineHeight * 2
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
}

// maxScrollOffset calculates the maximum scroll offset.
func (p *PromptPanel) maxScrollOffset() int {
	contentHeight := 0
	for _, msg := range p.messages {
		lines := p.countLines(msg.Text)
		contentHeight += lines*LineHeight + MessagePadding*2
	}

	viewableHeight := p.currentHeight - InputAreaHeight
	if contentHeight > viewableHeight {
		return contentHeight - viewableHeight
	}
	return 0
}

// Height returns the current panel height.
func (p *PromptPanel) Height() int {
	return p.currentHeight
}

// Submit triggers the submit callback with the current text.
// Does nothing if text is empty.
func (p *PromptPanel) Submit() {
	if p.Text == "" {
		return // Don't submit empty text
	}
	// Add user message to conversation
	p.AddUserMessage(p.Text)

	if p.OnSubmit != nil {
		p.OnSubmit(p.Text)
	}
	p.State = PromptStateProcessing
}

// Cancel triggers the cancel callback and clears the prompt.
func (p *PromptPanel) Cancel() {
	if p.OnCancel != nil {
		p.OnCancel()
	}
	p.Clear()
	p.State = PromptStateIdle
}

// Update updates the prompt panel state.
func (p *PromptPanel) Update() {
	// Blink cursor every 500ms when active
	if p.State == PromptStateActive {
		if time.Since(p.cursorTimer) > 500*time.Millisecond {
			p.cursorVisible = !p.cursorVisible
			p.cursorTimer = time.Now()
		}
	}

	// Animate height toward target
	if p.currentHeight < p.targetHeight {
		p.currentHeight += AnimationSpeed
		if p.currentHeight > p.targetHeight {
			p.currentHeight = p.targetHeight
		}
	} else if p.currentHeight > p.targetHeight {
		p.currentHeight -= AnimationSpeed
		if p.currentHeight < p.targetHeight {
			p.currentHeight = p.targetHeight
		}
	}

	// Update Y position based on current height
	p.Y = p.screenHeight - p.currentHeight
}

// Draw renders the prompt panel.
func (p *PromptPanel) Draw(screen *ebiten.Image) {
	const padding = 10

	// Draw semi-transparent background
	vector.DrawFilledRect(
		screen,
		float32(p.X),
		float32(p.Y),
		float32(p.Width),
		float32(p.currentHeight),
		p.BackgroundColor,
		false,
	)

	// Draw drag handle at top
	handleColor := color.RGBA{80, 80, 100, 200}
	if p.dragging {
		handleColor = color.RGBA{100, 150, 255, 255}
	}
	vector.DrawFilledRect(
		screen,
		float32(p.X),
		float32(p.Y),
		float32(p.Width),
		float32(DragHandleHeight),
		handleColor,
		false,
	)
	// Draw grip lines in the handle
	gripY := float32(p.Y + DragHandleHeight/2)
	gripColor := color.RGBA{150, 150, 170, 255}
	for i := 0; i < 3; i++ {
		lineY := gripY + float32(i-1)*2
		vector.StrokeLine(
			screen,
			float32(p.Width/2-20),
			lineY,
			float32(p.Width/2+20),
			lineY,
			1,
			gripColor,
			false,
		)
	}

	// Draw border (different color when focused)
	borderColor := p.BorderColor
	if p.State == PromptStateActive {
		borderColor = p.FocusedBorderColor
	}
	vector.StrokeRect(
		screen,
		float32(p.X),
		float32(p.Y),
		float32(p.Width),
		float32(p.currentHeight),
		2,
		borderColor,
		false,
	)

	// If minimized, just draw the input bar
	if p.minimized || len(p.messages) == 0 {
		p.drawInputBar(screen, p.Y+padding)
		return
	}

	// Draw conversation history (above input bar)
	conversationHeight := p.currentHeight - InputAreaHeight
	if conversationHeight > 0 {
		p.drawConversation(screen, p.Y, conversationHeight)
	}

	// Draw input bar at bottom
	inputY := p.Y + p.currentHeight - InputAreaHeight
	p.drawInputBar(screen, inputY)

	// Draw minimize/maximize hint
	hintText := "[-] minimize"
	if p.minimized {
		hintText = "[+] expand"
	}
	hintX := p.X + p.Width - len(hintText)*7 - padding
	ebitenutil.DebugPrintAt(screen, hintText, hintX, p.Y+4)
}

// drawConversation renders the message history.
func (p *PromptPanel) drawConversation(screen *ebiten.Image, startY, height int) {
	const padding = 10

	// Create a clipping region (manual - just don't draw outside)
	currentY := startY + padding - p.scrollOffset

	for _, msg := range p.messages {
		// Skip if above visible area
		msgHeight := p.countLines(msg.Text)*LineHeight + MessagePadding*2
		if currentY+msgHeight < startY {
			currentY += msgHeight
			continue
		}
		// Stop if below visible area
		if currentY > startY+height-InputAreaHeight {
			break
		}

		// Draw message
		p.drawMessage(screen, msg, padding, currentY, startY, startY+height-InputAreaHeight)
		currentY += msgHeight
	}
}

// drawMessage renders a single message.
func (p *PromptPanel) drawMessage(screen *ebiten.Image, msg Message, x, y, clipTop, clipBottom int) {
	// Role indicator and color
	var roleText string
	var textColor color.RGBA
	var bgColor color.RGBA

	if msg.Role == RoleUser {
		roleText = "You: "
		textColor = p.UserTextColor
		bgColor = color.RGBA{30, 40, 60, 150}
	} else {
		roleText = "Claude: "
		textColor = p.AssistantTextColor
		bgColor = color.RGBA{30, 50, 40, 150}
	}

	// Calculate message dimensions
	lines := p.wrapText(msg.Text, p.Width-x*2-80)
	msgHeight := len(lines)*LineHeight + MessagePadding*2

	// Draw message background (only if visible)
	if y >= clipTop && y+msgHeight <= clipBottom {
		vector.DrawFilledRect(
			screen,
			float32(x),
			float32(y),
			float32(p.Width-x*2),
			float32(msgHeight),
			bgColor,
			false,
		)
	}

	// Draw role indicator
	if y+MessagePadding >= clipTop && y+MessagePadding < clipBottom {
		// Use a simple color tint approach - draw role text
		ebitenutil.DebugPrintAt(screen, roleText, x+MessagePadding, y+MessagePadding)
	}

	// Draw message text (wrapped)
	textX := x + MessagePadding + len(roleText)*7
	textY := y + MessagePadding

	for i, line := range lines {
		lineY := textY + i*LineHeight
		if lineY >= clipTop && lineY < clipBottom {
			// Draw with color (using simple debug print for now)
			_ = textColor // TODO: Use colored text rendering
			ebitenutil.DebugPrintAt(screen, line, textX, lineY)
		}
		// After first line, reset textX to align left
		if i == 0 {
			textX = x + MessagePadding
		}
	}
}

// wrapText wraps text to fit within maxWidth pixels.
func (p *PromptPanel) wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 200
	}

	charsPerLine := maxWidth / 7
	if charsPerLine < 20 {
		charsPerLine = 20
	}

	var lines []string
	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		if para == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) <= charsPerLine {
				currentLine += " " + word
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		lines = append(lines, "")
	}

	return lines
}

// drawInputBar renders the input area at the bottom.
func (p *PromptPanel) drawInputBar(screen *ebiten.Image, y int) {
	const (
		padding    = 10
		modeWidth  = 70
		lineHeight = 14
	)

	barHeight := InputAreaHeight - padding

	// Draw input area background
	vector.DrawFilledRect(
		screen,
		float32(p.X+padding),
		float32(y),
		float32(p.Width-padding*2),
		float32(barHeight),
		color.RGBA{20, 20, 35, 220},
		false,
	)

	// Draw mode indicator box on the left
	modeBoxHeight := barHeight - padding
	modeBoxX := p.X + padding*2
	modeBoxY := y + padding/2

	// Mode background
	modeBgColor := color.RGBA{40, 40, 60, 255}
	if p.State == PromptStateActive {
		modeBgColor = color.RGBA{30, 50, 30, 255}
	}
	vector.DrawFilledRect(
		screen,
		float32(modeBoxX),
		float32(modeBoxY),
		float32(modeWidth),
		float32(modeBoxHeight),
		modeBgColor,
		false,
	)

	// Mode text (centered in box)
	modeTextX := modeBoxX + (modeWidth-len(p.Mode)*7)/2
	modeTextY := modeBoxY + (modeBoxHeight-lineHeight)/2
	ebitenutil.DebugPrintAt(screen, p.Mode, modeTextX, modeTextY)

	// Draw text input area
	textX := modeBoxX + modeWidth + padding
	textAreaWidth := p.Width - modeWidth - padding*5

	// Text area background
	vector.DrawFilledRect(
		screen,
		float32(textX),
		float32(modeBoxY),
		float32(textAreaWidth),
		float32(modeBoxHeight),
		color.RGBA{10, 10, 20, 255},
		false,
	)

	// Draw prompt symbol or text
	displayTextY := modeBoxY + (modeBoxHeight-lineHeight)/2

	switch p.State {
	case PromptStateIdle:
		ebitenutil.DebugPrintAt(screen, p.Placeholder, textX+padding, displayTextY)

	case PromptStateActive:
		// Show prompt symbol and text
		promptSymbol := "> "
		displayText := promptSymbol + p.Text
		ebitenutil.DebugPrintAt(screen, displayText, textX+padding, displayTextY)

		// Draw cursor
		if p.cursorVisible {
			cursorX := textX + padding + len(displayText)*7
			cursorY := displayTextY
			vector.DrawFilledRect(
				screen,
				float32(cursorX),
				float32(cursorY),
				2,
				float32(lineHeight),
				p.CursorColor,
				false,
			)
		}

	case PromptStateProcessing:
		ebitenutil.DebugPrintAt(screen, "Thinking...", textX+padding, displayTextY)
	}

	// Draw help text at right edge
	helpText := ""
	switch p.State {
	case PromptStateIdle:
		helpText = "Enter=chat"
	case PromptStateActive:
		helpText = "Enter=send Esc=cancel"
	case PromptStateProcessing:
		helpText = "..."
	}
	helpX := p.X + p.Width - len(helpText)*7 - padding*2
	ebitenutil.DebugPrintAt(screen, helpText, helpX, displayTextY)
}

// ClearHistory clears the conversation history.
func (p *PromptPanel) ClearHistory() {
	p.messages = make([]Message, 0)
	p.scrollOffset = 0
	p.updateTargetHeight()
}

// MessageCount returns the number of messages in history.
func (p *PromptPanel) MessageCount() int {
	return len(p.messages)
}
