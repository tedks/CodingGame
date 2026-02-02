package testutil

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

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

// Lifecycle tests

type errorGame struct {
	width, height   int
	errorAfterFrame int
	err             error
	frameCount      int
}

func (e *errorGame) Update() error {
	e.frameCount++
	if e.frameCount > e.errorAfterFrame {
		return e.err
	}
	return nil
}

func (e *errorGame) Draw(screen *ebiten.Image) {}

func (e *errorGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return e.width, e.height
}

func TestTestGame_InnerGameError(t *testing.T) {
	expectedErr := fmt.Errorf("inner game error")
	game := &errorGame{width: 100, height: 100, errorAfterFrame: 3, err: expectedErr}

	testGame := NewTestGame(game)
	testGame.CaptureAfterFrames(10)

	for i := 0; i < 5; i++ {
		err := testGame.Update()
		if i == 3 && err != expectedErr {
			t.Errorf("Expected error on frame %d", i)
		}
		if err != nil {
			break
		}
	}
}

type terminatingGame struct {
	width, height       int
	terminateAfterFrame int
	frameCount          int
}

func (tg *terminatingGame) Update() error {
	tg.frameCount++
	if tg.frameCount > tg.terminateAfterFrame {
		return ebiten.Termination
	}
	return nil
}

func (tg *terminatingGame) Draw(screen *ebiten.Image) {}

func (tg *terminatingGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return tg.width, tg.height
}

func TestTestGame_InnerGameTerminates(t *testing.T) {
	game := &terminatingGame{width: 100, height: 100, terminateAfterFrame: 2}
	testGame := NewTestGame(game)
	testGame.CaptureAfterFrames(10)

	for i := 0; i < 5; i++ {
		err := testGame.Update()
		if err == ebiten.Termination {
			break
		}
	}
}

func TestTestGame_DoneChannelSafety(t *testing.T) {
	game := NewSimpleTestGame(10, 10, nil)
	testGame := NewTestGame(game)
	testGame.CaptureAfterFrames(1)

	for i := 0; i < 10; i++ {
		err := testGame.Update()
		if err == ebiten.Termination {
			break
		}
		testGame.Draw(ebiten.NewImage(10, 10))
	}

	select {
	case <-testGame.Done():
		// Expected
	default:
		t.Error("Done channel should be closed")
	}
}

func TestTestGame_LayoutDelegation(t *testing.T) {
	inner := NewSimpleTestGame(800, 600, nil)
	testGame := NewTestGame(inner)

	w, h := testGame.Layout(1920, 1080)
	if w != 800 || h != 600 {
		t.Errorf("Layout() = (%d, %d), want (800, 600)", w, h)
	}
}

func TestTestGame_ScreenshotNilBeforeTarget(t *testing.T) {
	game := NewSimpleTestGame(10, 10, nil)
	testGame := NewTestGame(game)
	testGame.CaptureAfterFrames(10)

	if testGame.Screenshot() != nil {
		t.Error("Screenshot should be nil before updates")
	}

	for i := 0; i < 5; i++ {
		testGame.Update()
	}

	if testGame.Screenshot() != nil {
		t.Error("Screenshot should be nil before target frame")
	}
}
