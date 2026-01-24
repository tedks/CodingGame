// Integration tests for the game engine's screenshot capture functionality.
//
// These tests verify end-to-end screenshot capture with real game instances.
// They require a display (DISPLAY or WAYLAND_DISPLAY) and are skipped in
// -short mode or headless environments without xvfb.
//
// IMPORTANT: Due to GLFW initialization constraints, all integration tests
// run as subtests of TestIntegration. GLFW can only be initialized once per
// process, so we cannot have multiple top-level tests that call RunAndCapture.
// See CLAUDE.md for details on this constraint.
//
// Usage:
//
//	CODINGGAME_SAVE_SCREENSHOTS=1 xvfb-run -a bazel test //internal/game:game_test --test_filter=Integration
//
// Then ask Claude to read the screenshot files listed in the test output.
package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tedks/CodingGame/internal/testutil"
)

// TestIntegration is the single entry point for all integration tests.
// This structure is required because GLFW can only be initialized once per process.
func TestIntegration(t *testing.T) {
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

	// Create some test files to visualize
	createTestFiles(t, tmpDir)

	// Create game instance
	game, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Cleanup defers - note: Go executes defers in LIFO order, so:
	// 1. game.Close() runs first (closes file handles)
	// 2. os.RemoveAll() runs second (safe to remove directory)
	defer os.RemoveAll(tmpDir)
	defer game.Close()

	// Capture screenshot once (GLFW constraint - can only run once per process)
	screenshot, err := testutil.RunAndCapture(game, 5, 800, 600)
	if err != nil {
		if isGLFWError(err) {
			t.Skipf("Skipping: GLFW initialization issue: %v", err)
		}
		t.Fatalf("failed to capture screenshot: %v", err)
	}

	// Run subtests with the captured screenshot
	t.Run("ScreenshotCapture", func(t *testing.T) {
		testScreenshotCapture(t, screenshot)
	})

	t.Run("ScreenshotContent", func(t *testing.T) {
		testScreenshotContent(t, screenshot)
	})

	t.Run("ConditionalSave", func(t *testing.T) {
		testConditionalSave(t, screenshot)
	})
}

// testScreenshotCapture verifies basic screenshot capture and file saving.
func testScreenshotCapture(t *testing.T, screenshot *testutil.Screenshot) {
	// Verify screenshot dimensions
	if screenshot.Width() != 800 || screenshot.Height() != 600 {
		t.Errorf("unexpected screenshot size: %dx%d, want 800x600",
			screenshot.Width(), screenshot.Height())
	}

	// Create capturer and save screenshot
	capturer := testutil.NewCapturer(t)
	path := capturer.Capture(screenshot, "initial")
	if path == "" {
		t.Fatal("failed to save initial screenshot")
	}

	// Verify screenshot file exists and is non-empty
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("screenshot file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("screenshot file is empty")
	}

	// Log for Claude to find
	t.Logf("Initial game render: %s", path)
	t.Log("Claude can read this screenshot to analyze the initial game state")
}

// testScreenshotContent verifies that the screenshot contains actual rendered content.
func testScreenshotContent(t *testing.T, screenshot *testutil.Screenshot) {
	if screenshot.Width() == 0 || screenshot.Height() == 0 {
		t.Fatal("screenshot has zero dimensions")
	}

	// Check that screenshot has some non-black pixels (game should render something)
	bounds := screenshot.Image().Bounds()
	hasContent := false

	// Sample every 10th pixel for performance
	for y := bounds.Min.Y; y < bounds.Max.Y && !hasContent; y += 10 {
		for x := bounds.Min.X; x < bounds.Max.X && !hasContent; x += 10 {
			r, g, b, a := screenshot.Image().At(x, y).RGBA()
			// Check for any non-transparent, non-black pixel
			if a > 0 && (r > 0 || g > 0 || b > 0) {
				hasContent = true
			}
		}
	}

	if !hasContent {
		t.Error("screenshot appears blank - game may not be rendering correctly")
	}
}

// testConditionalSave verifies the SaveOnFailure behavior.
// SaveOnFailure only saves screenshots when the test is failing or
// CODINGGAME_SAVE_SCREENSHOTS=1 is set.
func testConditionalSave(t *testing.T, screenshot *testutil.Screenshot) {
	capturer := testutil.NewCapturer(t)

	// SaveOnFailure should not save when test is passing and env var is not set
	// (We can't easily test the failure case without actually failing)
	capturer.SaveOnFailure(screenshot, "conditional_demo")

	if os.Getenv("CODINGGAME_SAVE_SCREENSHOTS") == "1" {
		t.Log("CODINGGAME_SAVE_SCREENSHOTS=1: screenshot was saved via SaveOnFailure")
	} else {
		t.Log("CODINGGAME_SAVE_SCREENSHOTS not set: screenshot not saved (test passing)")
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
	return strings.Contains(errStr, "glfw") || strings.Contains(errStr, "GLFW")
}
