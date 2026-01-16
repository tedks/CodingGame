// Package game implements the main game loop and coordinates the CodingGame
// application components. It integrates the map visualization, resource tracking,
// and Claude Code tool interception to provide a real-time RTS-style view of
// software development activities.
//
// The game runs at 60 FPS using the Ebitengine framework and manages keyboard
// input, rendering, and event processing from the Claude subprocess.
package game

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/claude"
	"github.com/tedks/CodingGame/internal/mapview"
	"github.com/tedks/CodingGame/internal/resources"
)

const (
	// ResourceBarHeight is the height of the top resource bar in pixels
	ResourceBarHeight = 40

	// PanSpeed is the number of pixels to pan per frame when arrow keys are held
	PanSpeed = 5
)

// Game implements ebiten.Game interface and manages the game state
type Game struct {
	projectPath string
	width       int
	height      int

	// Phase 1 components
	mapView     *mapview.MapView
	resources   *resources.Tracker
	interceptor *claude.Interceptor

	// Input state
	lastMouseX int
	lastMouseY int
}

// New creates a new game instance
func New(projectPath string, width, height int) (*Game, error) {
	// Initialize map view
	mapView, err := mapview.New(projectPath, width, height-ResourceBarHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to create map view: %w", err)
	}

	// Initialize resource tracker
	resourceTracker := resources.New()

	// Initialize Claude interceptor
	interceptor := claude.New()

	g := &Game{
		projectPath: projectPath,
		width:       width,
		height:      height,
		mapView:     mapView,
		resources:   resourceTracker,
		interceptor: interceptor,
	}

	// Register event handlers for Claude tool interception
	interceptor.AddHandler(g.handleClaudeEvent)

	// Start interceptor
	if err := interceptor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Claude interceptor: %w", err)
	}

	return g, nil
}

// Update updates the game state (called 60 times per second)
func (g *Game) Update() error {
	// Handle keyboard input for navigation
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyH) {
		g.mapView.Pan(-PanSpeed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyL) {
		g.mapView.Pan(PanSpeed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyK) {
		g.mapView.Pan(0, -PanSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyJ) {
		g.mapView.Pan(0, PanSpeed)
	}

	// Handle zoom with +/- or =/- keys
	if ebiten.IsKeyPressed(ebiten.KeyEqual) || ebiten.IsKeyPressed(ebiten.KeyKPAdd) {
		g.mapView.ZoomIn()
	}
	if ebiten.IsKeyPressed(ebiten.KeyMinus) || ebiten.IsKeyPressed(ebiten.KeyKPSubtract) {
		g.mapView.ZoomOut()
	}

	// Update map view
	g.mapView.Update()

	// Update resource tracker
	g.resources.Update()

	return nil
}

// Draw renders the game (called 60 times per second)
func (g *Game) Draw(screen *ebiten.Image) {
	// Clear screen
	screen.Fill(ebiten.ColorScale{}.Scale(0, 0, 0, 1).Apply())

	// Draw resource bar at top
	g.resources.Draw(screen, 0, 0, g.width, ResourceBarHeight)

	// Draw map view below resource bar
	g.mapView.Draw(screen, 0, ResourceBarHeight)

	// Draw debug info
	ebitenutil.DebugPrint(screen, fmt.Sprintf(
		"FPS: %.1f\nZoom: %d\nPan: (%.0f, %.0f)\nControls: hjkl/arrows=pan, +/-=zoom",
		ebiten.ActualFPS(),
		g.mapView.ZoomLevel(),
		g.mapView.PanX(),
		g.mapView.PanY(),
	))
}

// Layout returns the game's screen dimensions
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.width, g.height
}

// handleClaudeEvent processes events from the Claude interceptor
func (g *Game) handleClaudeEvent(event *claude.Event) {
	switch event.Type {
	case claude.EventFileRead:
		// Reveal fog for files that Claude reads
		if path, ok := event.Data["file_path"].(string); ok {
			absPath := filepath.Join(g.projectPath, path)
			g.mapView.RevealTile(absPath)
		}

	case claude.EventFileWrite, claude.EventFileEdit:
		// Highlight files that Claude writes/edits
		if path, ok := event.Data["file_path"].(string); ok {
			absPath := filepath.Join(g.projectPath, path)
			// Reveal and highlight the tile
			g.mapView.RevealTile(absPath)
			// TODO: Add highlight functionality to mapview
		}

	case claude.EventBuildRun:
		// Update build status in resource tracker
		// TODO: Extract build results and update resources

	case claude.EventTestRun:
		// Update test results
		// TODO: Extract test results and update resources

	case claude.EventSubagentRun:
		// Show advisor panel activity
		// TODO: Implement advisor panel
	}
}

// Close cleans up game resources
func (g *Game) Close() error {
	if g.interceptor != nil {
		return g.interceptor.Stop()
	}
	return nil
}
