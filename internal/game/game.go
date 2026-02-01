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
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/advisor"
	"github.com/tedks/CodingGame/internal/claude"
	"github.com/tedks/CodingGame/internal/dependency"
	"github.com/tedks/CodingGame/internal/input"
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

	// Phase 3 components
	advisorPool *advisor.Pool

	// Input state
	lastMouseX  int
	lastMouseY  int
	inputSource input.InputSource
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

	// Initialize advisor pool with default advisors (Phase 3)
	advisorPool := advisor.NewPool()
	if err := advisorPool.LoadFromConfig(advisor.DefaultConfigs()); err != nil {
		return nil, fmt.Errorf("failed to load advisor configs: %w", err)
	}

	// Extract Go dependencies for dataflow visualization (Phase 4)
	extractor, err := dependency.NewExtractor(projectPath)
	if err == nil {
		// Use symbol-level extraction for more accurate coupling strength
		graph, err := extractor.ExtractGoWithSymbols()
		if err == nil && graph != nil {
			mapView.SetConnectionGraph(graph)
		}
	}

	g := &Game{
		projectPath: projectPath,
		width:       width,
		height:      height,
		mapView:     mapView,
		resources:   resourceTracker,
		interceptor: interceptor,
		advisorPool: advisorPool,
		inputSource: input.DefaultSource,
	}

	// Register event handlers for Claude tool interception
	interceptor.AddHandler(g.handleClaudeEvent)

	// Start interceptor
	if err := interceptor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Claude interceptor: %w", err)
	}

	return g, nil
}

// SetInputSource sets the input source for testing.
// Pass nil to reset to the default Ebitengine source.
func (g *Game) SetInputSource(source input.InputSource) {
	if source == nil {
		g.inputSource = input.DefaultSource
	} else {
		g.inputSource = source
	}
}

// Update updates the game state (called 60 times per second)
func (g *Game) Update() error {
	// Handle keyboard input for navigation
	if g.inputSource.IsKeyPressed(ebiten.KeyArrowLeft) || g.inputSource.IsKeyPressed(ebiten.KeyH) {
		g.mapView.Pan(-PanSpeed, 0)
	}
	if g.inputSource.IsKeyPressed(ebiten.KeyArrowRight) || g.inputSource.IsKeyPressed(ebiten.KeyL) {
		g.mapView.Pan(PanSpeed, 0)
	}
	if g.inputSource.IsKeyPressed(ebiten.KeyArrowUp) || g.inputSource.IsKeyPressed(ebiten.KeyK) {
		g.mapView.Pan(0, -PanSpeed)
	}
	if g.inputSource.IsKeyPressed(ebiten.KeyArrowDown) || g.inputSource.IsKeyPressed(ebiten.KeyJ) {
		g.mapView.Pan(0, PanSpeed)
	}

	// Handle zoom with +/- or =/- keys
	if g.inputSource.IsKeyPressed(ebiten.KeyEqual) || g.inputSource.IsKeyPressed(ebiten.KeyKPAdd) {
		g.mapView.ZoomIn()
	}
	if g.inputSource.IsKeyPressed(ebiten.KeyMinus) || g.inputSource.IsKeyPressed(ebiten.KeyKPSubtract) {
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
	// Clear screen with black
	screen.Fill(color.RGBA{0, 0, 0, 255})

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
	processClaudeEvent(event, g.projectPath, g.mapView, g.advisorPool)
}

// Close cleans up game resources
func (g *Game) Close() error {
	if g.interceptor != nil {
		return g.interceptor.Stop()
	}
	return nil
}
