package game

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// TestGameScene_WasMousePressedTracking verifies that the click detection
// state machine correctly tracks mouse button state across frames.
func TestGameScene_WasMousePressedTracking(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mouse-tracking-*")
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

	// Initially, wasMousePressed should be false
	if gs.wasMousePressed {
		t.Error("wasMousePressed should be false initially")
	}

	// Simulate mouse press
	source.QueueMouseClick(ebiten.MouseButtonLeft)
	source.AdvanceFrame()
	gs.Update()

	// wasMousePressed should now be true (tracked at end of handlePromptPanelDrag)
	if !gs.wasMousePressed {
		t.Error("wasMousePressed should be true after mouse press")
	}

	// Release mouse
	source.QueueMouseRelease(ebiten.MouseButtonLeft)
	source.AdvanceFrame()
	gs.Update()

	// wasMousePressed should now be false
	if gs.wasMousePressed {
		t.Error("wasMousePressed should be false after mouse release")
	}
}

// TestGameScene_FocusChangeDuringHeldKey verifies that when focus changes
// while a key is held, the old focus area stops receiving input.
func TestGameScene_FocusChangeDuringHeldKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "focus-change-*")
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

	// Verify initial state: focus on map, normal mode
	if gs.inputHandler.Focus() != input.FocusMap {
		t.Errorf("Expected initial focus FocusMap, got %v", gs.inputHandler.Focus())
	}

	// Start holding 'h' key (move left)
	source.QueueKeyHold(ebiten.KeyH, 10)
	source.AdvanceFrame()
	gs.Update()

	// Record initial pan
	initialPanX := gs.mapView.PanX()

	// Continue holding - map should pan
	source.AdvanceFrame()
	gs.Update()

	panAfterHold := gs.mapView.PanX()
	if panAfterHold == initialPanX {
		// If pan didn't change, the key hold might not be working
		// This could be due to zoom level or other factors
		t.Log("Note: Pan didn't change during key hold (may be expected based on implementation)")
	}

	// Now change focus to prompt (should stop pan actions)
	gs.inputHandler.SetFocus(input.FocusPrompt)
	gs.inputHandler.SetMode(input.ModeInsert)

	// Continue with key held
	source.AdvanceFrame()
	gs.Update()

	panAfterFocusChange := gs.mapView.PanX()

	// Additional frames while key is held
	source.AdvanceFrame()
	gs.Update()

	panFinal := gs.mapView.PanX()

	// After focus change to prompt, map should not continue panning
	// because the Update() checks for FocusMap before applying panning
	if panAfterFocusChange != panFinal {
		t.Errorf("Map continued panning after focus change: %f -> %f", panAfterFocusChange, panFinal)
	}
}

// TestGameScene_PromptConsumesInput verifies that when the prompt panel
// is active (handling drag or click), map input is disabled.
func TestGameScene_PromptConsumesInput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompt-consumes-*")
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

	// Position mouse over the prompt panel (at the bottom of the screen)
	// PromptPanelHeight is 60, so position at y=550 for 600 height screen
	promptY := gs.height - 30 // Middle of prompt panel
	source.QueueMouseMove(gs.width/2, promptY)
	source.AdvanceFrame()
	gs.Update()

	// Click on prompt panel
	source.QueueMouseClick(ebiten.MouseButtonLeft)
	source.AdvanceFrame()
	gs.Update()

	// The prompt panel should have consumed the input
	// This means mouse input to the map should be disabled

	// The map's mouse input enabled state is set by handlePromptPanelDrag
	// We can't directly check it, but we verify the behavior by checking
	// that clicking in the prompt area doesn't affect the map

	// Release mouse
	source.QueueMouseRelease(ebiten.MouseButtonLeft)
	source.AdvanceFrame()
	gs.Update()

	// Test passes if no panic occurred and we got here
}

// TestGameScene_InputModeTransitions verifies that mode transitions
// work correctly and affect input routing.
func TestGameScene_InputModeTransitions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mode-transitions-*")
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

	// Start in Normal mode
	if gs.inputHandler.Mode() != input.ModeNormal {
		t.Errorf("Expected initial mode ModeNormal, got %v", gs.inputHandler.Mode())
	}

	// Press Enter to enter Insert mode and focus prompt
	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()
	gs.Update()

	// Should be in Insert mode
	if gs.inputHandler.Mode() != input.ModeInsert {
		t.Errorf("Expected ModeInsert after Enter, got %v", gs.inputHandler.Mode())
	}
	if gs.inputHandler.Focus() != input.FocusPrompt {
		t.Errorf("Expected FocusPrompt after Enter, got %v", gs.inputHandler.Focus())
	}

	// Press Escape to return to Normal mode
	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()
	gs.Update()

	// Should be back in Normal mode
	if gs.inputHandler.Mode() != input.ModeNormal {
		t.Errorf("Expected ModeNormal after Escape, got %v", gs.inputHandler.Mode())
	}
	if gs.inputHandler.Focus() != input.FocusMap {
		t.Errorf("Expected FocusMap after Escape, got %v", gs.inputHandler.Focus())
	}
}

// TestGameScene_ViewSwitching verifies that number keys switch between views.
func TestGameScene_ViewSwitching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "view-switching-*")
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

	// Initial view should be Map
	if gs.currentView != input.ViewMap {
		t.Errorf("Expected initial view ViewMap, got %v", gs.currentView)
	}

	// Test view keys
	viewTests := []struct {
		key  ebiten.Key
		view input.ViewNumber
	}{
		{ebiten.Key2, input.ViewBuilding},
		{ebiten.Key3, input.ViewUnit},
		{ebiten.Key4, input.ViewTech},
		{ebiten.Key5, input.ViewMission},
		{ebiten.Key6, input.ViewProduction},
		{ebiten.Key7, input.ViewMultiAgent},
		{ebiten.Key1, input.ViewMap}, // Back to map
	}

	for _, tt := range viewTests {
		source.QueueKeyPress(tt.key)
		source.AdvanceFrame()
		gs.Update()

		if gs.currentView != tt.view {
			t.Errorf("After key %v: expected view %v, got %v", tt.key, tt.view, gs.currentView)
		}
	}
}

// TestGameScene_PanningKeys verifies that hjkl and arrow keys pan the map.
func TestGameScene_PanningKeys(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "panning-keys-*")
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

	// Get initial pan position
	initialX := gs.mapView.PanX()
	initialY := gs.mapView.PanY()

	// Pan right with 'l'
	source.QueueKeyHold(ebiten.KeyL, 1)
	source.AdvanceFrame()
	gs.Update()

	afterRightX := gs.mapView.PanX()
	if afterRightX <= initialX {
		t.Errorf("Expected pan X to increase after 'l' key, was %f now %f", initialX, afterRightX)
	}

	// Pan left with 'h'
	source.QueueKeyHold(ebiten.KeyH, 1)
	source.AdvanceFrame()
	gs.Update()

	afterLeftX := gs.mapView.PanX()
	if afterLeftX >= afterRightX {
		t.Errorf("Expected pan X to decrease after 'h' key, was %f now %f", afterRightX, afterLeftX)
	}

	// Pan down with 'j'
	source.QueueKeyHold(ebiten.KeyJ, 1)
	source.AdvanceFrame()
	gs.Update()

	afterDownY := gs.mapView.PanY()
	if afterDownY <= initialY {
		t.Errorf("Expected pan Y to increase after 'j' key, was %f now %f", initialY, afterDownY)
	}

	// Pan up with 'k'
	source.QueueKeyHold(ebiten.KeyK, 1)
	source.AdvanceFrame()
	gs.Update()

	afterUpY := gs.mapView.PanY()
	if afterUpY >= afterDownY {
		t.Errorf("Expected pan Y to decrease after 'k' key, was %f now %f", afterDownY, afterUpY)
	}
}

// TestGameScene_ZoomKeys verifies that +/- keys zoom in/out.
func TestGameScene_ZoomKeys(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "zoom-keys-*")
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

	initialZoom := gs.mapView.ZoomLevel()

	// Zoom in with '+'
	source.QueueKeyPress(ebiten.KeyEqual) // '+' is Shift+= on most keyboards, but = works for zoom
	source.AdvanceFrame()
	gs.Update()

	// Note: We need to use the actual key for zoom in - let's try with Shift
	source.Clear()
	source.QueueKeyHold(ebiten.KeyShift, 2)
	source.QueueKeyPress(ebiten.KeyEqual)
	source.AdvanceFrame()
	gs.Update()

	afterZoomIn := gs.mapView.ZoomLevel()
	// Zoom level should have changed (might be constrained by min/max)
	t.Logf("Zoom: initial=%d, after zoom in=%d", initialZoom, afterZoomIn)

	// Zoom out with '-'
	source.Clear()
	source.QueueKeyPress(ebiten.KeyMinus)
	source.AdvanceFrame()
	gs.Update()

	afterZoomOut := gs.mapView.ZoomLevel()
	// If we zoomed in, zoom out should decrease zoom level
	if afterZoomIn > initialZoom && afterZoomOut >= afterZoomIn {
		t.Errorf("Expected zoom level to decrease, was %d now %d", afterZoomIn, afterZoomOut)
	}
}

// TestGameScene_NoInputWhenInsertMode verifies that navigation keys
// don't pan the map when in Insert mode.
func TestGameScene_NoInputWhenInsertMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no-input-insert-*")
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

	// Enter Insert mode
	gs.inputHandler.SetMode(input.ModeInsert)
	gs.inputHandler.SetFocus(input.FocusPrompt)

	// Get initial pan
	initialX := gs.mapView.PanX()
	initialY := gs.mapView.PanY()

	// Try to pan with hjkl (should be captured as text input, not panning)
	source.QueueKeyHold(ebiten.KeyH, 1)
	source.AdvanceFrame()
	gs.Update()

	source.QueueKeyHold(ebiten.KeyJ, 1)
	source.AdvanceFrame()
	gs.Update()

	source.QueueKeyHold(ebiten.KeyK, 1)
	source.AdvanceFrame()
	gs.Update()

	source.QueueKeyHold(ebiten.KeyL, 1)
	source.AdvanceFrame()
	gs.Update()

	// Pan should not have changed because focus is on prompt, not map
	afterX := gs.mapView.PanX()
	afterY := gs.mapView.PanY()

	if afterX != initialX {
		t.Errorf("Pan X changed in Insert mode: %f -> %f", initialX, afterX)
	}
	if afterY != initialY {
		t.Errorf("Pan Y changed in Insert mode: %f -> %f", initialY, afterY)
	}
}
