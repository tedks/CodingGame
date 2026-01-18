package testutil

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestGame wraps an ebiten.Game with screenshot capture capability.
// Use this in tests that need to capture the game screen.
type TestGame struct {
	inner       ebiten.Game
	frameCount  int
	targetFrame int
	screenshot  *Screenshot
	captureErr  error
	done        chan struct{}
	mu          sync.Mutex
}

// NewTestGame wraps an ebiten.Game for testing with screenshot support.
func NewTestGame(g ebiten.Game) *TestGame {
	return &TestGame{
		inner: g,
		done:  make(chan struct{}),
	}
}

// Update delegates to the inner game and handles screenshot capture.
func (t *TestGame) Update() error {
	t.mu.Lock()
	t.frameCount++
	currentFrame := t.frameCount
	targetFrame := t.targetFrame
	hasScreenshot := t.screenshot != nil
	t.mu.Unlock()

	// Update inner game
	if err := t.inner.Update(); err != nil {
		return err
	}

	// Check if we've reached the target frame AND have captured
	// We wait one extra frame after target to ensure Draw captured
	if targetFrame > 0 && currentFrame > targetFrame && hasScreenshot {
		// Signal that we're done
		select {
		case <-t.done:
			// Already closed
		default:
			close(t.done)
		}
		return ebiten.Termination
	}

	return nil
}

// Draw delegates to the inner game and captures screenshot when requested.
func (t *TestGame) Draw(screen *ebiten.Image) {
	// Draw inner game first
	t.inner.Draw(screen)

	t.mu.Lock()
	defer t.mu.Unlock()

	currentFrame := t.frameCount
	targetFrame := t.targetFrame

	// Capture screenshot at or after target frame
	if targetFrame > 0 && currentFrame >= targetFrame && t.screenshot == nil {
		t.screenshot = CaptureScreen(screen)
	}
}

// Layout delegates to the inner game.
func (t *TestGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return t.inner.Layout(outsideWidth, outsideHeight)
}

// CaptureAfterFrames configures the test game to capture a screenshot
// after the specified number of frames.
func (t *TestGame) CaptureAfterFrames(frames int) {
	t.mu.Lock()
	t.targetFrame = frames
	t.mu.Unlock()
}

// Screenshot returns the captured screenshot, or nil if not yet captured.
func (t *TestGame) Screenshot() *Screenshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.screenshot
}

// Done returns a channel that closes when the test game is done.
func (t *TestGame) Done() <-chan struct{} {
	return t.done
}

// FrameCount returns the current frame count.
func (t *TestGame) FrameCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.frameCount
}

// RunAndCapture runs the game for the specified number of frames and returns a screenshot.
// This is a convenience function that sets up window configuration and runs the game.
func RunAndCapture(g ebiten.Game, frames int, width, height int) (*Screenshot, error) {
	testGame := NewTestGame(g)
	testGame.CaptureAfterFrames(frames)

	// Configure window for testing
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("Test")

	// Run game until target frame
	if err := ebiten.RunGame(testGame); err != nil && err != ebiten.Termination {
		return nil, fmt.Errorf("game error: %w", err)
	}

	screenshot := testGame.Screenshot()
	if screenshot == nil {
		return nil, fmt.Errorf("screenshot not captured")
	}

	return screenshot, nil
}

// SimpleTestGame is a minimal game for testing screenshot capture.
type SimpleTestGame struct {
	width  int
	height int
	drawFn func(*ebiten.Image)
}

// NewSimpleTestGame creates a simple test game with custom draw function.
func NewSimpleTestGame(width, height int, drawFn func(*ebiten.Image)) *SimpleTestGame {
	return &SimpleTestGame{
		width:  width,
		height: height,
		drawFn: drawFn,
	}
}

// Update does nothing for simple test games.
func (s *SimpleTestGame) Update() error {
	return nil
}

// Draw calls the custom draw function.
func (s *SimpleTestGame) Draw(screen *ebiten.Image) {
	if s.drawFn != nil {
		s.drawFn(screen)
	}
}

// Layout returns the configured dimensions.
func (s *SimpleTestGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return s.width, s.height
}
