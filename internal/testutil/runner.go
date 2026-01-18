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

// InputSourceSetter is an interface for objects that can have their input source set.
type InputSourceSetter interface {
	SetInputSource(source interface{})
}

// ScenarioGame wraps an ebiten.Game for scenario-based testing.
// It manages input injection and step execution.
type ScenarioGame struct {
	inner       ebiten.Game
	inputSource *TestInputSource
	scenario    *Scenario
	result      *ScenarioResult
	width       int
	height      int

	// Execution state
	mu          sync.Mutex
	frameCount  int
	currentStep int
	stepFrame   int // frame within current step
	done        bool
	screenshot  *Screenshot
	stepResults []StepResult
	terminated  bool
}

// NewScenarioGame creates a new scenario game wrapper.
func NewScenarioGame(g ebiten.Game, scenario *Scenario, width, height int) *ScenarioGame {
	inputSource := NewTestInputSource()

	// Try to inject the input source into the game
	injectInputSource(g, inputSource)

	return &ScenarioGame{
		inner:       g,
		inputSource: inputSource,
		scenario:    scenario,
		width:       width,
		height:      height,
		currentStep: 0,
		stepFrame:   0,
	}
}

// injectInputSource attempts to inject the TestInputSource into the game.
// It uses reflection to find SetInputSource methods.
func injectInputSource(g interface{}, source *TestInputSource) {
	// Try direct interface assertion first
	if setter, ok := g.(interface{ SetInputSource(interface{}) }); ok {
		setter.SetInputSource(source)
		return
	}

	// Try specific known types via type assertion
	// This avoids import cycles while still allowing injection
	type inputSourceSetter interface {
		SetInputSource(s interface{})
	}
	if setter, ok := g.(inputSourceSetter); ok {
		setter.SetInputSource(source)
	}
}

// Update executes the scenario step by step.
func (sg *ScenarioGame) Update() error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if sg.terminated {
		return ebiten.Termination
	}

	sg.frameCount++

	// Process input events before calling inner Update
	sg.inputSource.AdvanceFrame()

	// Execute current step logic
	if sg.currentStep < len(sg.scenario.Steps) {
		step := sg.scenario.Steps[sg.currentStep]

		// On first frame of step, apply the action
		if sg.stepFrame == 0 && step.Action != nil {
			step.Action.Apply(sg.inputSource)
		}

		sg.stepFrame++

		// Check if step is complete (waited enough frames)
		if sg.stepFrame > step.WaitFrames {
			// Run assertion if present
			var assertErr error
			if step.Assertion != nil {
				assertErr = step.Assertion()
			}

			sg.stepResults = append(sg.stepResults, StepResult{
				StepIndex:      sg.currentStep,
				FrameCount:     sg.frameCount,
				AssertionError: assertErr,
			})

			// Move to next step
			sg.currentStep++
			sg.stepFrame = 0
		}
	}

	// Update inner game
	if err := sg.inner.Update(); err != nil {
		if err == ebiten.Termination {
			sg.terminated = true
			return err
		}
		return err
	}

	// Check if scenario is complete
	if sg.currentStep >= len(sg.scenario.Steps) && !sg.done {
		sg.done = true
		// Don't terminate yet - wait for Draw to capture screenshot
	}

	return nil
}

// Draw delegates to the inner game and captures final screenshot.
func (sg *ScenarioGame) Draw(screen *ebiten.Image) {
	sg.inner.Draw(screen)

	sg.mu.Lock()
	defer sg.mu.Unlock()

	// Capture screenshot when done
	if sg.done && sg.screenshot == nil {
		sg.screenshot = CaptureScreen(screen)
		sg.terminated = true
	}
}

// Layout delegates to the inner game.
func (sg *ScenarioGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return sg.inner.Layout(outsideWidth, outsideHeight)
}

// Result returns the scenario result after execution.
func (sg *ScenarioGame) Result() *ScenarioResult {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	return &ScenarioResult{
		StepResults:     sg.stepResults,
		TotalFrames:     sg.frameCount,
		FinalScreenshot: sg.screenshot,
	}
}

// InputSource returns the test input source for direct manipulation.
func (sg *ScenarioGame) InputSource() *TestInputSource {
	return sg.inputSource
}

// RunScenario runs a scenario against a game and returns the result.
// This is the main entry point for scenario-based system tests.
func RunScenario(g ebiten.Game, scenario *Scenario, width, height int) (*ScenarioResult, error) {
	sg := NewScenarioGame(g, scenario, width, height)

	// Configure window for testing
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("Scenario: " + scenario.Name)

	// Run the game
	if err := ebiten.RunGame(sg); err != nil && err != ebiten.Termination {
		return nil, fmt.Errorf("scenario %q failed: %w", scenario.Name, err)
	}

	return sg.Result(), nil
}

// RunScenarioWithInput runs a scenario and allows direct input source access.
// Returns both the result and the scenario game for inspection.
func RunScenarioWithInput(g ebiten.Game, scenario *Scenario, width, height int) (*ScenarioGame, *ScenarioResult, error) {
	sg := NewScenarioGame(g, scenario, width, height)

	// Configure window for testing
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("Scenario: " + scenario.Name)

	// Run the game
	if err := ebiten.RunGame(sg); err != nil && err != ebiten.Termination {
		return sg, nil, fmt.Errorf("scenario %q failed: %w", scenario.Name, err)
	}

	return sg, sg.Result(), nil
}
