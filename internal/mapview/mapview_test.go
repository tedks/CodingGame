package mapview

import (
	"os"
	"path/filepath"
	"testing"
)

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

	// Test panning
	initialX := mapView.PanX()
	initialY := mapView.PanY()

	mapView.Pan(10, 20)

	if mapView.PanX() != initialX+10 {
		t.Errorf("expected panX %f, got %f", initialX+10, mapView.PanX())
	}
	if mapView.PanY() != initialY+20 {
		t.Errorf("expected panY %f, got %f", initialY+20, mapView.PanY())
	}

	// Test negative pan
	mapView.Pan(-5, -10)

	if mapView.PanX() != initialX+5 {
		t.Errorf("expected panX %f, got %f", initialX+5, mapView.PanX())
	}
	if mapView.PanY() != initialY+10 {
		t.Errorf("expected panY %f, got %f", initialY+10, mapView.PanY())
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

	if mapView.ZoomLevel() != int(ZoomWorld) {
		t.Errorf("expected zoom to stay at %d, got %d", ZoomWorld, mapView.ZoomLevel())
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
		{ZoomWorld, 40},
		{ZoomRegion, 60},
		{ZoomCity, 80},
		{ZoomStreet, 100},
		{ZoomInterior, 120},
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
