package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// HeaderLayout defines shared spacing and background styling for renderer headers.
type HeaderLayout struct {
	Height         int
	Padding        int
	TitleOffsetY   int
	SummaryOffsetY int
	Background     color.RGBA
}

// DefaultHeaderLayout returns the shared header layout used by renderer summaries.
func DefaultHeaderLayout() HeaderLayout {
	return HeaderLayout{
		Height:         60,
		Padding:        20,
		TitleOffsetY:   10,
		SummaryOffsetY: 30,
		Background:     color.RGBA{R: 0x1A, G: 0x1A, B: 0x2E, A: 0xFF},
	}
}

// DrawHeader draws a standard renderer header background and title.
func DrawHeader(screen *ebiten.Image, title string, x, y, width int, layout HeaderLayout) {
	DrawHeaderBackground(screen, x, y, width, layout)
	DrawHeaderTitle(screen, title, x, y, layout)
}

// DrawHeaderBackground draws the background bar for a header.
func DrawHeaderBackground(screen *ebiten.Image, x, y, width int, layout HeaderLayout) {
	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(layout.Height),
		layout.Background,
		false,
	)
}

// DrawHeaderTitle draws the title text within a header.
func DrawHeaderTitle(screen *ebiten.Image, title string, x, y int, layout HeaderLayout) {
	ebitenutil.DebugPrintAt(screen, title, x+layout.Padding, y+layout.TitleOffsetY)
}

// DrawHeaderSummary draws a left-aligned summary line in a header.
func DrawHeaderSummary(screen *ebiten.Image, summary string, x, y int, layout HeaderLayout) {
	ebitenutil.DebugPrintAt(screen, summary, x+layout.Padding, y+layout.SummaryOffsetY)
}

// RightAlignedSummaryX returns the starting X for a right-aligned header summary block.
func RightAlignedSummaryX(x, width, blockWidth, minPadding int) (int, bool) {
	startX := x + width - blockWidth
	minX := x + minPadding
	if startX < minX {
		return minX, false
	}
	return startX, true
}

// CardStyle defines shared styling for card-like panels.
type CardStyle struct {
	Background  color.RGBA
	BorderColor color.RGBA
	BorderWidth float32
	AccentWidth int
	AccentColor color.RGBA
}

// DefaultCardStyle returns a shared card style with standard background and border width.
func DefaultCardStyle(borderColor color.RGBA) CardStyle {
	return CardStyle{
		Background:  color.RGBA{R: 0x2A, G: 0x2A, B: 0x3E, A: 0xFF},
		BorderColor: borderColor,
		BorderWidth: 2,
	}
}

// DrawCard draws a card-like panel with optional border and accent.
func DrawCard(screen *ebiten.Image, x, y, width, height int, style CardStyle) {
	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(height),
		style.Background,
		false,
	)

	if style.AccentWidth > 0 {
		vector.DrawFilledRect(
			screen,
			float32(x), float32(y),
			float32(style.AccentWidth), float32(height),
			style.AccentColor,
			false,
		)
	}

	if style.BorderWidth > 0 {
		vector.StrokeRect(
			screen,
			float32(x), float32(y),
			float32(width), float32(height),
			style.BorderWidth,
			style.BorderColor,
			false,
		)
	}
}

// DrawEmptyState centers a message and hint for empty renderer content.
func DrawEmptyState(screen *ebiten.Image, message, hint string, x, y, width, height int) {
	const approxCharWidth = 3
	msgX := x + width/2 - len(message)*approxCharWidth
	msgY := y + height/2 - 20
	hintX := x + width/2 - len(hint)*approxCharWidth
	hintY := y + height/2

	ebitenutil.DebugPrintAt(screen, message, msgX, msgY)
	ebitenutil.DebugPrintAt(screen, hint, hintX, hintY)
}

// DrawProgressBar draws a simple horizontal progress bar.
func DrawProgressBar(screen *ebiten.Image, x, y, width, height int, progress float64, background, fill color.RGBA) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(height),
		background,
		false,
	)

	fillWidth := int(float64(width) * progress)
	if fillWidth > 0 {
		vector.DrawFilledRect(
			screen,
			float32(x), float32(y),
			float32(fillWidth), float32(height),
			fill,
			false,
		)
	}
}
