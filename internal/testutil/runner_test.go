package testutil

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMain(m *testing.M) {
	// Skip tests if no display is available
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		println("Skipping testutil tests: no display available")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestTestGameFrameCount tests the frame counter without running the full game.
func TestTestGameFrameCount(t *testing.T) {
	game := NewSimpleTestGame(10, 10, nil)
	testGame := NewTestGame(game)

	if testGame.FrameCount() != 0 {
		t.Errorf("Initial FrameCount() = %d, want 0", testGame.FrameCount())
	}

	// Simulate update calls (doesn't require full game loop)
	for i := 0; i < 5; i++ {
		testGame.Update()
	}

	if testGame.FrameCount() != 5 {
		t.Errorf("After 5 updates, FrameCount() = %d, want 5", testGame.FrameCount())
	}
}

// TestTestGameCaptureConfig tests screenshot configuration.
func TestTestGameCaptureConfig(t *testing.T) {
	game := NewSimpleTestGame(10, 10, nil)
	testGame := NewTestGame(game)

	// Initially no screenshot
	if testGame.Screenshot() != nil {
		t.Error("Initial Screenshot() should be nil")
	}

	// Configure capture
	testGame.CaptureAfterFrames(10)

	// Done channel should not be closed initially
	select {
	case <-testGame.Done():
		t.Error("Done() channel should not be closed initially")
	default:
		// Expected
	}
}

// TestSimpleTestGameLayout tests the layout function.
func TestSimpleTestGameLayout(t *testing.T) {
	game := NewSimpleTestGame(640, 480, nil)

	w, h := game.Layout(800, 600)
	if w != 640 || h != 480 {
		t.Errorf("Layout() = (%d, %d), want (640, 480)", w, h)
	}
}

// TestNewTestGame tests TestGame creation.
func TestNewTestGame(t *testing.T) {
	inner := NewSimpleTestGame(100, 100, nil)
	testGame := NewTestGame(inner)

	if testGame == nil {
		t.Fatal("NewTestGame() returned nil")
	}
	if testGame.FrameCount() != 0 {
		t.Errorf("FrameCount() = %d, want 0", testGame.FrameCount())
	}
}

// Integration tests that require a running game are below.
// These use a single test to avoid GLFW reinitialization issues.

// TestRunAndCaptureIntegration is an integration test that runs the full game loop.
// It's structured as a single test to work around GLFW's one-init-per-process limitation.
func TestRunAndCaptureIntegration(t *testing.T) {
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
