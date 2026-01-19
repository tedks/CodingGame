package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/testutil"
)

// TestIntegrationScreenshotCapture demonstrates the iterative visual debugging workflow.
// This test captures screenshots at various game states for Claude to analyze.
//
// Usage:
//
//	CODINGGAME_SAVE_SCREENSHOTS=1 xvfb-run -a bazel test //internal/game:game_test --test_filter=Integration
//
// Then ask Claude to read the screenshot files listed in the test output.
func TestIntegrationScreenshotCapture(t *testing.T) {
	// Skip if no display available
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("Skipping integration test: no display available")
	}

	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "codinggame-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files to visualize
	createTestFiles(t, tmpDir)

	// Create game instance
	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
	defer game.Close()

	// Create capturer for this test
	capturer := testutil.NewCapturer(t)

	// Capture initial state
	screenshot1, err := testutil.RunAndCapture(game, 5, 800, 600)
	if err != nil {
		// Check if it's a GLFW error - skip gracefully
		if isGLFWError(err) {
			t.Skipf("Skipping: GLFW initialization issue: %v", err)
		}
		t.Fatalf("failed to capture initial state: %v", err)
	}

	path1 := capturer.Capture(screenshot1, "initial")
	if path1 == "" {
		t.Error("failed to save initial screenshot")
	}

	// Log for Claude to find
	t.Logf("Initial game render: %s", path1)
	t.Log("Claude can read this screenshot to analyze the initial game state")
}

// TestIntegrationSaveOnFailure demonstrates the SaveOnFailure pattern.
// Screenshots are only saved when the test fails or CODINGGAME_SAVE_SCREENSHOTS=1.
func TestIntegrationSaveOnFailure(t *testing.T) {
	// Skip if no display available
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("Skipping integration test: no display available")
	}

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "codinggame-onfailure-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	createTestFiles(t, tmpDir)

	// Create a simple test game (avoids GLFW issues with multiple tests)
	simpleGame := testutil.NewSimpleTestGame(800, 600, func(screen *ebiten.Image) {
		// Draw a simple colored background
		screen.Clear()
	})

	capturer := testutil.NewCapturer(t)

	// This screenshot is only saved on failure or when CODINGGAME_SAVE_SCREENSHOTS=1
	screenshot, err := testutil.RunAndCapture(simpleGame, 3, 800, 600)
	if err != nil {
		if isGLFWError(err) {
			t.Skipf("Skipping: GLFW initialization issue: %v", err)
		}
		t.Fatalf("failed to capture: %v", err)
	}

	// This will only save if the test is failing
	capturer.SaveOnFailure(screenshot, "on_failure_demo")

	// Check if screenshot was saved based on env var
	if os.Getenv("CODINGGAME_SAVE_SCREENSHOTS") == "1" {
		t.Log("CODINGGAME_SAVE_SCREENSHOTS=1, screenshot was saved")
	} else {
		t.Log("Screenshot not saved (test passing and CODINGGAME_SAVE_SCREENSHOTS not set)")
	}
}

// createTestFiles creates a directory structure for testing.
func createTestFiles(t *testing.T, dir string) {
	t.Helper()

	// Create a simple Go project structure
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, CodingGame!")
}
`,
		"util/helper.go": `package util

func Helper() string {
	return "helping"
}
`,
		"internal/core.go": `package internal

type Core struct {
	Name string
}
`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", path, err)
		}
	}
}

// isGLFWError checks if an error is related to GLFW initialization.
func isGLFWError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsString(errStr, "glfw") || containsString(errStr, "GLFW")
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
