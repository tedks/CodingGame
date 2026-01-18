package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Text input tests verify character typing, backspace, and unicode support.

func testTextInputCharacters(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Type "hello"
	source.QueueCharInput('h', 'e', 'l', 'l', 'o')
	source.AdvanceFrame()
	h.Update()

	// Text buffer should contain "hello"
	assertTextBuffer(t, h, "hello")
}

func testTextInputBackspace(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode with some text
	h.SetMode(input.ModeInsert)
	h.SetTextBuffer("hello")

	// Press Backspace
	source.QueueKeyPress(ebiten.KeyBackspace)
	source.AdvanceFrame()
	h.Update()

	// Text buffer should be "hell"
	assertTextBuffer(t, h, "hell")
}

func testTextInputUnicode(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Type unicode characters
	source.QueueCharInput('H', 'e', 'l', 'l', 'o', ' ')
	source.QueueCharInput('\u4e16', '\u754c') // "World" in Chinese
	source.AdvanceFrame()
	h.Update()

	// Text buffer should contain unicode
	expected := "Hello \u4e16\u754c"
	assertTextBuffer(t, h, expected)
}

func testTextInputEmptyBackspace(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode with empty buffer
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Press Backspace on empty buffer - should be no-op
	source.QueueKeyPress(ebiten.KeyBackspace)
	source.AdvanceFrame()
	h.Update()

	// Text buffer should still be empty
	assertTextBuffer(t, h, "")
}

func testTextInputRapidTyping(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Track text changes
	textChanges := []string{}
	h.OnTextChange(func(text string) {
		textChanges = append(textChanges, text)
	})

	// Type multiple characters rapidly
	for _, c := range "rapid" {
		source.QueueCharInput(c)
		source.AdvanceFrame()
		h.Update()
	}

	// Final text should be complete
	assertTextBuffer(t, h, "rapid")

	// Should have had 5 text change callbacks
	if len(textChanges) != 5 {
		t.Errorf("expected 5 text changes, got %d", len(textChanges))
	}
}

func testTextInputSpecialChars(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Enter Insert mode
	h.SetMode(input.ModeInsert)
	h.ClearTextBuffer()

	// Type special characters
	source.QueueCharInput('!', '@', '#', '$', '%')
	source.AdvanceFrame()
	h.Update()

	assertTextBuffer(t, h, "!@#$%")

	// Type more special characters
	source.QueueCharInput('(', ')', '[', ']', '{', '}')
	source.AdvanceFrame()
	h.Update()

	assertTextBuffer(t, h, "!@#$%()[]{}")
}
