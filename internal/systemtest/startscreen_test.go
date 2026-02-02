package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// StartScreen tests verify navigation and state transitions in the start screen.

func testStartScreenNavigatesWithJ(t *testing.T) {
	source := testutil.NewTestInputSource()
	ss := testStartScreen(source)

	// Initial state should be main menu with first item selected
	initialSelection := ss.SelectedIndex()

	// Press J to move down
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	ss.Update()

	// Selection should have moved down
	if ss.SelectedIndex() != initialSelection+1 {
		t.Errorf("expected selection to move from %d to %d, got %d",
			initialSelection, initialSelection+1, ss.SelectedIndex())
	}
}

func testStartScreenNavigatesWithK(t *testing.T) {
	source := testutil.NewTestInputSource()
	ss := testStartScreen(source)

	// Move down first so we can move up
	source.QueueKeyPress(ebiten.KeyJ)
	source.AdvanceFrame()
	ss.Update()

	currentSelection := ss.SelectedIndex()

	// Press K to move up
	source.QueueKeyPress(ebiten.KeyK)
	source.AdvanceFrame()
	ss.Update()

	// Selection should have moved up
	if ss.SelectedIndex() != currentSelection-1 {
		t.Errorf("expected selection to move from %d to %d, got %d",
			currentSelection, currentSelection-1, ss.SelectedIndex())
	}
}

func testStartScreenEnterAdvancesState(t *testing.T) {
	source := testutil.NewTestInputSource()
	ss := testStartScreen(source)

	// Initial state should be MainMenu
	if ss.State() != ui.StateMainMenu {
		t.Fatalf("expected initial state StateMainMenu, got %v", ss.State())
	}

	// Press Enter to select "NEW GAME"
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	ss.Update()

	// State should advance to HarnessSelect
	if ss.State() != ui.StateHarnessSelect {
		t.Errorf("expected state StateHarnessSelect after Enter, got %v", ss.State())
	}
}

func testStartScreenEscapeGoesBack(t *testing.T) {
	source := testutil.NewTestInputSource()
	ss := testStartScreen(source)

	// Advance to HarnessSelect first
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	ss.Update()

	if ss.State() != ui.StateHarnessSelect {
		t.Fatalf("expected state StateHarnessSelect, got %v", ss.State())
	}

	// Press Escape to go back
	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	ss.Update()

	// State should go back to MainMenu
	if ss.State() != ui.StateMainMenu {
		t.Errorf("expected state StateMainMenu after Escape, got %v", ss.State())
	}
}

func testStartScreenArrowKeysNavigate(t *testing.T) {
	source := testutil.NewTestInputSource()
	ss := testStartScreen(source)

	initialSelection := ss.SelectedIndex()

	// Press Down arrow to move down
	source.QueueKeyPress(ebiten.KeyArrowDown)
	source.AdvanceFrame()
	ss.Update()

	if ss.SelectedIndex() != initialSelection+1 {
		t.Errorf("expected Down arrow to move selection down, got %d", ss.SelectedIndex())
	}

	currentSelection := ss.SelectedIndex()

	// Press Up arrow to move up
	source.QueueKeyPress(ebiten.KeyArrowUp)
	source.AdvanceFrame()
	ss.Update()

	if ss.SelectedIndex() != currentSelection-1 {
		t.Errorf("expected Up arrow to move selection up, got %d", ss.SelectedIndex())
	}
}
