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
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/advisor"
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

	// Phase 3 components
	advisorPool *advisor.Pool

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

	// Initialize advisor pool with default advisors (Phase 3)
	advisorPool := advisor.NewPool()
	if err := advisorPool.LoadFromConfig(advisor.DefaultConfigs()); err != nil {
		return nil, fmt.Errorf("failed to load advisor configs: %w", err)
	}

	g := &Game{
		projectPath: projectPath,
		width:       width,
		height:      height,
		mapView:     mapView,
		resources:   resourceTracker,
		interceptor: interceptor,
		advisorPool: advisorPool,
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
			// Trigger advisors that watch this file (Phase 3)
			g.triggerAdvisorsForFile(path)
		}

	case claude.EventBuildRun:
		// Update build status in resource tracker
		// TODO: Extract build results and update resources

	case claude.EventTestRun:
		// Update test results
		// TODO: Extract test results and update resources

	case claude.EventSubagentRun:
		// Handle advisor/subagent execution (Phase 3)
		g.handleAdvisorEvent(event)
	}
}

// handleAdvisorEvent processes advisor-related events
func (g *Game) handleAdvisorEvent(event *claude.Event) {
	// Extract advisor ID from event data if present
	advisorID, _ := event.Data["advisor_id"].(string)
	if advisorID == "" {
		return
	}

	adv := g.advisorPool.Get(advisorID)
	if adv == nil {
		return
	}

	// Check if this is a start or completion event
	if status, ok := event.Data["status"].(string); ok {
		switch status {
		case "started":
			adv.StartAnalysis()
		case "completed":
			duration, _ := event.Data["duration_ms"].(float64)
			tokensIn, _ := event.Data["tokens_in"].(float64)
			tokensOut, _ := event.Data["tokens_out"].(float64)
			adv.CompleteAnalysis(
				durationFromMs(duration),
				int64(tokensIn),
				int64(tokensOut),
				nil,
			)
		case "error":
			errMsg, _ := event.Data["error"].(string)
			adv.CompleteAnalysis(0, 0, 0, fmt.Errorf("%s", errMsg))
		}
	}

	// Check for insights in the event
	if insightsData, ok := event.Data["insights"].([]interface{}); ok {
		for _, insightData := range insightsData {
			if insightMap, ok := insightData.(map[string]interface{}); ok {
				insight := g.parseInsight(advisorID, insightMap)
				if insight != nil {
					adv.AddInsight(insight)
				}
			}
		}
	}
}

// parseInsight parses insight data from an event
func (g *Game) parseInsight(advisorID string, data map[string]interface{}) *advisor.Insight {
	title, _ := data["title"].(string)
	description, _ := data["description"].(string)
	if title == "" {
		return nil
	}

	// Map severity
	severityStr, _ := data["severity"].(string)
	severity := advisor.SeverityInfo
	switch severityStr {
	case "warning":
		severity = advisor.SeverityWarning
	case "critical":
		severity = advisor.SeverityCritical
	}

	// Map category
	categoryStr, _ := data["category"].(string)
	category := advisor.CategoryGeneral
	switch categoryStr {
	case "security":
		category = advisor.CategorySecurity
	case "performance":
		category = advisor.CategoryPerformance
	case "refactoring":
		category = advisor.CategoryRefactoring
	case "testing":
		category = advisor.CategoryTesting
	}

	insight := advisor.NewInsight(advisorID, title, description, severity, category)

	// Add location if present
	if filePath, ok := data["file_path"].(string); ok {
		line, _ := data["line"].(float64)
		column, _ := data["column"].(float64)
		insight.WithLocation(filePath, int(line), int(column))
	}

	// Add suggestion if present
	if suggestion, ok := data["suggestion"].(string); ok {
		codeBefore, _ := data["code_before"].(string)
		codeAfter, _ := data["code_after"].(string)
		insight.WithSuggestion(suggestion, codeBefore, codeAfter)
	}

	return insight
}

// triggerAdvisorsForFile triggers advisors that should run when a file changes
func (g *Game) triggerAdvisorsForFile(filePath string) {
	triggered := g.advisorPool.TriggerOnFileChange(filePath)
	for _, adv := range triggered {
		// In a real implementation, this would spawn the advisor subagent
		// For now, we just mark that the advisor should analyze this file
		_ = adv // TODO: Implement actual advisor execution
	}
}

// durationFromMs converts milliseconds to time.Duration
func durationFromMs(ms float64) time.Duration {
	return time.Duration(ms * float64(time.Millisecond))
}

// Close cleans up game resources
func (g *Game) Close() error {
	if g.interceptor != nil {
		return g.interceptor.Stop()
	}
	return nil
}
