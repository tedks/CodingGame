package game

import (
	"fmt"
	"image/color"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/advisor"
	"github.com/tedks/CodingGame/internal/claude"
	"github.com/tedks/CodingGame/internal/mapview"
	"github.com/tedks/CodingGame/internal/resources"
	"github.com/tedks/CodingGame/internal/ui"
)

// GameScene implements ui.Scene for the main gameplay view.
// It wraps the existing game functionality and integrates with the scene system.
type GameScene struct {
	projectPath string
	width       int
	height      int

	// Configuration from start screen
	config ui.GameConfig

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

// NewGameScene creates a new game scene for the given project.
func NewGameScene(projectPath string, width, height int) (*GameScene, error) {
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

	gs := &GameScene{
		projectPath: projectPath,
		width:       width,
		height:      height,
		mapView:     mapView,
		resources:   resourceTracker,
		interceptor: interceptor,
		advisorPool: advisorPool,
	}

	// Register event handlers for Claude tool interception
	interceptor.AddHandler(gs.handleClaudeEvent)

	// Start interceptor
	if err := interceptor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Claude interceptor: %w", err)
	}

	return gs, nil
}

// SetConfig sets the configuration from the start screen.
func (gs *GameScene) SetConfig(config ui.GameConfig) {
	gs.config = config
}

// Update implements ui.Scene.
func (gs *GameScene) Update() (ui.Scene, error) {
	// Handle keyboard input for navigation
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyH) {
		gs.mapView.Pan(-PanSpeed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyL) {
		gs.mapView.Pan(PanSpeed, 0)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyK) {
		gs.mapView.Pan(0, -PanSpeed)
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyJ) {
		gs.mapView.Pan(0, PanSpeed)
	}

	// Handle zoom with +/- or =/- keys
	if ebiten.IsKeyPressed(ebiten.KeyEqual) || ebiten.IsKeyPressed(ebiten.KeyKPAdd) {
		gs.mapView.ZoomIn()
	}
	if ebiten.IsKeyPressed(ebiten.KeyMinus) || ebiten.IsKeyPressed(ebiten.KeyKPSubtract) {
		gs.mapView.ZoomOut()
	}

	// Update map view
	gs.mapView.Update()

	// Update resource tracker
	gs.resources.Update()

	// No scene transition
	return nil, nil
}

// Draw implements ui.Scene.
func (gs *GameScene) Draw(screen *ebiten.Image) {
	// Clear screen with black
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// Draw resource bar at top
	gs.resources.Draw(screen, 0, 0, gs.width, ResourceBarHeight)

	// Draw map view below resource bar
	gs.mapView.Draw(screen, 0, ResourceBarHeight)

	// Draw debug info
	debugText := fmt.Sprintf(
		"FPS: %.1f\nZoom: %d\nPan: (%.0f, %.0f)\nControls: hjkl/arrows=pan, +/-=zoom",
		ebiten.ActualFPS(),
		gs.mapView.ZoomLevel(),
		gs.mapView.PanX(),
		gs.mapView.PanY(),
	)
	if gs.config.Harness != "" {
		debugText += fmt.Sprintf("\nHarness: %s\nModel: %s", gs.config.Harness, gs.config.Model)
	}
	ebitenutil.DebugPrint(screen, debugText)
}

// OnEnter implements ui.Scene.
func (gs *GameScene) OnEnter() {
	// Game scene entered
}

// OnExit implements ui.Scene.
func (gs *GameScene) OnExit() {
	// Game scene exiting
}

// Close cleans up game scene resources.
func (gs *GameScene) Close() error {
	if gs.interceptor != nil {
		return gs.interceptor.Stop()
	}
	return nil
}

// handleClaudeEvent processes events from the Claude interceptor.
func (gs *GameScene) handleClaudeEvent(event *claude.Event) {
	switch event.Type {
	case claude.EventFileRead:
		// Reveal fog for files that Claude reads
		if path, ok := event.Data["file_path"].(string); ok {
			absPath := filepath.Join(gs.projectPath, path)
			gs.mapView.RevealTile(absPath)
		}

	case claude.EventFileWrite, claude.EventFileEdit:
		// Highlight files that Claude writes/edits
		if path, ok := event.Data["file_path"].(string); ok {
			absPath := filepath.Join(gs.projectPath, path)
			// Reveal and highlight the tile
			gs.mapView.RevealTile(absPath)
			// Trigger advisors that watch this file (Phase 3)
			gs.triggerAdvisorsForFile(path)
		}

	case claude.EventBuildRun:
		// Update build status in resource tracker
		// TODO: Extract build results and update resources

	case claude.EventTestRun:
		// Update test results
		// TODO: Extract test results and update resources

	case claude.EventSubagentRun:
		// Handle advisor/subagent execution (Phase 3)
		gs.handleAdvisorEvent(event)
	}
}

// handleAdvisorEvent processes advisor-related events.
func (gs *GameScene) handleAdvisorEvent(event *claude.Event) {
	// Extract advisor ID from event data if present
	advisorID, _ := event.Data["advisor_id"].(string)
	if advisorID == "" {
		return
	}

	adv := gs.advisorPool.Get(advisorID)
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
				insight := gs.parseInsight(advisorID, insightMap)
				if insight != nil {
					adv.AddInsight(insight)
				}
			}
		}
	}
}

// parseInsight parses insight data from an event.
func (gs *GameScene) parseInsight(advisorID string, data map[string]interface{}) *advisor.Insight {
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

// triggerAdvisorsForFile triggers advisors that should run when a file changes.
func (gs *GameScene) triggerAdvisorsForFile(filePath string) {
	triggered := gs.advisorPool.TriggerOnFileChange(filePath)
	for _, adv := range triggered {
		// In a real implementation, this would spawn the advisor subagent
		// For now, we just mark that the advisor should analyze this file
		_ = adv // TODO: Implement actual advisor execution
	}
}
