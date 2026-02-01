package theme

// RightAlignedX returns the x coordinate for an element aligned to the right edge.
// It returns ok=false when the element would overlap the minimum padding.
func RightAlignedX(x, width, elementWidth, minPadding int) (alignedX int, ok bool) {
	alignedX = x + width - elementWidth
	minX := x + minPadding
	if alignedX < minX {
		return minX, false
	}
	return alignedX, true
}

// CenterTextX approximates centering text by character width.
func CenterTextX(x, width int, text string, charWidth int) int {
	return x + (width-len(text)*charWidth)/2
}

// CenterTextY centers a single line of text with a known line height.
func CenterTextY(y, height, lineHeight int) int {
	return y + (height-lineHeight)/2
}
