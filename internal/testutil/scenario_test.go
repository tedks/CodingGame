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
