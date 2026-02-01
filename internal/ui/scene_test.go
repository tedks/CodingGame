package ui

import "testing"

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
