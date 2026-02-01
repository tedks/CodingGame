package testutil

import "testing"

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
