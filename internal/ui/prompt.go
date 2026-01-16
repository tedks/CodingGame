// Package ui provides user interface components for the CodingGame application.
package ui

import (
	"image/color"
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

// PromptPanel provides a text input area for user commands.
type PromptPanel struct {
	// Position and size
	X, Y          int
	Width, Height int

	// State
	State      PromptState
	Text       string
	Mode       string // Current mode indicator (NORMAL, INSERT, etc.)
	Placeholder string

	// Styling
	BackgroundColor   color.RGBA
	BorderColor       color.RGBA
	FocusedBorderColor color.RGBA
	TextColor         color.RGBA
	PlaceholderColor  color.RGBA
	ModeColor         color.RGBA
	CursorColor       color.RGBA

	// Cursor animation
	cursorVisible bool
	cursorTimer   time.Time

	// Callbacks
	OnSubmit func(text string) // Called when prompt is submitted
	OnCancel func()           // Called when prompt is cancelled
}

// NewPromptPanel creates a new prompt panel with default styling.
func NewPromptPanel(width int) *PromptPanel {
	return &PromptPanel{
		Width:  width,
		Height: 60,

		State:       PromptStateIdle,
		Mode:        "NORMAL",
		Placeholder: "Press Enter or ':' to enter command...",

		BackgroundColor:    color.RGBA{20, 20, 30, 250},
		BorderColor:        color.RGBA{60, 60, 80, 255},
		FocusedBorderColor: color.RGBA{100, 150, 255, 255},
		TextColor:          color.RGBA{255, 255, 255, 255},
		PlaceholderColor:   color.RGBA{100, 100, 120, 255},
		ModeColor:          color.RGBA{100, 200, 255, 255},
		CursorColor:        color.RGBA{255, 255, 255, 255},

		cursorVisible: true,
		cursorTimer:   time.Now(),
	}
}

// SetPosition sets the prompt panel's position.
func (p *PromptPanel) SetPosition(x, y int) {
	p.X = x
	p.Y = y
}

// SetSize sets the prompt panel's dimensions.
func (p *PromptPanel) SetSize(width, height int) {
	p.Width = width
	p.Height = height
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

// Submit triggers the submit callback with the current text.
// Does nothing if text is empty.
func (p *PromptPanel) Submit() {
	if p.Text == "" {
		return // Don't submit empty text
	}
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
}

// Draw renders the prompt panel.
func (p *PromptPanel) Draw(screen *ebiten.Image) {
	const (
		padding    = 10
		modeWidth  = 80
		lineHeight = 14
	)

	// Draw background
	vector.DrawFilledRect(
		screen,
		float32(p.X),
		float32(p.Y),
		float32(p.Width),
		float32(p.Height),
		p.BackgroundColor,
		false,
	)

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
		float32(p.Height),
		2,
		borderColor,
		false,
	)

	// Draw mode indicator box on the left
	modeBoxWidth := modeWidth
	modeBoxHeight := p.Height - 2*padding
	modeBoxX := p.X + padding
	modeBoxY := p.Y + padding

	// Mode background
	modeBgColor := color.RGBA{40, 40, 60, 255}
	if p.State == PromptStateActive {
		modeBgColor = color.RGBA{30, 50, 30, 255}
	}
	vector.DrawFilledRect(
		screen,
		float32(modeBoxX),
		float32(modeBoxY),
		float32(modeBoxWidth),
		float32(modeBoxHeight),
		modeBgColor,
		false,
	)

	// Mode text (centered in box)
	modeTextX := modeBoxX + (modeBoxWidth-len(p.Mode)*7)/2
	modeTextY := modeBoxY + (modeBoxHeight-lineHeight)/2
	ebitenutil.DebugPrintAt(screen, p.Mode, modeTextX, modeTextY)

	// Draw text input area
	textX := modeBoxX + modeBoxWidth + padding
	textY := p.Y + padding
	textAreaWidth := p.Width - modeBoxWidth - 3*padding

	// Text area background
	vector.DrawFilledRect(
		screen,
		float32(textX),
		float32(textY),
		float32(textAreaWidth),
		float32(modeBoxHeight),
		color.RGBA{15, 15, 25, 255},
		false,
	)

	// Draw prompt symbol or text
	displayTextY := textY + (modeBoxHeight-lineHeight)/2

	switch p.State {
	case PromptStateIdle:
		// Show placeholder
		ebitenutil.DebugPrintAt(screen, p.Placeholder, textX+padding, displayTextY)

	case PromptStateActive:
		// Show prompt symbol and text
		promptSymbol := "> "
		displayText := promptSymbol + p.Text
		ebitenutil.DebugPrintAt(screen, displayText, textX+padding, displayTextY)

		// Draw cursor
		if p.cursorVisible {
			cursorX := textX + padding + (len(displayText))*7
			cursorY := displayTextY
			// Draw cursor as a vertical line
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
		// Show processing indicator
		ebitenutil.DebugPrintAt(screen, "Processing...", textX+padding, displayTextY)
	}

	// Draw help text at right edge
	helpText := ""
	switch p.State {
	case PromptStateIdle:
		helpText = "Enter=focus"
	case PromptStateActive:
		helpText = "Enter=submit  Esc=cancel"
	case PromptStateProcessing:
		helpText = "Working..."
	}
	helpX := p.X + p.Width - len(helpText)*7 - padding
	ebitenutil.DebugPrintAt(screen, helpText, helpX, displayTextY)
}
