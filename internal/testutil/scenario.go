package testutil

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Action represents an input action in a scenario.
type Action interface {
	// Apply queues the action's events on the TestInputSource.
	Apply(source *TestInputSource)
}

// PressKey is an action that presses a single key.
type PressKey struct {
	Key ebiten.Key
}

// Apply implements Action.
func (p PressKey) Apply(source *TestInputSource) {
	source.QueueKeyPress(p.Key)
}

// HoldKey is an action that holds a key for multiple frames.
type HoldKey struct {
	Key      ebiten.Key
	Duration int // frames
}

// Apply implements Action.
func (h HoldKey) Apply(source *TestInputSource) {
	source.QueueKeyHold(h.Key, h.Duration)
}

// TypeText is an action that types a string of text.
type TypeText struct {
	Text string
}

// Apply implements Action.
func (t TypeText) Apply(source *TestInputSource) {
	source.QueueTextInput(t.Text)
}

// MoveMouse is an action that moves the mouse cursor.
type MoveMouse struct {
	X, Y int
}

// Apply implements Action.
func (m MoveMouse) Apply(source *TestInputSource) {
	source.QueueMouseMove(m.X, m.Y)
}

// ClickMouse is an action that clicks a mouse button.
type ClickMouse struct {
	Button ebiten.MouseButton
}

// Apply implements Action.
func (c ClickMouse) Apply(source *TestInputSource) {
	source.QueueMouseClick(c.Button)
}

// ScrollWheel is an action that scrolls the mouse wheel.
type ScrollWheel struct {
	X, Y float64
}

// Apply implements Action.
func (s ScrollWheel) Apply(source *TestInputSource) {
	source.QueueMouseWheel(s.X, s.Y)
}

// PressKeys is an action that presses multiple keys simultaneously.
type PressKeys struct {
	Keys []ebiten.Key
}

// Apply implements Action.
func (p PressKeys) Apply(source *TestInputSource) {
	for _, key := range p.Keys {
		source.QueueKeyPress(key)
	}
}

// Wait is a no-op action used to wait for frames without input.
type Wait struct{}

// Apply implements Action.
func (w Wait) Apply(source *TestInputSource) {
	// No-op: just waiting
}

// Step represents a single step in a scenario.
type Step struct {
	// Action to perform (can be nil for wait-only steps)
	Action Action

	// WaitFrames is the number of frames to wait after the action.
	// The action is applied on frame 0, then we wait WaitFrames more frames.
	WaitFrames int

	// Assertion is an optional check to run after the step completes.
	// Return an error if the assertion fails.
	Assertion func() error
}

// Scenario represents a sequence of input actions and assertions.
type Scenario struct {
	Name  string
	Steps []Step
}

// ScenarioResult contains the results of running a scenario.
type ScenarioResult struct {
	// StepResults contains the result of each step
	StepResults []StepResult

	// TotalFrames is the total number of frames executed
	TotalFrames int

	// FinalScreenshot is the screenshot at the end of the scenario
	FinalScreenshot *Screenshot
}

// StepResult contains the result of a single step.
type StepResult struct {
	// StepIndex is the index of this step in the scenario
	StepIndex int

	// FrameCount is the frame number when this step completed
	FrameCount int

	// AssertionError is the error from the assertion, if any
	AssertionError error
}

// Failed returns true if any step's assertion failed.
func (r *ScenarioResult) Failed() bool {
	for _, step := range r.StepResults {
		if step.AssertionError != nil {
			return true
		}
	}
	return false
}

// FirstError returns the first assertion error, or nil if all passed.
func (r *ScenarioResult) FirstError() error {
	for _, step := range r.StepResults {
		if step.AssertionError != nil {
			return fmt.Errorf("step %d failed: %w", step.StepIndex, step.AssertionError)
		}
	}
	return nil
}

// Builder helpers for creating scenarios

// NewScenario creates a new scenario with the given name.
func NewScenario(name string) *Scenario {
	return &Scenario{Name: name}
}

// AddStep adds a step to the scenario.
func (s *Scenario) AddStep(action Action, waitFrames int, assertion func() error) *Scenario {
	s.Steps = append(s.Steps, Step{
		Action:     action,
		WaitFrames: waitFrames,
		Assertion:  assertion,
	})
	return s
}

// Press adds a key press step.
func (s *Scenario) Press(key ebiten.Key, waitFrames int) *Scenario {
	return s.AddStep(PressKey{Key: key}, waitFrames, nil)
}

// PressWithCheck adds a key press step with an assertion.
func (s *Scenario) PressWithCheck(key ebiten.Key, waitFrames int, check func() error) *Scenario {
	return s.AddStep(PressKey{Key: key}, waitFrames, check)
}

// Hold adds a key hold step.
func (s *Scenario) Hold(key ebiten.Key, duration, waitFrames int) *Scenario {
	return s.AddStep(HoldKey{Key: key, Duration: duration}, waitFrames, nil)
}

// Type adds a text typing step.
func (s *Scenario) Type(text string, waitFrames int) *Scenario {
	return s.AddStep(TypeText{Text: text}, waitFrames, nil)
}

// TypeWithCheck adds a text typing step with an assertion.
func (s *Scenario) TypeWithCheck(text string, waitFrames int, check func() error) *Scenario {
	return s.AddStep(TypeText{Text: text}, waitFrames, check)
}

// WaitFrames adds a wait step without any input.
func (s *Scenario) WaitFrames(frames int) *Scenario {
	return s.AddStep(Wait{}, frames, nil)
}

// WaitWithCheck adds a wait step with an assertion.
func (s *Scenario) WaitWithCheck(frames int, check func() error) *Scenario {
	return s.AddStep(Wait{}, frames, check)
}

// Click adds a mouse click step.
func (s *Scenario) Click(button ebiten.MouseButton, waitFrames int) *Scenario {
	return s.AddStep(ClickMouse{Button: button}, waitFrames, nil)
}

// Move adds a mouse move step.
func (s *Scenario) Move(x, y, waitFrames int) *Scenario {
	return s.AddStep(MoveMouse{X: x, Y: y}, waitFrames, nil)
}

// Scroll adds a mouse wheel scroll step.
func (s *Scenario) Scroll(xoff, yoff float64, waitFrames int) *Scenario {
	return s.AddStep(ScrollWheel{X: xoff, Y: yoff}, waitFrames, nil)
}
