package ui

import (
	"os"
	"testing"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// If no display is available, skip all tests in this package
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		// Print skip message and exit successfully
		println("Skipping ui tests: no display available (DISPLAY and WAYLAND_DISPLAY not set)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestSceneManagerBasics(t *testing.T) {
	// Test NewSceneManager with nil
	sm := NewSceneManager(nil, 800, 600)

	if sm.Current() != nil {
		t.Error("expected current scene to be nil")
	}

	w, h := sm.Dimensions()
	if w != 800 || h != 600 {
		t.Errorf("expected dimensions 800x600, got %dx%d", w, h)
	}
}

func TestSceneManager_UpdateNil(t *testing.T) {
	sm := NewSceneManager(nil, 800, 600)

	// Should not panic or error with nil scene
	err := sm.Update()
	if err != nil {
		t.Errorf("Update() returned error: %v", err)
	}
}

func TestSceneManager_DrawNil(t *testing.T) {
	sm := NewSceneManager(nil, 800, 600)

	// Should not panic with nil scene
	sm.Draw(nil)
}
