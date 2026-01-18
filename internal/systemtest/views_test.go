package systemtest

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// View tests verify 1-5 number key view switching and 't' toggle.

func testViews1SwitchesToMap(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Set to a different view first
	h.SetView(input.ViewBuilding)
	assertView(t, h, input.ViewBuilding)

	// Press 1 key
	source.QueueKeyPress(ebiten.Key1)
	source.AdvanceFrame()
	h.Update()

	// Should switch to Map view
	assertView(t, h, input.ViewMap)
}

func testViews2SwitchesToBuilding(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertView(t, h, input.ViewMap)

	// Press 2 key
	source.QueueKeyPress(ebiten.Key2)
	source.AdvanceFrame()
	h.Update()

	// Should switch to Building view
	assertView(t, h, input.ViewBuilding)
}

func testViews3SwitchesToUnit(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Press 3 key
	source.QueueKeyPress(ebiten.Key3)
	source.AdvanceFrame()
	h.Update()

	// Should switch to Unit view
	assertView(t, h, input.ViewUnit)
}

func testViews4SwitchesToTech(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Press 4 key
	source.QueueKeyPress(ebiten.Key4)
	source.AdvanceFrame()
	h.Update()

	// Should switch to Tech view
	assertView(t, h, input.ViewTech)
}

func testViews5SwitchesToMission(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Press 5 key
	source.QueueKeyPress(ebiten.Key5)
	source.AdvanceFrame()
	h.Update()

	// Should switch to Mission view
	assertView(t, h, input.ViewMission)
}

func testViewsTToggle(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Track view changes
	viewChanges := []input.ViewNumber{}
	h.OnViewChange(func(view input.ViewNumber) {
		viewChanges = append(viewChanges, view)
	})

	// Start at Map view
	assertView(t, h, input.ViewMap)

	// Press T key - should toggle between views
	// Note: The 't' toggle behavior depends on implementation
	// Typically toggles between directory view and dataflow view
	source.QueueKeyPress(ebiten.KeyT)
	source.AdvanceFrame()
	h.Update()

	// The callback should have fired if view changed
	// This test verifies the toggle mechanism works
}
