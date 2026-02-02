package game

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// gameState captures the relevant state of a GameScene for comparison.
type gameState struct {
	panX      float64
	panY      float64
	zoomLevel int
	mode      input.Mode
	focus     input.FocusArea
	view      input.ViewNumber
}

func captureState(gs *GameScene) gameState {
	return gameState{
		panX:      gs.mapView.PanX(),
		panY:      gs.mapView.PanY(),
		zoomLevel: gs.mapView.ZoomLevel(),
		mode:      gs.inputHandler.Mode(),
		focus:     gs.inputHandler.Focus(),
		view:      gs.currentView,
	}
}

// TestGameScene_Determinism verifies that the same sequence of inputs
// produces the same final state every time.
//
// This test runs the same input sequence multiple times and verifies
// that the game state converges to the same values each run.
func TestGameScene_Determinism(t *testing.T) {
	// Create a temp directory once - will be used for all runs
	tmpDir, err := os.MkdirTemp("", "determinism-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Define a deterministic sequence of inputs
	inputSequence := []testutil.InputEvent{
		// Initial navigation
		{Type: testutil.KeyHold, Key: ebiten.KeyH, Duration: 5}, // Pan left
		{Type: testutil.KeyHold, Key: ebiten.KeyJ, Duration: 5}, // Pan down
		{Type: testutil.KeyPress, Key: ebiten.Key2},             // Switch to building view
		{Type: testutil.KeyPress, Key: ebiten.Key1},             // Back to map view
		{Type: testutil.KeyHold, Key: ebiten.KeyL, Duration: 3}, // Pan right
		{Type: testutil.KeyHold, Key: ebiten.KeyK, Duration: 3}, // Pan up
		{Type: testutil.KeyPress, Key: ebiten.KeyEnter},         // Enter insert mode
		{Type: testutil.KeyPress, Key: ebiten.KeyEscape},        // Exit insert mode
		{Type: testutil.KeyHold, Key: ebiten.KeyH, Duration: 2}, // Pan left again
		{Type: testutil.KeyPress, Key: ebiten.Key4},             // Tech view
		{Type: testutil.KeyPress, Key: ebiten.Key1},             // Map view
	}

	const numRuns = 3
	var states []gameState

	for run := 0; run < numRuns; run++ {
		gs, err := NewGameScene(tmpDir, 800, 600)
		if err != nil {
			t.Fatalf("Run %d: Failed to create game scene: %v", run, err)
		}

		source := testutil.NewTestInputSource()
		gs.SetInputSource(source)

		// Queue all events
		source.QueueEvents(inputSequence...)

		// Process all events
		for source.HasPendingEvents() {
			source.AdvanceFrame()
			gs.Update()
		}

		// Capture final state
		states = append(states, captureState(gs))

		gs.Close()
	}

	// Compare all states - they should be identical
	for i := 1; i < len(states); i++ {
		if states[i].panX != states[0].panX {
			t.Errorf("Run %d panX differs: %f vs %f", i, states[i].panX, states[0].panX)
		}
		if states[i].panY != states[0].panY {
			t.Errorf("Run %d panY differs: %f vs %f", i, states[i].panY, states[0].panY)
		}
		if states[i].zoomLevel != states[0].zoomLevel {
			t.Errorf("Run %d zoomLevel differs: %d vs %d", i, states[i].zoomLevel, states[0].zoomLevel)
		}
		if states[i].mode != states[0].mode {
			t.Errorf("Run %d mode differs: %v vs %v", i, states[i].mode, states[0].mode)
		}
		if states[i].focus != states[0].focus {
			t.Errorf("Run %d focus differs: %v vs %v", i, states[i].focus, states[0].focus)
		}
		if states[i].view != states[0].view {
			t.Errorf("Run %d view differs: %v vs %v", i, states[i].view, states[0].view)
		}
	}
}

// TestGameScene_DeterminismWithTextInput verifies that text input
// produces consistent state.
func TestGameScene_DeterminismWithTextInput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "determinism-text-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Sequence with text input
	inputSequence := []testutil.InputEvent{
		{Type: testutil.KeyPress, Key: ebiten.KeyEnter}, // Enter insert mode
		{Type: testutil.CharInput, Char: 'h'},
		{Type: testutil.CharInput, Char: 'e'},
		{Type: testutil.CharInput, Char: 'l'},
		{Type: testutil.CharInput, Char: 'l'},
		{Type: testutil.CharInput, Char: 'o'},
		{Type: testutil.KeyPress, Key: ebiten.KeyEscape}, // Exit insert mode
	}

	const numRuns = 3
	var textBuffers []string

	for run := 0; run < numRuns; run++ {
		gs, err := NewGameScene(tmpDir, 800, 600)
		if err != nil {
			t.Fatalf("Run %d: Failed to create game scene: %v", run, err)
		}

		source := testutil.NewTestInputSource()
		gs.SetInputSource(source)

		source.QueueEvents(inputSequence...)

		for source.HasPendingEvents() {
			source.AdvanceFrame()
			gs.Update()
		}

		textBuffers = append(textBuffers, gs.inputHandler.TextBuffer())
		gs.Close()
	}

	// All text buffers should be identical
	for i := 1; i < len(textBuffers); i++ {
		if textBuffers[i] != textBuffers[0] {
			t.Errorf("Run %d textBuffer differs: %q vs %q", i, textBuffers[i], textBuffers[0])
		}
	}
}

// TestGameScene_UpdateIdempotent verifies that calling Update() multiple
// times with no new input doesn't change state.
func TestGameScene_UpdateIdempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "idempotent-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	source := testutil.NewTestInputSource()
	gs.SetInputSource(source)

	// Do some initial operations
	source.QueueKeyHold(ebiten.KeyH, 3)
	for source.HasPendingEvents() {
		source.AdvanceFrame()
		gs.Update()
	}

	// Capture state
	state1 := captureState(gs)

	// Call Update multiple times with no new input
	for i := 0; i < 10; i++ {
		source.AdvanceFrame()
		gs.Update()
	}

	// State should be unchanged (except possibly time-based state)
	state2 := captureState(gs)

	if state1.panX != state2.panX {
		t.Errorf("panX changed without input: %f -> %f", state1.panX, state2.panX)
	}
	if state1.panY != state2.panY {
		t.Errorf("panY changed without input: %f -> %f", state1.panY, state2.panY)
	}
	if state1.mode != state2.mode {
		t.Errorf("mode changed without input: %v -> %v", state1.mode, state2.mode)
	}
	if state1.focus != state2.focus {
		t.Errorf("focus changed without input: %v -> %v", state1.focus, state2.focus)
	}
	if state1.view != state2.view {
		t.Errorf("view changed without input: %v -> %v", state1.view, state2.view)
	}
}
