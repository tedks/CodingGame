package systemtest

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/game"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/ui"
)

func runScenarioOnScene(t *testing.T, scene *game.GameScene, source *testutil.TestInputSource, scenario *testutil.Scenario) {
	t.Helper()

	for i, step := range scenario.Steps {
		if step.Action != nil {
			step.Action.Apply(source)
		}

		frames := step.WaitFrames + 1
		for frame := 0; frame < frames; frame++ {
			source.AdvanceFrame()
			if _, err := scene.Update(); err != nil {
				t.Fatalf("scenario %q update failed: %v", scenario.Name, err)
			}
		}

		if step.Assertion != nil {
			if err := step.Assertion(); err != nil {
				t.Fatalf("scenario %q step %d failed: %v", scenario.Name, i, err)
			}
		}
	}
}

func testGameScenePromptPanelDragUsesInputSource(t *testing.T) {
	tmpDir := t.TempDir()

	scene, err := game.NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create game scene: %v", err)
	}
	defer scene.Close()

	source := testutil.NewTestInputSource()
	scene.SetInputSource(source)

	// Prime the prompt panel layout before dragging.
	source.AdvanceFrame()
	if _, err := scene.Update(); err != nil {
		t.Fatalf("initial update failed: %v", err)
	}

	initialHeight := scene.PromptPanelHeight()
	handleX := 400
	handleY := 600 - initialHeight + ui.DragHandleHeight/2
	dragY := handleY - 80

	scenario := testutil.NewScenario("GameScenePromptPanelDragUsesInputSource")
	scenario.AddStep(testutil.MoveMouse{X: handleX, Y: handleY}, 0, nil)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(testutil.MoveMouse{X: handleX, Y: dragY}, 0, func() error {
		updatedHeight := scene.PromptPanelHeight()
		if updatedHeight <= initialHeight {
			return fmt.Errorf("expected prompt panel height to increase after drag: initial=%d updated=%d", initialHeight, updatedHeight)
		}
		return nil
	})

	runScenarioOnScene(t, scene, source, scenario)
}
