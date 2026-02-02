package mapview

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/tile"
)

// Test helper functions

// createManyTiles creates n test tiles for performance testing.
func createManyTiles(n int) []*tile.Tile {
	tiles := make([]*tile.Tile, n)
	for i := 0; i < n; i++ {
		path := filepath.Join("/test", "file"+string(rune('a'+i%26))+string(rune('0'+i/26%10))+".go")
		tiles[i] = tile.New(path, filepath.Base(path), false)
	}
	return tiles
}

// assertFloatNear fails if got and want differ by more than tolerance.
func assertFloatNear(t *testing.T, got, want, tolerance float64, msg string) {
	t.Helper()
	if diff := math.Abs(got - want); diff > tolerance {
		t.Errorf("%s: got %.6f, want %.6f (diff %.6f > tolerance %.6f)", msg, got, want, diff, tolerance)
	}
}

func TestNew(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	// Create a new map view
	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Verify map view was initialized properly
	if mapView == nil {
		t.Fatal("expected non-nil map view")
	}
	if mapView.width != 800 {
		t.Errorf("expected width 800, got %d", mapView.width)
	}
	if mapView.height != 600 {
		t.Errorf("expected height 600, got %d", mapView.height)
	}
	if mapView.zoomLevel != ZoomWorld {
		t.Errorf("expected zoom level ZoomWorld, got %d", mapView.zoomLevel)
	}
	if len(mapView.tiles) == 0 {
		t.Error("expected tiles to be populated")
	}
	if mapView.tileMap == nil {
		t.Fatal("expected non-nil tileMap")
	}
	if len(mapView.tileMap) != len(mapView.tiles) {
		t.Errorf("tileMap length %d doesn't match tiles length %d", len(mapView.tileMap), len(mapView.tiles))
	}
}

func TestPan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Test panning (Pan is inverted - subtracts from pan values to move camera)
	initialX := mapView.PanX()
	initialY := mapView.PanY()

	mapView.Pan(10, 20)

	if mapView.PanX() != initialX-10 {
		t.Errorf("expected panX %f, got %f", initialX-10, mapView.PanX())
	}
	if mapView.PanY() != initialY-20 {
		t.Errorf("expected panY %f, got %f", initialY-20, mapView.PanY())
	}

	// Test negative pan
	mapView.Pan(-5, -10)

	if mapView.PanX() != initialX-5 {
		t.Errorf("expected panX %f, got %f", initialX-5, mapView.PanX())
	}
	if mapView.PanY() != initialY-10 {
		t.Errorf("expected panY %f, got %f", initialY-10, mapView.PanY())
	}
}

func TestZoom(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Initial zoom level should be ZoomWorld
	if mapView.ZoomLevel() != int(ZoomWorld) {
		t.Errorf("expected initial zoom %d, got %d", ZoomWorld, mapView.ZoomLevel())
	}

	// Test zoom in
	mapView.ZoomIn()
	if mapView.ZoomLevel() != int(ZoomRegion) {
		t.Errorf("expected zoom %d, got %d", ZoomRegion, mapView.ZoomLevel())
	}

	// Test multiple zoom in
	mapView.ZoomIn()
	mapView.ZoomIn()
	mapView.ZoomIn()

	if mapView.ZoomLevel() != int(ZoomInterior) {
		t.Errorf("expected zoom %d, got %d", ZoomInterior, mapView.ZoomLevel())
	}

	// Test zoom in at max level (should not go higher)
	mapView.ZoomIn()
	if mapView.ZoomLevel() != int(ZoomInterior) {
		t.Errorf("expected zoom to stay at %d, got %d", ZoomInterior, mapView.ZoomLevel())
	}

	// Test zoom out
	mapView.ZoomOut()
	if mapView.ZoomLevel() != int(ZoomStreet) {
		t.Errorf("expected zoom %d, got %d", ZoomStreet, mapView.ZoomLevel())
	}

	// Test zoom out to minimum
	for i := 0; i < 10; i++ {
		mapView.ZoomOut()
	}

	if mapView.ZoomLevel() != int(ZoomOverview) {
		t.Errorf("expected zoom to stay at %d, got %d", ZoomOverview, mapView.ZoomLevel())
	}
}

func TestRevealTile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Test revealing an existing tile
	mapView.RevealTile(testFile)

	// Verify tile is in map
	tile, exists := mapView.tileMap[testFile]
	if !exists {
		t.Error("expected tile to exist in tileMap")
	}
	if tile != nil && !tile.IsRevealed() {
		t.Error("expected tile to be revealed")
	}

	// Test revealing non-existent tile (should not panic)
	mapView.RevealTile("/nonexistent/file.go")
}

func TestUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Test Update method (should not panic)
	mapView.Update()

	// Update should be callable multiple times
	for i := 0; i < 10; i++ {
		mapView.Update()
	}
}

func TestGetTileSize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Test tile sizes at different zoom levels
	tests := []struct {
		zoom     ZoomLevel
		expected float64
	}{
		{ZoomOverview, 40},
		{ZoomWorld, 64},
		{ZoomRegion, 80},
		{ZoomCity, 100},
		{ZoomStreet, 120},
		{ZoomInterior, 140},
	}

	for _, tt := range tests {
		mapView.zoomLevel = tt.zoom
		size := mapView.getTileSize()
		if size != tt.expected {
			t.Errorf("zoom %d: expected tile size %f, got %f", tt.zoom, tt.expected, size)
		}
	}
}

func TestDotfileFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory (should be skipped)
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	// Create .github directory (should be included)
	githubDir := filepath.Join(tmpDir, ".github")
	if err := os.Mkdir(githubDir, 0755); err != nil {
		t.Fatalf("failed to create .github dir: %v", err)
	}

	// Create .gitignore file (should be included)
	gitignoreFile := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignoreFile, []byte("*.log"), 0644); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	// Create .random hidden file (should be skipped)
	randomFile := filepath.Join(tmpDir, ".random")
	if err := os.WriteFile(randomFile, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create .random: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Verify .git is not in tiles
	if _, exists := mapView.tileMap[gitDir]; exists {
		t.Error("expected .git directory to be skipped")
	}

	// Verify .github is in tiles
	if _, exists := mapView.tileMap[githubDir]; !exists {
		t.Error("expected .github directory to be included")
	}

	// Verify .gitignore is in tiles
	if _, exists := mapView.tileMap[gitignoreFile]; !exists {
		t.Error("expected .gitignore file to be included")
	}

	// Verify .random is not in tiles
	if _, exists := mapView.tileMap[randomFile]; exists {
		t.Error("expected .random file to be skipped")
	}
}

func TestClampUint8(t *testing.T) {
	tests := []struct {
		input    int
		expected uint8
	}{
		{-10, 0},
		{0, 0},
		{100, 100},
		{255, 255},
		{300, 255},
		{1000, 255},
	}

	for _, tt := range tests {
		result := clampUint8(tt.input)
		if result != tt.expected {
			t.Errorf("clampUint8(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestConstants(t *testing.T) {
	if TileBorderSpacing != 2 {
		t.Errorf("expected TileBorderSpacing 2, got %d", TileBorderSpacing)
	}
	if BorderColorBoost != 20 {
		t.Errorf("expected BorderColorBoost 20, got %d", BorderColorBoost)
	}
	if CharWidth != 6 {
		t.Errorf("expected CharWidth 6, got %d", CharWidth)
	}
	if LineHeight != 14 {
		t.Errorf("expected LineHeight 14, got %d", LineHeight)
	}
}

func TestViewMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Initial view mode should be Directory
	if mapView.ViewMode() != ViewDirectory {
		t.Errorf("expected initial view mode ViewDirectory, got %v", mapView.ViewMode())
	}

	// Test SetViewMode
	mapView.SetViewMode(ViewDataflow)
	if mapView.ViewMode() != ViewDataflow {
		t.Errorf("expected view mode ViewDataflow, got %v", mapView.ViewMode())
	}

	mapView.SetViewMode(ViewDirectory)
	if mapView.ViewMode() != ViewDirectory {
		t.Errorf("expected view mode ViewDirectory, got %v", mapView.ViewMode())
	}
}

func TestToggleViewMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Initial state: Directory
	if mapView.ViewMode() != ViewDirectory {
		t.Errorf("expected initial view mode ViewDirectory, got %v", mapView.ViewMode())
	}

	// Toggle to Dataflow
	mapView.ToggleViewMode()
	if mapView.ViewMode() != ViewDataflow {
		t.Errorf("expected view mode ViewDataflow after toggle, got %v", mapView.ViewMode())
	}

	// Toggle back to Directory
	mapView.ToggleViewMode()
	if mapView.ViewMode() != ViewDirectory {
		t.Errorf("expected view mode ViewDirectory after second toggle, got %v", mapView.ViewMode())
	}

	// Multiple toggles
	for i := 0; i < 10; i++ {
		mapView.ToggleViewMode()
	}
	// After even number of toggles, should be back to original (Directory)
	if mapView.ViewMode() != ViewDirectory {
		t.Errorf("expected view mode ViewDirectory after even toggles, got %v", mapView.ViewMode())
	}
}

func TestViewModeString(t *testing.T) {
	tests := []struct {
		mode     ViewMode
		expected string
	}{
		{ViewDirectory, "Directory"},
		{ViewDataflow, "Dataflow"},
		{ViewMode(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.expected {
				t.Errorf("ViewMode(%d).String() = %q, want %q", tt.mode, got, tt.expected)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0K"},
		{1536, "1.5K"},
		{10240, "10.0K"},
		{1048576, "1.0M"},    // 1 MB
		{1572864, "1.5M"},    // 1.5 MB
		{1073741824, "1.0G"}, // 1 GB
		{1610612736, "1.5G"}, // 1.5 GB
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{"zero time", time.Time{}, ""},
		{"just now", now.Add(-30 * time.Second), "now"},
		{"minutes ago", now.Add(-5 * time.Minute), "5m"},
		{"hours ago", now.Add(-3 * time.Hour), "3h"},
		{"days ago", now.Add(-2 * 24 * time.Hour), "2d"},
		{"weeks ago", now.Add(-14 * 24 * time.Hour), "2w"},
		{"months ago", now.Add(-45 * 24 * time.Hour), "1mo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRelativeTime(tt.time)
			if result != tt.expected {
				t.Errorf("formatRelativeTime() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSelectedTile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Initially no selection
	if mapView.SelectedTile() != nil {
		t.Error("expected nil selected tile initially")
	}

	// Get the tile from the map
	testTile, exists := mapView.tileMap[testFile]
	if !exists {
		t.Fatal("expected test tile to exist")
	}

	// Set selection
	mapView.SetSelectedTile(testTile)
	if mapView.SelectedTile() != testTile {
		t.Error("expected selected tile to match set tile")
	}

	// Clear selection
	mapView.SetSelectedTile(nil)
	if mapView.SelectedTile() != nil {
		t.Error("expected nil selected tile after clearing")
	}
}

func TestTileSelectionCallbacks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Track callback invocations
	var selectCalled bool
	var doubleClickCalled bool

	mapView.SetOnTileSelect(func(t *tile.Tile) {
		selectCalled = true
	})
	mapView.SetOnTileDoubleClick(func(t *tile.Tile) {
		doubleClickCalled = true
	})

	// Get the tile
	testTile, _ := mapView.tileMap[testFile]

	// Simulate a single click (sets selectedTile and calls callback)
	mapView.selectedTile = testTile
	if mapView.onTileSelect != nil {
		mapView.onTileSelect(testTile)
	}

	if !selectCalled {
		t.Error("expected select callback to be called")
	}

	// Simulate a double click
	if mapView.onTileDoubleClick != nil {
		mapView.onTileDoubleClick(testTile)
	}

	if !doubleClickCalled {
		t.Error("expected double-click callback to be called")
	}
}

func TestZoomCentering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Set initial pan position to test centering behavior
	initialPanX := -100.0
	initialPanY := -50.0
	mapView.panX = initialPanX
	mapView.panY = initialPanY

	// Record center of screen before zoom
	centerX := float64(mapView.width) / 2
	centerY := float64(mapView.height) / 2

	// Calculate world coordinate of center before zoom
	oldSize := mapView.getTileSize()
	worldCenterXBefore := (centerX - mapView.panX) / oldSize
	worldCenterYBefore := (centerY - mapView.panY) / oldSize

	// Zoom in
	mapView.ZoomIn()

	// Calculate world coordinate of center after zoom
	newSize := mapView.getTileSize()
	worldCenterXAfter := (centerX - mapView.panX) / newSize
	worldCenterYAfter := (centerY - mapView.panY) / newSize

	// The tile coordinate at center should be approximately the same
	// Allow small floating point tolerance
	tolerance := 0.01
	if diff := worldCenterXBefore - worldCenterXAfter; diff > tolerance || diff < -tolerance {
		t.Errorf("center X shifted: before=%.4f, after=%.4f, diff=%.4f", worldCenterXBefore, worldCenterXAfter, diff)
	}
	if diff := worldCenterYBefore - worldCenterYAfter; diff > tolerance || diff < -tolerance {
		t.Errorf("center Y shifted: before=%.4f, after=%.4f, diff=%.4f", worldCenterYBefore, worldCenterYAfter, diff)
	}
}

func TestAdjustPanForZoom(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	// Test that adjustPanForZoom preserves the center point correctly
	// Formula: panX' = panX * ratio + centerX * (1 - ratio)
	// Where ratio = newSize / oldSize

	tests := []struct {
		name         string
		initialPanX  float64
		initialPanY  float64
		oldSize      float64
		newSize      float64
		expectedPanX float64
		expectedPanY float64
	}{
		{
			name:         "zoom in 2x",
			initialPanX:  0,
			initialPanY:  0,
			oldSize:      64,
			newSize:      128,
			expectedPanX: -400, // 0 * 2 + 400 * (1-2) = -400 for width 800
			expectedPanY: -300, // 0 * 2 + 300 * (1-2) = -300 for height 600
		},
		{
			name:         "zoom out 0.5x",
			initialPanX:  -200,
			initialPanY:  -150,
			oldSize:      128,
			newSize:      64,
			expectedPanX: 100, // -200 * 0.5 + 400 * 0.5 = -100 + 200 = 100
			expectedPanY: 75,  // -150 * 0.5 + 300 * 0.5 = -75 + 150 = 75
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapView.panX = tt.initialPanX
			mapView.panY = tt.initialPanY
			mapView.adjustPanForZoom(tt.oldSize, tt.newSize)

			if mapView.panX != tt.expectedPanX {
				t.Errorf("panX = %.2f, want %.2f", mapView.panX, tt.expectedPanX)
			}
			if mapView.panY != tt.expectedPanY {
				t.Errorf("panY = %.2f, want %.2f", mapView.panY, tt.expectedPanY)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-100, 100},
		{100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestDrawDataflowViewTilesPerRowZeroGuard(t *testing.T) {
	m := &MapView{
		width:     0,
		height:    100,
		zoomLevel: ZoomOverview,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("drawDataflowView panicked: %v", r)
		}
	}()

	m.drawDataflowView(nil, 0, 0)
}

// --- Coordinate Math Tests ---

func TestZoomCyclePrecisionDrift(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	mapView.panX = 100.0
	mapView.panY = 75.0
	initialX := mapView.panX
	initialY := mapView.panY

	for i := 0; i < 100; i++ {
		mapView.ZoomIn()
		mapView.ZoomOut()
	}

	driftX := math.Abs(mapView.panX - initialX)
	driftY := math.Abs(mapView.panY - initialY)
	totalDrift := driftX + driftY

	if totalDrift > 1.0 {
		t.Errorf("Pan drifted %.6f pixels after 100 zoom cycles (X: %.6f, Y: %.6f)",
			totalDrift, driftX, driftY)
	}
}

func TestTileAtScreenPos_NegativeCoordinates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	result := mapView.TileAtScreenPos(-100, -100)
	if result != nil {
		t.Error("expected nil tile at negative coordinates")
	}

	result = mapView.TileAtScreenPos(-1000000, -1000000)
	if result != nil {
		t.Error("expected nil tile at extreme negative coordinates")
	}
}

func TestTileAtScreenPos_ExtremePositiveCoordinates(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	result := mapView.TileAtScreenPos(1<<20, 1<<20)
	if result != nil {
		t.Error("expected nil tile at extreme positive coordinates")
	}

	result = mapView.TileAtScreenPos(math.MaxInt32, math.MaxInt32)
	if result != nil {
		t.Error("expected nil tile at MaxInt32 coordinates")
	}
}

func TestEffectiveSizeNeverNegative(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	zoomLevels := []ZoomLevel{ZoomOverview, ZoomWorld, ZoomRegion, ZoomCity, ZoomStreet, ZoomInterior}
	for _, zoom := range zoomLevels {
		mapView.zoomLevel = zoom
		tileSize := mapView.getTileSize()
		effectiveSize := tileSize - TileBorderSpacing

		if effectiveSize <= 0 {
			t.Errorf("effectiveSize <= 0 at zoom %d: tileSize=%.2f", zoom, tileSize)
		}
	}
}

// --- Zoom/Pan Boundary Tests ---

func TestZoomLevelSymmetry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	for startZoom := ZoomOverview; startZoom <= ZoomInterior; startZoom++ {
		mapView.zoomLevel = startZoom
		mapView.panX = 100.0
		mapView.panY = 50.0

		initialPanX := mapView.panX
		initialPanY := mapView.panY

		if startZoom < ZoomInterior {
			mapView.ZoomIn()
			mapView.ZoomOut()

			if mapView.zoomLevel != startZoom {
				t.Errorf("zoom level not symmetric: started at %d, ended at %d", startZoom, mapView.zoomLevel)
			}

			assertFloatNear(t, mapView.panX, initialPanX, 0.001, "panX not symmetric after ZoomIn+ZoomOut")
			assertFloatNear(t, mapView.panY, initialPanY, 0.001, "panY not symmetric after ZoomIn+ZoomOut")
		}

		mapView.zoomLevel = startZoom
		mapView.panX = initialPanX
		mapView.panY = initialPanY

		if startZoom > ZoomOverview {
			mapView.ZoomOut()
			mapView.ZoomIn()

			if mapView.zoomLevel != startZoom {
				t.Errorf("zoom level not symmetric: started at %d, ended at %d", startZoom, mapView.zoomLevel)
			}

			assertFloatNear(t, mapView.panX, initialPanX, 0.001, "panX not symmetric after ZoomOut+ZoomIn")
			assertFloatNear(t, mapView.panY, initialPanY, 0.001, "panY not symmetric after ZoomOut+ZoomIn")
		}
	}
}

func TestPanBoundsAtExtremeZoom(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 10; i++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('a'+i))+".go")
		if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	mapView.zoomLevel = ZoomOverview

	extremePans := []struct {
		panX, panY float64
	}{
		{10000, 10000},
		{-10000, -10000},
		{0, 0},
		{1000000, 0},
		{0, 1000000},
	}

	for _, ep := range extremePans {
		mapView.panX = ep.panX
		mapView.panY = ep.panY

		tileSize := mapView.getTileSize()
		viewX := -mapView.panX
		viewY := -mapView.panY - float64(TopPadding)*tileSize
		visible := mapView.treeLayout.VisibleNodes(viewX, viewY, float64(mapView.width), float64(mapView.height))
		_ = visible
	}

	mapView.panX = 0
	mapView.panY = 0
	tileSize := mapView.getTileSize()
	viewX := -mapView.panX
	viewY := -mapView.panY - float64(TopPadding)*tileSize
	visible := mapView.treeLayout.VisibleNodes(viewX, viewY, float64(mapView.width), float64(mapView.height))

	if len(visible) == 0 {
		t.Error("expected some visible tiles at origin pan with ZoomOverview")
	}
}

// --- Performance Tests ---

func TestVisibleNodesCullingEfficiency(t *testing.T) {
	tiles := createManyTiles(10000)
	tileSize := 64.0
	layout := NewTreeLayout(tiles, tileSize)

	viewportWidth := 800.0
	viewportHeight := 600.0

	totalWidth := layout.TotalWidth()
	totalHeight := layout.TotalHeight()
	viewX := totalWidth/2 - viewportWidth/2
	viewY := totalHeight/2 - viewportHeight/2

	visible := layout.VisibleNodes(viewX, viewY, viewportWidth, viewportHeight)

	maxReasonableVisible := 200

	if len(visible) > maxReasonableVisible {
		t.Errorf("culling inefficient: %d tiles visible out of 10000 (expected < %d)",
			len(visible), maxReasonableVisible)
	}

	if len(visible) == 0 {
		t.Error("no tiles visible - culling may be too aggressive or layout is broken")
	}
}

func TestTileAtScreenPosPerformance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 100; i++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('a'+i%26))+string(rune('0'+i/26))+".go")
		if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	_ = mapView.TileAtScreenPos(100, 100)

	start := time.Now()
	for i := 0; i < 10000; i++ {
		x := (i * 7) % 800
		y := (i * 11) % 600
		_ = mapView.TileAtScreenPos(x, y)
	}
	elapsed := time.Since(start)

	maxDuration := 100 * time.Millisecond
	if elapsed > maxDuration {
		t.Errorf("TileAtScreenPos too slow: 10K lookups took %v (expected < %v)",
			elapsed, maxDuration)
	}
}

// --- Mouse Interaction Tests ---

func TestDragThresholdBoundary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	input := testutil.NewTestInputSource()
	mapView.SetInputSource(input)

	var selectCalled bool
	mapView.SetOnTileSelect(func(t *tile.Tile) {
		selectCalled = true
	})

	input.QueueMouseMove(100, 100)
	input.QueueMouseClick(ebiten.MouseButtonLeft)
	input.AdvanceFrame()
	mapView.Update()

	initialPanX := mapView.panX

	input.QueueMouseMove(100+DragThreshold-1, 100)
	input.AdvanceFrame()
	mapView.Update()

	input.QueueMouseRelease(ebiten.MouseButtonLeft)
	input.AdvanceFrame()
	mapView.Update()

	if mapView.panX != initialPanX {
		t.Errorf("pan changed during click (under threshold): delta = %.2f", mapView.panX-initialPanX)
	}

	selectCalled = false
	input.Clear()

	input.QueueMouseMove(200, 200)
	input.QueueMouseClick(ebiten.MouseButtonLeft)
	input.AdvanceFrame()
	mapView.Update()

	initialPanX = mapView.panX

	input.QueueMouseMove(200+DragThreshold+1, 200)
	input.AdvanceFrame()
	mapView.Update()

	if mapView.panX == initialPanX {
		t.Error("pan did not change during drag (over threshold)")
	}

	input.QueueMouseRelease(ebiten.MouseButtonLeft)
	input.AdvanceFrame()
	mapView.Update()

	if selectCalled {
		t.Error("select callback was called on drag (should only be called on click)")
	}
}

func TestDoubleClickDistanceIsManhattan(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	var doubleClickCount int

	mapView.SetOnTileDoubleClick(func(t *tile.Tile) {
		doubleClickCount++
	})

	mapView.handleTileClick(10, 10, DoubleClickIntervalMs)
	mapView.handleTileClick(10+7, 10+7, DoubleClickIntervalMs)

	if doubleClickCount > 0 {
		t.Error("14 Manhattan distance triggered double-click (expected single click)")
	}

	doubleClickCount = 0

	mapView.handleTileClick(50, 50, DoubleClickIntervalMs)
	mapView.handleTileClick(50+5, 50+4, DoubleClickIntervalMs)

	if doubleClickCount != 1 {
		t.Errorf("9 Manhattan distance did not trigger double-click (got %d)", doubleClickCount)
	}
}

func TestRapidClickSequence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mapview-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	mapView, err := New(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("failed to create map view: %v", err)
	}

	var singleClickCount int
	var doubleClickCount int

	mapView.SetOnTileSelect(func(t *tile.Tile) {
		singleClickCount++
	})
	mapView.SetOnTileDoubleClick(func(t *tile.Tile) {
		doubleClickCount++
	})

	pos := 10

	for i := 0; i < 5; i++ {
		mapView.handleTileClick(pos, pos, DoubleClickIntervalMs)
	}

	if singleClickCount < 1 {
		t.Error("expected at least 1 single click for the first click")
	}

	if doubleClickCount < 1 {
		t.Error("expected at least 1 double-click for rapid clicks at same position")
	}

	totalCallbacks := singleClickCount + doubleClickCount
	if totalCallbacks != 5 {
		t.Errorf("expected 5 total callbacks (single + double), got %d", totalCallbacks)
	}
}
