package testutil

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestScenario_Builder(t *testing.T) {
	s := NewScenario("Test Scenario")

	// Add steps using builder methods
	s.Press(ebiten.KeyA, 1)
	s.Hold(ebiten.KeyB, 5, 2)
	s.Type("hello", 1)
	s.WaitFrames(10)
	s.Click(ebiten.MouseButtonLeft, 1)
	s.Move(100, 200, 1)
	s.Scroll(0, -1.0, 1)

	// Verify steps were added
	if len(s.Steps) != 7 {
		t.Errorf("expected 7 steps, got %d", len(s.Steps))
	}
}

func TestScenario_StepWithCheck(t *testing.T) {
	checkCalled := false

	s := NewScenario("Test Scenario")
	s.PressWithCheck(ebiten.KeyA, 1, func() error {
		checkCalled = true
		return nil
	})

	// Verify step has assertion
	if s.Steps[0].Assertion == nil {
		t.Error("expected step to have assertion")
	}

	// Run assertion
	err := s.Steps[0].Assertion()
	if err != nil {
		t.Errorf("assertion failed: %v", err)
	}
	if !checkCalled {
		t.Error("expected assertion to be called")
	}
}

func TestScenarioResult_Failed(t *testing.T) {
	// Test with no failures
	result := &ScenarioResult{
		StepResults: []StepResult{
			{StepIndex: 0, FrameCount: 1, AssertionError: nil},
			{StepIndex: 1, FrameCount: 2, AssertionError: nil},
		},
	}
	if result.Failed() {
		t.Error("expected no failures")
	}

	// Test with failure
	result.StepResults = append(result.StepResults, StepResult{
		StepIndex:      2,
		FrameCount:     3,
		AssertionError: fmt.Errorf("test error"),
	})
	if !result.Failed() {
		t.Error("expected failure")
	}

	// Test FirstError
	err := result.FirstError()
	if err == nil {
		t.Error("expected FirstError to return an error")
	}
}

func TestAction_PressKey(t *testing.T) {
	source := NewTestInputSource()

	action := PressKey{Key: ebiten.KeyA}
	action.Apply(source)

	// Event should be queued
	if !source.HasPendingEvents() {
		t.Error("expected pending events after action apply")
	}

	source.AdvanceFrame()

	// Key should be pressed
	if !source.IsKeyJustPressed(ebiten.KeyA) {
		t.Error("expected KeyA to be just pressed")
	}
}

func TestAction_HoldKey(t *testing.T) {
	source := NewTestInputSource()

	action := HoldKey{Key: ebiten.KeyH, Duration: 3}
	action.Apply(source)
	source.AdvanceFrame()

	// Key should be held
	if !source.IsKeyPressed(ebiten.KeyH) {
		t.Error("expected KeyH to be pressed")
	}
}

func TestAction_TypeText(t *testing.T) {
	source := NewTestInputSource()

	action := TypeText{Text: "hello"}
	action.Apply(source)
	source.AdvanceFrame()

	// Characters should be queued
	chars := source.AppendInputChars(nil)
	if string(chars) != "hello" {
		t.Errorf("expected 'hello', got %q", string(chars))
	}
}

func TestAction_MoveMouse(t *testing.T) {
	source := NewTestInputSource()

	action := MoveMouse{X: 100, Y: 200}
	action.Apply(source)
	source.AdvanceFrame()

	x, y := source.CursorPosition()
	if x != 100 || y != 200 {
		t.Errorf("expected (100, 200), got (%d, %d)", x, y)
	}
}

func TestAction_ClickMouse(t *testing.T) {
	source := NewTestInputSource()

	action := ClickMouse{Button: ebiten.MouseButtonLeft}
	action.Apply(source)
	source.AdvanceFrame()

	if !source.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		t.Error("expected left button to be just pressed")
	}
}

func TestAction_ScrollWheel(t *testing.T) {
	source := NewTestInputSource()

	action := ScrollWheel{X: 0, Y: -1.5}
	action.Apply(source)
	source.AdvanceFrame()

	xoff, yoff := source.Wheel()
	if xoff != 0 || yoff != -1.5 {
		t.Errorf("expected (0, -1.5), got (%v, %v)", xoff, yoff)
	}
}

func TestAction_Wait(t *testing.T) {
	source := NewTestInputSource()

	action := Wait{}
	action.Apply(source)

	// Wait should not queue any events
	if source.HasPendingEvents() {
		t.Error("Wait action should not queue events")
	}
}

func TestAction_PressKeys(t *testing.T) {
	source := NewTestInputSource()

	action := PressKeys{Keys: []ebiten.Key{ebiten.KeyA, ebiten.KeyB, ebiten.KeyC}}
	action.Apply(source)
	source.AdvanceFrame()

	// All keys should be pressed
	for _, key := range []ebiten.Key{ebiten.KeyA, ebiten.KeyB, ebiten.KeyC} {
		if !source.IsKeyJustPressed(key) {
			t.Errorf("expected key %v to be just pressed", key)
		}
	}
}

// Edge case tests

func TestScenario_EmptySteps(t *testing.T) {
	s := NewScenario("Empty Scenario")
	if len(s.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(s.Steps))
	}

	result := &ScenarioResult{StepResults: nil}
	if result.Failed() {
		t.Error("Empty scenario should not be failed")
	}
	if err := result.FirstError(); err != nil {
		t.Errorf("Empty scenario should have no errors, got %v", err)
	}
}

func TestScenario_OnlyWaitActions(t *testing.T) {
	s := NewScenario("Wait Only")
	s.WaitFrames(5)
	s.WaitFrames(10)
	s.WaitFrames(3)

	if len(s.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(s.Steps))
	}
	for i, step := range s.Steps {
		if _, ok := step.Action.(Wait); !ok {
			t.Errorf("step %d should have Wait action", i)
		}
	}
}

func TestScenario_AssertionError(t *testing.T) {
	result := &ScenarioResult{
		StepResults: []StepResult{
			{StepIndex: 0, FrameCount: 1, AssertionError: nil},
			{StepIndex: 1, FrameCount: 5, AssertionError: fmt.Errorf("expected foo")},
			{StepIndex: 2, FrameCount: 10, AssertionError: nil},
		},
	}

	if !result.Failed() {
		t.Error("Should be considered failed")
	}
	err := result.FirstError()
	if err == nil {
		t.Fatal("Expected FirstError to return an error")
	}
}

func TestScenario_AssertionPanic(t *testing.T) {
	s := NewScenario("Panic Test")
	s.AddStep(Wait{}, 1, func() error {
		panic("intentional panic")
	})

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		_ = s.Steps[0].Assertion()
	}()

	if !didPanic {
		t.Error("Expected assertion to panic")
	}
}

type minimalGame struct {
	width, height int
}

func (m *minimalGame) Update() error                                  { return nil }
func (m *minimalGame) Draw(screen *ebiten.Image)                      {}
func (m *minimalGame) Layout(outsideWidth, outsideHeight int) (int, int) { return m.width, m.height }

func TestScenarioGame_NoSetInputSource(t *testing.T) {
	game := &minimalGame{width: 100, height: 100}
	sg := NewScenarioGame(game, NewScenario("Test"), 100, 100)

	if sg == nil {
		t.Fatal("NewScenarioGame returned nil")
	}
	if sg.InputSource() == nil {
		t.Fatal("InputSource should still be created")
	}
}

func TestScenario_ChainedBuilderMethods(t *testing.T) {
	s := NewScenario("Chained").
		Press(ebiten.KeyA, 1).
		Hold(ebiten.KeyB, 3, 1).
		Type("test", 1).
		WaitFrames(5).
		Click(ebiten.MouseButtonLeft, 1).
		Move(100, 100, 1).
		Scroll(0, -1.0, 1)

	if len(s.Steps) != 7 {
		t.Errorf("expected 7 steps, got %d", len(s.Steps))
	}
}

func TestScenario_AddStepWithNilAction(t *testing.T) {
	s := NewScenario("Nil Action")
	s.AddStep(nil, 5, nil)

	if len(s.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(s.Steps))
	}
	if s.Steps[0].Action != nil {
		t.Error("expected nil action")
	}
	if s.Steps[0].WaitFrames != 5 {
		t.Errorf("expected WaitFrames=5, got %d", s.Steps[0].WaitFrames)
	}
}
