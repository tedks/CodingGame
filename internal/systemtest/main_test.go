// Package systemtest provides exhaustive system tests that drive virtual keyboard
// and mouse input to verify all interactions work end-to-end.
//
// IMPORTANT: Due to GLFW initialization constraints, all system tests must run as
// subtests of a single TestSystemTests entry point. GLFW can only be initialized
// once per process, and Ebitengine's RunGame() initializes GLFW.
//
// To add new tests:
// 1. Create a function: func testMyFeature(t *testing.T, handler *input.Handler) { ... }
// 2. Register it in the TestSystemTests function below
// 3. Use the testutil.Scenario DSL for declarative test definitions
package systemtest

import (
	"os"
	"testing"

	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// hasDisplay checks if a display is available for running graphical tests.
func hasDisplay() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

// skipIfNoDisplay skips the test if no display is available.
func skipIfNoDisplay(t *testing.T) {
	if !hasDisplay() {
		t.Skip("Skipping: no display available (set DISPLAY or WAYLAND_DISPLAY)")
	}
}

// TestSystemTests is the single entry point for all system tests.
// All tests run as subtests to avoid GLFW re-initialization issues.
func TestSystemTests(t *testing.T) {
	skipIfNoDisplay(t)

	// Navigation tests
	t.Run("Navigation", func(t *testing.T) {
		t.Run("HKeyPansLeft", testNavigationHKeyPansLeft)
		t.Run("JKeyPansDown", testNavigationJKeyPansDown)
		t.Run("KKeyPansUp", testNavigationKKeyPansUp)
		t.Run("LKeyPansRight", testNavigationLKeyPansRight)
		t.Run("ArrowKeysPan", testNavigationArrowKeys)
		t.Run("PlusKeyZoomsIn", testNavigationPlusKeyZoomsIn)
		t.Run("MinusKeyZoomsOut", testNavigationMinusKeyZoomsOut)
		t.Run("HeldKeysContinuePanning", testNavigationHeldKeys)
	})

	// Mode tests
	t.Run("Modes", func(t *testing.T) {
		t.Run("IEntersInsertMode", testModesIEntersInsertMode)
		t.Run("VEntersVisualMode", testModesVEntersVisualMode)
		t.Run("EscapeReturnsToNormal", testModesEscapeReturnsToNormal)
		t.Run("ModeTransitionsCorrectly", testModeTransitions)
		t.Run("ColonNotEnterCommandMode", testModesColonBehavior)
		t.Run("ModeIndicatorUpdates", testModeIndicatorUpdates)
	})

	// Focus tests
	t.Run("Focus", func(t *testing.T) {
		t.Run("TabCyclesFocus", testFocusTabCycles)
		t.Run("ShiftTabCyclesReverse", testFocusShiftTabReverse)
		t.Run("SlashFocusesPrompt", testFocusSlashPrompt)
		t.Run("FocusChangesAffectInput", testFocusInputRouting)
		t.Run("FocusIndicatorVisible", testFocusIndicatorVisible)
		t.Run("FocusWrapAround", testFocusWrapAround)
	})

	// View tests
	t.Run("Views", func(t *testing.T) {
		t.Run("Key1SwitchesToMap", testViews1SwitchesToMap)
		t.Run("Key2SwitchesToBuilding", testViews2SwitchesToBuilding)
		t.Run("Key3SwitchesToUnit", testViews3SwitchesToUnit)
		t.Run("Key4SwitchesToTech", testViews4SwitchesToTech)
		t.Run("Key5SwitchesToMission", testViews5SwitchesToMission)
		t.Run("TTogglesBetweenViews", testViewsTToggle)
	})

	// Text input tests
	t.Run("TextInput", func(t *testing.T) {
		t.Run("CharactersAppendToBuffer", testTextInputCharacters)
		t.Run("BackspaceDeletesCharacter", testTextInputBackspace)
		t.Run("UnicodeCharactersWork", testTextInputUnicode)
		t.Run("EmptyBackspaceNoOp", testTextInputEmptyBackspace)
		t.Run("RapidTyping", testTextInputRapidTyping)
		t.Run("SpecialCharacters", testTextInputSpecialChars)
	})

	// Menu tests
	t.Run("Menu", func(t *testing.T) {
		t.Run("JMovesDown", testMenuJMovesDown)
		t.Run("KMovesUp", testMenuKMovesUp)
		t.Run("EnterSelects", testMenuEnterSelects)
		t.Run("SpaceSelects", testMenuSpaceSelects)
		t.Run("EscapeCancels", testMenuEscapeCancels)
		t.Run("DisabledItemsSkipped", testMenuDisabledItemsSkipped)
		t.Run("SelectionWraps", testMenuSelectionWraps)
		t.Run("ArrowKeysNavigate", testMenuArrowKeysNavigate)
	})

	// Prompt tests
	t.Run("Prompt", func(t *testing.T) {
		t.Run("SlashFocusesAndInsert", testPromptSlashFocuses)
		t.Run("EnterSubmitsPrompt", testPromptEnterSubmits)
		t.Run("EscapeCancelsPrompt", testPromptEscapeCancels)
		t.Run("CursorBlinks", testPromptCursorBlinks)
		t.Run("TextDisplaysCorrectly", testPromptTextDisplay)
		t.Run("ClearAfterSubmit", testPromptClearAfterSubmit)
	})

	// MapView mouse interaction tests
	t.Run("MapView", func(t *testing.T) {
		t.Run("SingleClickSelectsTile", testMapViewSingleClickSelectsTile)
		t.Run("DoubleClickTriggersCallback", testMapViewDoubleClickTriggersCallback)
		t.Run("DoubleClickUpdatesSelection", testMapViewDoubleClickUpdatesSelection)
		t.Run("DragVsClickThreshold", testMapViewDragVsClickThreshold)
		t.Run("ClickInBorderGap", testMapViewClickInBorderGap)
		t.Run("TripleClickBehavior", testMapViewTripleClickBehavior)
	})
}

// Test helper types and functions

// testHandler creates a new input handler for testing with the given input source.
func testHandler(source *testutil.TestInputSource) *input.Handler {
	h := input.NewHandler()
	h.SetInputSource(source)
	return h
}

// testMenu creates a new menu for testing with the given input source.
func testMenu(source *testutil.TestInputSource, items ...*ui.MenuItem) *ui.Menu {
	if len(items) == 0 {
		items = []*ui.MenuItem{
			ui.NewMenuItem("Item 1"),
			ui.NewMenuItem("Item 2"),
			ui.NewMenuItem("Item 3"),
		}
	}
	m := ui.NewMenu("Test Menu", items)
	m.SetInputSource(source)
	return m
}

// testStartScreen creates a new start screen for testing with the given input source.
func testStartScreen(source *testutil.TestInputSource) *ui.StartScreen {
	ss := ui.NewStartScreen(800, 600, nil)
	ss.SetInputSource(source)
	return ss
}

// assertMode checks that the handler is in the expected mode.
func assertMode(t *testing.T, h *input.Handler, expected input.Mode) {
	t.Helper()
	if h.Mode() != expected {
		t.Errorf("expected mode %v, got %v", expected, h.Mode())
	}
}

// assertFocus checks that the handler has the expected focus.
func assertFocus(t *testing.T, h *input.Handler, expected input.FocusArea) {
	t.Helper()
	if h.Focus() != expected {
		t.Errorf("expected focus %v, got %v", expected, h.Focus())
	}
}

// assertView checks that the handler has the expected view.
func assertView(t *testing.T, h *input.Handler, expected input.ViewNumber) {
	t.Helper()
	if h.View() != expected {
		t.Errorf("expected view %v, got %v", expected, h.View())
	}
}

// assertTextBuffer checks that the handler's text buffer matches expected.
func assertTextBuffer(t *testing.T, h *input.Handler, expected string) {
	t.Helper()
	if h.TextBuffer() != expected {
		t.Errorf("expected text buffer %q, got %q", expected, h.TextBuffer())
	}
}

// assertMenuSelection checks that the menu has the expected selection index.
func assertMenuSelection(t *testing.T, m *ui.Menu, expected int) {
	t.Helper()
	if m.SelectedIndex != expected {
		t.Errorf("expected selection index %d, got %d", expected, m.SelectedIndex)
	}
}
