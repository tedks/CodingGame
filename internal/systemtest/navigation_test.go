package systemtest

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
)

// Navigation tests verify hjkl keys, arrow keys, and zoom controls.

func runScenarioOnHandler(t *testing.T, h *input.Handler, source *testutil.TestInputSource, scenario *testutil.Scenario) {
	t.Helper()

	for i, step := range scenario.Steps {
		if step.Action != nil {
			step.Action.Apply(source)
		}

		frames := step.WaitFrames + 1
		for frame := 0; frame < frames; frame++ {
			source.AdvanceFrame()
			h.Update()
		}

		if step.Assertion != nil {
			if err := step.Assertion(); err != nil {
				t.Fatalf("scenario %q step %d failed: %v", scenario.Name, i, err)
			}
		}
	}
}

func testNavigationHKeyPansLeft(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	scenario := testutil.NewScenario("NavigationHKeyPansLeft")
	scenario.PressWithCheck(ebiten.KeyH, 0, func() error {
		if !h.IsActionHeld(input.ActionMoveLeft) {
			return fmt.Errorf("expected ActionMoveLeft to be held after H key press")
		}
		return nil
	})
	scenario.WaitWithCheck(0, func() error {
		if h.IsActionHeld(input.ActionMoveLeft) {
			return fmt.Errorf("expected ActionMoveLeft to be released after key press frame")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}

func testNavigationJKeyPansDown(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	scenario := testutil.NewScenario("NavigationJKeyPansDown")
	scenario.PressWithCheck(ebiten.KeyJ, 0, func() error {
		if !h.IsActionHeld(input.ActionMoveDown) {
			return fmt.Errorf("expected ActionMoveDown to be held after J key press")
		}
		return nil
	})
	scenario.WaitWithCheck(0, func() error {
		if h.IsActionHeld(input.ActionMoveDown) {
			return fmt.Errorf("expected ActionMoveDown to be released after key press frame")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}

func testNavigationKKeyPansUp(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	scenario := testutil.NewScenario("NavigationKKeyPansUp")
	scenario.PressWithCheck(ebiten.KeyK, 0, func() error {
		if !h.IsActionHeld(input.ActionMoveUp) {
			return fmt.Errorf("expected ActionMoveUp to be held after K key press")
		}
		return nil
	})
	scenario.WaitWithCheck(0, func() error {
		if h.IsActionHeld(input.ActionMoveUp) {
			return fmt.Errorf("expected ActionMoveUp to be released after key press frame")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}

func testNavigationLKeyPansRight(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	// Verify initial state
	assertMode(t, h, input.ModeNormal)

	scenario := testutil.NewScenario("NavigationLKeyPansRight")
	scenario.PressWithCheck(ebiten.KeyL, 0, func() error {
		if !h.IsActionHeld(input.ActionMoveRight) {
			return fmt.Errorf("expected ActionMoveRight to be held after L key press")
		}
		return nil
	})
	scenario.WaitWithCheck(0, func() error {
		if h.IsActionHeld(input.ActionMoveRight) {
			return fmt.Errorf("expected ActionMoveRight to be released after key press frame")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}

func testNavigationArrowKeys(t *testing.T) {
	// Test each arrow key
	arrowTests := []struct {
		name   string
		key    ebiten.Key
		action input.Action
	}{
		{"Left", ebiten.KeyArrowLeft, input.ActionMoveLeft},
		{"Right", ebiten.KeyArrowRight, input.ActionMoveRight},
		{"Up", ebiten.KeyArrowUp, input.ActionMoveUp},
		{"Down", ebiten.KeyArrowDown, input.ActionMoveDown},
	}

	for _, tt := range arrowTests {
		t.Run(tt.name, func(t *testing.T) {
			source := testutil.NewTestInputSource()
			h := testHandler(source)

			scenario := testutil.NewScenario("NavigationArrowKey" + tt.name)
			scenario.PressWithCheck(tt.key, 0, func() error {
				if !h.IsActionHeld(tt.action) {
					return fmt.Errorf("expected %v to be held after %s arrow key press", tt.action, tt.name)
				}
				return nil
			})
			scenario.WaitWithCheck(0, func() error {
				if h.IsActionHeld(tt.action) {
					return fmt.Errorf("expected %v to be released after %s arrow key press frame", tt.action, tt.name)
				}
				return nil
			})

			runScenarioOnHandler(t, h, source, scenario)
		})
	}
}

func testNavigationPlusKeyZoomsIn(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	scenario := testutil.NewScenario("NavigationPlusKeyZoomsIn")
	scenario.PressWithCheck(ebiten.KeyEqual, 0, func() error {
		if !h.IsActionHeld(input.ActionZoomIn) {
			return fmt.Errorf("expected ActionZoomIn to be held after + key press")
		}
		return nil
	})
	scenario.WaitWithCheck(0, func() error {
		if h.IsActionHeld(input.ActionZoomIn) {
			return fmt.Errorf("expected ActionZoomIn to be released after key press frame")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}

func testNavigationMinusKeyZoomsOut(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	scenario := testutil.NewScenario("NavigationMinusKeyZoomsOut")
	scenario.PressWithCheck(ebiten.KeyMinus, 0, func() error {
		if !h.IsActionHeld(input.ActionZoomOut) {
			return fmt.Errorf("expected ActionZoomOut to be held after - key press")
		}
		return nil
	})
	scenario.WaitWithCheck(0, func() error {
		if h.IsActionHeld(input.ActionZoomOut) {
			return fmt.Errorf("expected ActionZoomOut to be released after key press frame")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}

func testNavigationHeldKeys(t *testing.T) {
	source := testutil.NewTestInputSource()
	h := testHandler(source)

	heldCheck := func(frame int) func() error {
		return func() error {
			if !source.IsKeyPressed(ebiten.KeyH) {
				return fmt.Errorf("frame %d: expected H key to be held", frame)
			}
			if !h.IsActionHeld(input.ActionMoveLeft) {
				return fmt.Errorf("frame %d: expected ActionMoveLeft to be held", frame)
			}
			return nil
		}
	}

	scenario := testutil.NewScenario("NavigationHeldKeys")
	scenario.AddStep(testutil.HoldKey{Key: ebiten.KeyH, Duration: 5}, 0, heldCheck(0))
	for frame := 1; frame < 5; frame++ {
		frame := frame
		scenario.WaitWithCheck(0, heldCheck(frame))
	}
	scenario.WaitWithCheck(0, func() error {
		if source.IsKeyPressed(ebiten.KeyH) {
			return fmt.Errorf("expected H key to be released after hold duration")
		}
		if h.IsActionHeld(input.ActionMoveLeft) {
			return fmt.Errorf("expected ActionMoveLeft to be released after hold duration")
		}
		return nil
	})

	runScenarioOnHandler(t, h, source, scenario)
}
