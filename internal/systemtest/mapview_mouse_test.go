package systemtest

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/mapview"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/tile"
)

// MapView mouse interaction tests verify single-click selection, double-click callbacks,
// drag vs click threshold, and border gap hit detection.

const (
	mapViewTestWidth  = 800
	mapViewTestHeight = 600
)

type mouseReleaseAction struct {
	Button ebiten.MouseButton
}

func (a mouseReleaseAction) Apply(source *testutil.TestInputSource) {
	source.QueueMouseRelease(a.Button)
}

// createTestMapView creates a MapView with a temp directory containing test files.
// Returns the MapView and a cleanup function.
func createTestMapView(t *testing.T) (*mapview.MapView, func()) {
	t.Helper()

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mapview-mouse-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create some test files
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tmpDir, "other.go")
	if err := os.WriteFile(testFile2, []byte("package other"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create test file 2: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create a new map view
	mv, err := mapview.New(tmpDir, mapViewTestWidth, mapViewTestHeight)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create map view: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return mv, cleanup
}

func runScenarioOnMapView(t *testing.T, mv *mapview.MapView, source *testutil.TestInputSource, scenario *testutil.Scenario) {
	t.Helper()

	for i, step := range scenario.Steps {
		if step.Action != nil {
			step.Action.Apply(source)
		}

		frames := step.WaitFrames + 1
		for frame := 0; frame < frames; frame++ {
			source.AdvanceFrame()
			mv.Update()
		}

		if step.Assertion != nil {
			if err := step.Assertion(); err != nil {
				t.Fatalf("scenario %q step %d failed: %v", scenario.Name, i, err)
			}
		}
	}
}

func primeMapViewLayout(mv *mapview.MapView) {
	screen := ebiten.NewImage(mapViewTestWidth, mapViewTestHeight)
	mv.Draw(screen, 0, 0)
}

func findTileScreenPos(t *testing.T, mv *mapview.MapView) (int, int, *tile.Tile) {
	t.Helper()

	primeMapViewLayout(mv)

	const step = 4
	for y := 0; y < mapViewTestHeight; y += step {
		for x := 0; x < mapViewTestWidth; x += step {
			found := mv.TileAtScreenPos(x, y)
			if found != nil {
				return x, y, found
			}
		}
	}

	t.Fatalf("failed to find a tile within %dx%d viewport", mapViewTestWidth, mapViewTestHeight)
	return 0, 0, nil
}

func findTileBounds(t *testing.T, mv *mapview.MapView, startX, startY int, target *tile.Tile) (int, int, int, int) {
	t.Helper()

	minX, maxX := startX, startX
	for x := startX; x >= 0; x-- {
		if mv.TileAtScreenPos(x, startY) != target {
			break
		}
		minX = x
	}
	for x := startX; x < mapViewTestWidth; x++ {
		if mv.TileAtScreenPos(x, startY) != target {
			break
		}
		maxX = x
	}

	minY, maxY := startY, startY
	for y := startY; y >= 0; y-- {
		if mv.TileAtScreenPos(startX, y) != target {
			break
		}
		minY = y
	}
	for y := startY; y < mapViewTestHeight; y++ {
		if mv.TileAtScreenPos(startX, y) != target {
			break
		}
		maxY = y
	}

	return minX, maxX, minY, maxY
}

func findGapX(t *testing.T, mv *mapview.MapView, minX, maxX, y int) int {
	t.Helper()

	for x := maxX + 1; x < mapViewTestWidth && x <= maxX+5; x++ {
		if mv.TileAtScreenPos(x, y) == nil {
			return x
		}
	}
	for x := minX - 1; x >= 0 && x >= minX-5; x-- {
		if mv.TileAtScreenPos(x, y) == nil {
			return x
		}
	}

	t.Fatalf("unable to find border gap near tile at y=%d", y)
	return 0
}

func testMapViewSingleClickSelectsTile(t *testing.T) {
	mv, cleanup := createTestMapView(t)
	defer cleanup()

	source := testutil.NewTestInputSource()
	mv.SetInputSource(source)

	clickX, clickY, target := findTileScreenPos(t, mv)

	// Initially no selection
	if mv.SelectedTile() != nil {
		t.Error("expected no tile selected initially")
	}

	var selected *tile.Tile
	mv.SetOnTileSelect(func(t *tile.Tile) {
		selected = t
	})

	scenario := testutil.NewScenario("MapViewSingleClickSelectsTile")
	scenario.Move(clickX, clickY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if mv.SelectedTile() == nil {
			return fmt.Errorf("expected tile to be selected after click")
		}
		if mv.SelectedTile() != target {
			return fmt.Errorf("expected selected tile %q, got %q", target.Path(), mv.SelectedTile().Path())
		}
		if selected != target {
			return fmt.Errorf("expected select callback for %q", target.Path())
		}
		return nil
	})

	runScenarioOnMapView(t, mv, source, scenario)
}

func testMapViewDoubleClickTriggersCallback(t *testing.T) {
	mv, cleanup := createTestMapView(t)
	defer cleanup()

	source := testutil.NewTestInputSource()
	mv.SetInputSource(source)

	clickX, clickY, target := findTileScreenPos(t, mv)

	var doubleClickCount int
	var doubleClickTile *tile.Tile
	mv.SetOnTileDoubleClick(func(t *tile.Tile) {
		doubleClickCount++
		doubleClickTile = t
	})

	scenario := testutil.NewScenario("MapViewDoubleClickTriggersCallback")
	scenario.Move(clickX, clickY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if doubleClickCount != 1 {
			return fmt.Errorf("expected double-click callback once, got %d", doubleClickCount)
		}
		if doubleClickTile != target {
			return fmt.Errorf("expected double-click tile %q, got %v", target.Path(), doubleClickTile)
		}
		if mv.SelectedTile() != target {
			return fmt.Errorf("expected selected tile %q after double click", target.Path())
		}
		return nil
	})

	runScenarioOnMapView(t, mv, source, scenario)
}

func testMapViewDoubleClickUpdatesSelection(t *testing.T) {
	mv, cleanup := createTestMapView(t)
	defer cleanup()

	source := testutil.NewTestInputSource()
	mv.SetInputSource(source)

	clickX, clickY, target := findTileScreenPos(t, mv)

	// This test verifies that double-click also updates selectedTile
	// (the bug fix being tested)

	scenario := testutil.NewScenario("MapViewDoubleClickUpdatesSelection")
	scenario.Move(clickX, clickY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if mv.SelectedTile() != target {
			return fmt.Errorf("expected selected tile %q after first click", target.Path())
		}
		return nil
	})
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if mv.SelectedTile() != target {
			return fmt.Errorf("expected selected tile %q after double click", target.Path())
		}
		return nil
	})

	runScenarioOnMapView(t, mv, source, scenario)
}

func testMapViewDragVsClickThreshold(t *testing.T) {
	mv, cleanup := createTestMapView(t)
	defer cleanup()

	source := testutil.NewTestInputSource()
	mv.SetInputSource(source)

	startX, startY, _ := findTileScreenPos(t, mv)

	var selectCalled bool
	mv.SetOnTileSelect(func(t *tile.Tile) {
		selectCalled = true
	})

	scenario := testutil.NewScenario("MapViewDragVsClickThreshold")
	scenario.Move(startX, startY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.Move(startX+10, startY+10, 0)
	scenario.Move(startX+20, startY+20, 0)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if selectCalled {
			return fmt.Errorf("drag operation should not trigger tile selection")
		}
		if mv.SelectedTile() != nil {
			return fmt.Errorf("expected no tile selected after drag")
		}
		return nil
	})

	runScenarioOnMapView(t, mv, source, scenario)
}

func testMapViewClickInBorderGap(t *testing.T) {
	mv, cleanup := createTestMapView(t)
	defer cleanup()

	source := testutil.NewTestInputSource()
	mv.SetInputSource(source)

	clickX, clickY, target := findTileScreenPos(t, mv)
	minX, maxX, _, _ := findTileBounds(t, mv, clickX, clickY, target)
	gapX := findGapX(t, mv, minX, maxX, clickY)

	var selectCount int
	mv.SetOnTileSelect(func(t *tile.Tile) {
		selectCount++
	})

	scenario := testutil.NewScenario("MapViewClickInBorderGap")
	scenario.Move(clickX, clickY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if mv.SelectedTile() != target {
			return fmt.Errorf("expected selected tile %q before border gap click", target.Path())
		}
		if selectCount != 1 {
			return fmt.Errorf("expected one selection before border gap click, got %d", selectCount)
		}
		return nil
	})
	scenario.Move(gapX, clickY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if mv.SelectedTile() != nil {
			return fmt.Errorf("expected no tile selected after border gap click")
		}
		if selectCount != 1 {
			return fmt.Errorf("expected border gap click to not select a tile, got %d selections", selectCount)
		}
		return nil
	})

	runScenarioOnMapView(t, mv, source, scenario)
}

func testMapViewTripleClickBehavior(t *testing.T) {
	mv, cleanup := createTestMapView(t)
	defer cleanup()

	source := testutil.NewTestInputSource()
	mv.SetInputSource(source)

	clickX, clickY, target := findTileScreenPos(t, mv)

	// Track callback invocations
	var singleClickCount int
	var doubleClickCount int
	mv.SetOnTileSelect(func(t *tile.Tile) {
		singleClickCount++
	})
	mv.SetOnTileDoubleClick(func(t *tile.Tile) {
		doubleClickCount++
	})

	scenario := testutil.NewScenario("MapViewTripleClickBehavior")
	scenario.Move(clickX, clickY, 0)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(testutil.ClickMouse{Button: ebiten.MouseButtonLeft}, 0, nil)
	scenario.AddStep(mouseReleaseAction{Button: ebiten.MouseButtonLeft}, 0, func() error {
		if singleClickCount != 1 {
			return fmt.Errorf("expected 1 single-click callback, got %d", singleClickCount)
		}
		if doubleClickCount != 2 {
			return fmt.Errorf("expected 2 double-click callbacks, got %d", doubleClickCount)
		}
		if mv.SelectedTile() != target {
			return fmt.Errorf("expected selected tile %q after triple click", target.Path())
		}
		return nil
	})

	runScenarioOnMapView(t, mv, source, scenario)
}
