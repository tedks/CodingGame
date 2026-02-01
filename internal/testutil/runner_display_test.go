package testutil

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func requireDisplay(t *testing.T) {
	t.Helper()
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		t.Skip("Skipping: no display available (set DISPLAY or WAYLAND_DISPLAY)")
	}
}

// Integration tests that require a running game are below.
// These use a single test to avoid GLFW reinitialization issues.
func TestRunAndCaptureIntegration(t *testing.T) {
	requireDisplay(t)

	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a simple game that draws a colored background
	drawCalled := false
	game := NewSimpleTestGame(100, 100, func(screen *ebiten.Image) {
		drawCalled = true
		// Just clear screen - don't need to fill with color for this test
	})

	// Run for a few frames and capture
	screenshot, err := RunAndCapture(game, 3, 100, 100)

	// Note: This test may fail if GLFW was already initialized in another test.
	// That's expected - run this test in isolation if needed.
	if err != nil {
		// Check if it's a GLFW error - if so, skip gracefully
		if isGLFWError(err) {
			t.Skipf("Skipping: GLFW initialization issue (expected if other tests ran first): %v", err)
		}
		t.Fatalf("RunAndCapture() error: %v", err)
	}

	if !drawCalled {
		t.Error("Draw function was not called")
	}

	if screenshot == nil {
		t.Fatal("Screenshot is nil")
	}

	// Verify dimensions
	if screenshot.Width() != 100 || screenshot.Height() != 100 {
		t.Errorf("Screenshot dimensions = (%d, %d), want (100, 100)",
			screenshot.Width(), screenshot.Height())
	}

	// Verify we can get the underlying image
	if screenshot.Image() == nil {
		t.Error("Screenshot.Image() returned nil")
	}
}

// isGLFWError checks if an error is related to GLFW initialization.
func isGLFWError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "glfw") || contains(errStr, "GLFW")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
