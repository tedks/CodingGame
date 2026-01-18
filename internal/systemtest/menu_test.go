package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// Menu tests verify j/k navigation, Enter/Space selection, and item skipping.

func testMenuJMovesDown(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)

	// Verify initial selection is 0
	assertMenuSelection(t, m, 0)

	// Press J key
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	m.Update()

	// Selection should move down
	assertMenuSelection(t, m, 1)
}

func testMenuKMovesUp(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)

	// Set initial selection to 1
	m.SelectedIndex = 1

	// Press K key
	source.QueueKeyPress(ebiten.KeyK)
	source.AdvanceFrame()
	m.Update()

	// Selection should move up
	assertMenuSelection(t, m, 0)
}

func testMenuEnterSelects(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)

	// Press Enter
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	selected, cancelled, err := m.Update()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cancelled {
		t.Error("expected selection, not cancellation")
	}
	if selected != "Item 1" {
		t.Errorf("expected 'Item 1', got %q", selected)
	}
}

func testMenuSpaceSelects(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)

	// Move to second item
	m.SelectedIndex = 1

	// Press Space
	source.QueueKeyPress(ebiten.KeySpace)
	source.AdvanceFrame()
	selected, cancelled, err := m.Update()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cancelled {
		t.Error("expected selection, not cancellation")
	}
	if selected != "Item 2" {
		t.Errorf("expected 'Item 2', got %q", selected)
	}
}

func testMenuEscapeCancels(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)
	m.CancelAllowed = true

	// Press Escape
	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	selected, cancelled, err := m.Update()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cancelled {
		t.Error("expected cancellation")
	}
	if selected != "" {
		t.Errorf("expected empty selection on cancel, got %q", selected)
	}
}

func testMenuDisabledItemsSkipped(t *testing.T) {
	source := testutil.NewTestInputSource()

	// Create menu with disabled item in the middle
	items := []*ui.MenuItem{
		ui.NewMenuItem("Item 1"),
		{Label: "Item 2 (Disabled)", Value: "Item 2", Enabled: false},
		ui.NewMenuItem("Item 3"),
	}
	m := testMenu(source, items...)

	// Verify initial selection is 0
	assertMenuSelection(t, m, 0)

	// Press J - should skip disabled item and land on Item 3
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	m.Update()

	// Selection should skip disabled item
	assertMenuSelection(t, m, 2)
}

func testMenuSelectionWraps(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)

	// Move to last item
	m.SelectedIndex = 2

	// Press J - should wrap to first item
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	m.Update()

	assertMenuSelection(t, m, 0)

	// Press K - should wrap to last item
	source.QueueKeyPress(ebiten.KeyK)
	source.AdvanceFrame()
	m.Update()

	assertMenuSelection(t, m, 2)
}

func testMenuArrowKeysNavigate(t *testing.T) {
	source := testutil.NewTestInputSource()
	m := testMenu(source)

	// Verify initial selection is 0
	assertMenuSelection(t, m, 0)

	// Press Down Arrow
	source.QueueKeyPress(ebiten.KeyArrowDown)
	source.AdvanceFrame()
	m.Update()

	assertMenuSelection(t, m, 1)

	// Press Up Arrow
	source.QueueKeyPress(ebiten.KeyArrowUp)
	source.AdvanceFrame()
	m.Update()

	assertMenuSelection(t, m, 0)
}
