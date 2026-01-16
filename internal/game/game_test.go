package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "codinggame-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a new game instance
	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	defer game.Close()

	// Verify game was initialized properly
	if game == nil {
		t.Fatal("expected non-nil game")
	}
	if game.width != 800 {
		t.Errorf("expected width 800, got %d", game.width)
	}
	if game.height != 600 {
		t.Errorf("expected height 600, got %d", game.height)
	}
	if game.mapView == nil {
		t.Error("expected non-nil mapView")
	}
	if game.resources == nil {
		t.Error("expected non-nil resources")
	}
	if game.interceptor == nil {
		t.Error("expected non-nil interceptor")
	}
}

func TestLayout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codinggame-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	defer game.Close()

	// Test Layout method
	width, height := game.Layout(1024, 768)
	if width != 800 {
		t.Errorf("expected width 800, got %d", width)
	}
	if height != 600 {
		t.Errorf("expected height 600, got %d", height)
	}
}

func TestUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codinggame-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	defer game.Close()

	// Test Update method (should not error)
	err = game.Update()
	if err != nil {
		t.Errorf("Update() returned error: %v", err)
	}

	// Update should be callable multiple times
	for i := 0; i < 10; i++ {
		if err := game.Update(); err != nil {
			t.Errorf("Update() iteration %d returned error: %v", i, err)
		}
	}
}

func TestHandleClaudeEvent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codinggame-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	defer game.Close()

	// Test FileRead event
	game.interceptor.SimulateFileRead("test.go")

	// Test FileWrite event
	game.interceptor.SimulateFileWrite("newfile.go")

	// Test FileEdit event
	game.interceptor.SimulateFileEdit("test.go")

	// Give events time to process
	// Note: In a real test, you'd want to verify the effects, but that
	// requires checking map state which isn't currently exposed
}

func TestClose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codinggame-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Test Close method
	err = game.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	// Closing twice should not panic
	err = game.Close()
	if err != nil {
		t.Errorf("Close() second call returned error: %v", err)
	}
}

func TestConstants(t *testing.T) {
	if ResourceBarHeight != 40 {
		t.Errorf("expected ResourceBarHeight 40, got %d", ResourceBarHeight)
	}
	if PanSpeed != 5 {
		t.Errorf("expected PanSpeed 5, got %d", PanSpeed)
	}
}
