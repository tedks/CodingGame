package game

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/advisor"
	"github.com/tedks/CodingGame/internal/capability"
	"github.com/tedks/CodingGame/internal/claude"
	"github.com/tedks/CodingGame/internal/harness"
	hclaude "github.com/tedks/CodingGame/internal/harness/claude"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/mapview"
	"github.com/tedks/CodingGame/internal/multiagent"
	"github.com/tedks/CodingGame/internal/production"
	"github.com/tedks/CodingGame/internal/resources"
	"github.com/tedks/CodingGame/internal/ui"
)

// PromptPanelHeight is the height of the bottom prompt panel in pixels.
const PromptPanelHeight = 60

// GameScene implements ui.Scene for the main gameplay view.
// It wraps the existing game functionality and integrates with the scene system.
type GameScene struct {
	projectPath string
	width       int
	height      int

	// Configuration from start screen
	config ui.GameConfig

	// Phase 0 components
	inputHandler *input.Handler
	promptPanel  *ui.PromptPanel

	// Phase 1 components
	mapView     *mapview.MapView
	resources   *resources.Tracker
	interceptor *claude.Interceptor // Legacy, kept for backwards compatibility

	// Harness system (Phase: Agent Support)
	registry    *harness.Registry
	mainHarness harness.Harness
	harnessCtx  context.Context
	harnessStop context.CancelFunc
	harnessWg   sync.WaitGroup

	// Phase 3 components
	advisorPool *advisor.Pool

	// Phase 5 components
	capabilityRegistry *capability.Registry
	capabilityRenderer *capability.Renderer
	capabilityWatcher  *capability.Watcher

	// Phase 6 components
	productionRegistry *production.Registry
	productionRenderer *production.Renderer
	productionWatcher  *production.Watcher

	// Phase 7 components
	multiagentOrchestrator *multiagent.Orchestrator
	multiagentRenderer     *multiagent.Renderer

	// Current view
	currentView input.ViewNumber

	// Callbacks
	onPromptSubmit func(text string) // Called when a prompt is submitted

	// Input state
	inputSource     input.InputSource
	lastMouseX      int
	lastMouseY      int
	wasMousePressed bool
}

// NewGameScene creates a new game scene for the given project.
func NewGameScene(projectPath string, width, height int) (*GameScene, error) {
	// Calculate map view height (accounting for resource bar and prompt panel)
	mapViewHeight := height - ResourceBarHeight - PromptPanelHeight

	// Initialize map view
	mapView, err := mapview.New(projectPath, width, mapViewHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to create map view: %w", err)
	}

	// Initialize resource tracker
	resourceTracker := resources.New()

	// Initialize Claude interceptor (legacy, for backwards compatibility)
	interceptor := claude.New()

	// Initialize harness registry and register available harnesses
	harnessRegistry := harness.NewRegistry()
	harnessRegistry.Register("claude-code", hclaude.NewHarness)

	// Initialize advisor pool with default advisors (Phase 3)
	advisorPool := advisor.NewPool()
	advisorPool.SetHarnessRegistry(harnessRegistry)
	advisorPool.SetWorkingDir(projectPath)
	if err := advisorPool.LoadFromConfig(advisor.DefaultConfigs()); err != nil {
		return nil, fmt.Errorf("failed to load advisor configs: %w", err)
	}

	// Initialize capability registry (Phase 5)
	capRegistry := capability.NewRegistry()
	capRegistry.RegisterDiscoverer(capability.NewBuiltinToolDiscoverer())
	capRegistry.RegisterDiscoverer(capability.NewMCPDiscoverer(projectPath))
	capRegistry.Refresh()

	// Initialize capability renderer
	capRenderer := capability.NewRenderer()

	// Initialize capability watcher
	capWatcher := capability.NewWatcher(capRegistry)

	// Initialize production registry (Phase 6)
	prodRegistry := production.NewRegistry()
	prodRegistry.RegisterDiscoverer(production.NewConfigDiscoverer(projectPath))
	prodRegistry.Refresh()

	// Initialize production renderer
	prodRenderer := production.NewRenderer()

	// Initialize production watcher
	prodWatcher := production.NewWatcher(prodRegistry)

	// Initialize multi-agent orchestrator (Phase 7)
	maOrchestrator := multiagent.NewOrchestrator()

	// Initialize multi-agent renderer
	maRenderer := multiagent.NewRenderer()

	// Initialize input handler
	inputHandler := input.NewHandler()

	// Initialize prompt panel
	promptPanel := ui.NewPromptPanel(width)
	promptPanel.SetScreenHeight(height)

	gs := &GameScene{
		projectPath:            projectPath,
		width:                  width,
		height:                 height,
		inputHandler:           inputHandler,
		promptPanel:            promptPanel,
		mapView:                mapView,
		resources:              resourceTracker,
		interceptor:            interceptor,
		registry:               harnessRegistry,
		advisorPool:            advisorPool,
		capabilityRegistry:     capRegistry,
		capabilityRenderer:     capRenderer,
		capabilityWatcher:      capWatcher,
		productionRegistry:     prodRegistry,
		productionRenderer:     prodRenderer,
		productionWatcher:      prodWatcher,
		multiagentOrchestrator: maOrchestrator,
		multiagentRenderer:     maRenderer,
		currentView:            input.ViewMap,
		inputSource:            input.DefaultSource,
	}

	// Wire up input handler callbacks
	gs.setupInputCallbacks()

	// Register event handlers for Claude tool interception
	interceptor.AddHandler(gs.handleClaudeEvent)

	// Start interceptor
	if err := interceptor.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Claude interceptor: %w", err)
	}

	return gs, nil
}

// SetInputSource sets the input source for testing.
// Pass nil to reset to the default Ebitengine source.
func (gs *GameScene) SetInputSource(source input.InputSource) {
	if source == nil {
		source = input.DefaultSource
	}
	gs.inputSource = source
	if gs.inputHandler != nil {
		gs.inputHandler.SetInputSource(source)
	}
	if gs.mapView != nil {
		gs.mapView.SetInputSource(source)
	}
}

// setupInputCallbacks wires up the input handler callbacks to the game scene.
func (gs *GameScene) setupInputCallbacks() {
	// Update prompt mode indicator when mode changes
	gs.inputHandler.OnModeChange(func(mode input.Mode) {
		gs.promptPanel.SetMode(mode.String())
	})

	// Update prompt focus state when focus changes
	gs.inputHandler.OnFocusChange(func(focus input.FocusArea) {
		if focus == input.FocusPrompt {
			gs.promptPanel.Focus()
		} else {
			gs.promptPanel.Unfocus()
		}
	})

	// Update prompt text when text buffer changes
	gs.inputHandler.OnTextChange(func(text string) {
		gs.promptPanel.SetText(text)
	})

	// Handle view changes
	gs.inputHandler.OnViewChange(func(view input.ViewNumber) {
		gs.currentView = view
	})

	// Handle actions
	gs.inputHandler.OnAction(func(action input.Action) {
		switch action {
		case input.ActionSubmitPrompt:
			gs.handlePromptSubmit()
		case input.ActionCancelPrompt:
			gs.handlePromptCancel()
		case input.ActionToggleMapView:
			gs.mapView.ToggleViewMode()
		}
	})

	// Set up prompt panel callbacks
	gs.promptPanel.OnSubmit = func(text string) {
		if gs.onPromptSubmit != nil {
			gs.onPromptSubmit(text)
		}
		// Clear text buffer and return to normal mode
		gs.inputHandler.ClearTextBuffer()
		gs.inputHandler.SetMode(input.ModeNormal)
		gs.inputHandler.SetFocus(input.FocusMap)
	}

	gs.promptPanel.OnCancel = func() {
		gs.inputHandler.ClearTextBuffer()
		gs.inputHandler.SetMode(input.ModeNormal)
		gs.inputHandler.SetFocus(input.FocusMap)
	}
}

// OnPromptSubmit sets the callback for when a prompt is submitted.
func (gs *GameScene) OnPromptSubmit(callback func(text string)) {
	gs.onPromptSubmit = callback
}

// handlePromptSubmit handles prompt submission.
func (gs *GameScene) handlePromptSubmit() {
	text := gs.inputHandler.TextBuffer()
	if text != "" {
		gs.promptPanel.Submit()
	}
}

// handlePromptCancel handles prompt cancellation.
func (gs *GameScene) handlePromptCancel() {
	gs.promptPanel.Cancel()
}

// SetConfig sets the configuration from the start screen.
// If a harness is specified, it will be started.
func (gs *GameScene) SetConfig(config ui.GameConfig) {
	gs.config = config

	// Start the selected harness if one is specified
	if config.Harness != "" && gs.registry.IsRegistered(config.Harness) {
		gs.startHarness(config)
	}
}

// SetHarnessRegistry sets the harness registry, replacing the default one.
// This should be called before SetConfig to ensure the harness is available.
func (gs *GameScene) SetHarnessRegistry(registry *harness.Registry) {
	gs.registry = registry
	gs.advisorPool.SetHarnessRegistry(registry)
}

// startHarness creates and starts the main harness based on config.
func (gs *GameScene) startHarness(config ui.GameConfig) {
	// Create harness instance
	h, err := gs.registry.Create(config.Harness)
	if err != nil {
		// Log error but don't fail - game can still work without harness
		fmt.Fprintf(os.Stderr, "Warning: Failed to create harness %s: %v\n", config.Harness, err)
		return
	}

	// Create context for harness lifecycle
	gs.harnessCtx, gs.harnessStop = context.WithCancel(context.Background())

	// Build harness configuration
	harnessConfig := harness.NewConfig(gs.projectPath).
		WithModel(config.Model)

	// Start the harness
	if err := h.Start(gs.harnessCtx, harnessConfig); err != nil {
		// Log error but don't fail
		fmt.Fprintf(os.Stderr, "Warning: Failed to start harness %s: %v\n", config.Harness, err)
		gs.harnessStop()
		gs.harnessCtx = nil
		gs.harnessStop = nil
		return
	}

	gs.mainHarness = h

	// Wire up prompt submission to send to harness
	gs.onPromptSubmit = func(text string) {
		if err := gs.mainHarness.SendPrompt(text); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to send prompt: %v\n", err)
		}
	}

	// Configure advisor pool to use the same harness as the main agent by default
	if gs.advisorPool != nil {
		gs.advisorPool.SetMainHarness(config.Harness)
	}

	// Start event processing goroutine
	gs.harnessWg.Add(1)
	go gs.processHarnessEvents()
}

// processHarnessEvents reads events from the main harness and processes them.
func (gs *GameScene) processHarnessEvents() {
	defer gs.harnessWg.Done()

	if gs.mainHarness == nil {
		return
	}

	for event := range gs.mainHarness.Events() {
		gs.handleHarnessEvent(&event)
	}
}

// handleHarnessEvent processes events from the harness system.
func (gs *GameScene) handleHarnessEvent(event *harness.Event) {
	switch event.Type {
	case harness.EventFileRead:
		// Reveal fog for files that the agent reads
		path := event.FilePath()
		if path != "" {
			absPath := gs.resolveFilePath(path)
			gs.mapView.RevealTile(absPath)
		}

	case harness.EventFileWrite, harness.EventFileEdit:
		// Highlight files that the agent writes/edits
		path := event.FilePath()
		if path != "" {
			absPath := gs.resolveFilePath(path)
			gs.mapView.RevealTile(absPath)
			// Trigger advisors that watch this file
			gs.triggerAdvisorsForFile(path)
		}

	case harness.EventBuildRun:
		// Update build status in resource tracker
		// TODO: Extract build results and update resources

	case harness.EventTestRun:
		// Update test results
		// TODO: Extract test results and update resources

	case harness.EventSubagentRun:
		// Handle advisor/subagent execution
		gs.handleHarnessAdvisorEvent(event)

	case harness.EventText:
		// Display response text in prompt panel
		if event.Text != "" {
			gs.promptPanel.SetResponseText(event.Text)
		}

	case harness.EventTurnComplete:
		// Reset prompt panel to idle when turn completes
		gs.promptPanel.SetState(ui.PromptStateIdle)
	}
}

// handleHarnessAdvisorEvent processes advisor-related events from the harness.
func (gs *GameScene) handleHarnessAdvisorEvent(event *harness.Event) {
	// Extract advisor ID from event
	advisorID, _ := event.ToolInput["advisor_id"].(string)
	if advisorID == "" {
		advisorID, _ = event.Raw["advisor_id"].(string)
	}
	if advisorID == "" {
		return
	}

	adv := gs.advisorPool.Get(advisorID)
	if adv == nil {
		return
	}

	// Check status in raw data
	if status, ok := event.Raw["status"].(string); ok {
		switch status {
		case "started":
			adv.StartAnalysis()
		case "completed":
			duration, _ := event.Raw["duration_ms"].(float64)
			tokensIn, _ := event.Raw["tokens_in"].(float64)
			tokensOut, _ := event.Raw["tokens_out"].(float64)
			adv.CompleteAnalysis(
				durationFromMs(duration),
				int64(tokensIn),
				int64(tokensOut),
				nil,
			)
		case "error":
			errMsg, _ := event.Raw["error"].(string)
			adv.CompleteAnalysis(0, 0, 0, fmt.Errorf("%s", errMsg))
		}
	}

	// Check for insights in the event
	if insightsData, ok := event.Raw["insights"].([]interface{}); ok {
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

// resolveFilePath resolves a file path relative to the project path.
func (gs *GameScene) resolveFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(gs.projectPath, path)
}

// handlePromptPanelDrag handles mouse drag for resizing the prompt panel.
// Returns true if the prompt panel consumed the input.
func (gs *GameScene) handlePromptPanelDrag() bool {
	inputSource := gs.inputSource
	x, y := inputSource.CursorPosition()
	consumed := false

	// Handle mouse wheel scrolling
	_, wheelY := inputSource.Wheel()
	if wheelY != 0 && gs.promptPanel.HandleScroll(x, y, 0, wheelY) {
		consumed = true
	}

	// Check for mouse button state
	if inputSource.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if gs.promptPanel.IsDragging() {
			// Continue dragging
			gs.promptPanel.UpdateDrag(y)
			consumed = true
		} else if gs.promptPanel.IsOnDragHandle(x, y) {
			// Start dragging
			gs.promptPanel.StartDrag(y)
			consumed = true
		} else if gs.promptPanel.ContainsPoint(x, y) {
			// Mouse is over panel but not dragging
			consumed = true
		}
	} else {
		// End dragging if mouse released
		if gs.promptPanel.IsDragging() {
			gs.promptPanel.EndDrag()
		}

		// Handle click on mouse release (detect click by checking if we just released)
		if gs.wasMousePressed && !inputSource.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			if gs.promptPanel.HandleClick(x, y) {
				consumed = true
			}
		}
	}

	// Track mouse pressed state for click detection
	gs.wasMousePressed = inputSource.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Update last mouse position
	gs.lastMouseX = x
	gs.lastMouseY = y

	return consumed
}

// PromptPanelHeight returns the current prompt panel height.
func (gs *GameScene) PromptPanelHeight() int {
	if gs.promptPanel == nil {
		return 0
	}
	return gs.promptPanel.Height()
}

// Update implements ui.Scene.
func (gs *GameScene) Update() (ui.Scene, error) {
	// Handle prompt panel mouse interactions first
	// Returns true if the prompt panel consumed the input
	promptConsumedInput := gs.handlePromptPanelDrag()

	// Disable map mouse input when prompt panel is consuming input
	gs.mapView.SetMouseInputEnabled(!promptConsumedInput)

	// Update input handler (processes all keybindings)
	gs.inputHandler.Update()

	// Handle navigation based on current mode and focus
	// Navigation only works when focused on map and in Normal mode
	// Skip if prompt panel is handling input
	if gs.inputHandler.Focus() == input.FocusMap && !promptConsumedInput {
		// Use the input handler's action checks for continuous movement
		if gs.inputHandler.IsActionHeld(input.ActionMoveLeft) {
			gs.mapView.Pan(-PanSpeed, 0)
		}
		if gs.inputHandler.IsActionHeld(input.ActionMoveRight) {
			gs.mapView.Pan(PanSpeed, 0)
		}
		if gs.inputHandler.IsActionHeld(input.ActionMoveUp) {
			gs.mapView.Pan(0, -PanSpeed)
		}
		if gs.inputHandler.IsActionHeld(input.ActionMoveDown) {
			gs.mapView.Pan(0, PanSpeed)
		}

		// Handle zoom (use IsAction for single press, not held)
		if gs.inputHandler.IsAction(input.ActionZoomIn) {
			gs.mapView.ZoomIn()
		}
		if gs.inputHandler.IsAction(input.ActionZoomOut) {
			gs.mapView.ZoomOut()
		}
	}

	// Update prompt panel
	gs.promptPanel.Update()

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

	// Content area dimensions
	contentY := ResourceBarHeight
	contentHeight := gs.height - ResourceBarHeight - PromptPanelHeight

	// Draw current view
	switch gs.currentView {
	case input.ViewMap:
		gs.mapView.Draw(screen, 0, contentY)
	case input.ViewTech:
		gs.capabilityRenderer.Draw(
			screen,
			gs.capabilityRegistry.GetAll(),
			0, contentY,
			gs.width, contentHeight,
		)
	case input.ViewProduction:
		gs.productionRenderer.Draw(
			screen,
			gs.productionRegistry.GetAll(),
			0, contentY,
			gs.width, contentHeight,
		)
	case input.ViewMultiAgent:
		gs.multiagentRenderer.Draw(
			screen,
			gs.multiagentOrchestrator.GetAll(),
			0, contentY,
			gs.width, contentHeight,
		)
	default:
		// Other views not yet implemented - show placeholder
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("View %d not yet implemented", gs.currentView), 20, contentY+20)
	}

	// Draw prompt panel at bottom
	gs.promptPanel.Draw(screen)

	// Draw debug info
	mode := gs.inputHandler.Mode().String()
	focus := gs.inputHandler.Focus().String()
	viewName := gs.viewName()
	debugText := fmt.Sprintf(
		"FPS: %.1f | Mode: %s | Focus: %s | View: %s\nZoom: %d | Pan: (%.0f, %.0f)\nEnter=prompt, hjkl=pan, +/-=zoom, 1-7=views",
		ebiten.ActualFPS(),
		mode,
		focus,
		viewName,
		gs.mapView.ZoomLevel(),
		gs.mapView.PanX(),
		gs.mapView.PanY(),
	)
	if gs.config.Harness != "" {
		debugText += fmt.Sprintf("\nHarness: %s | Model: %s", gs.config.Harness, gs.config.Model)
	}
	ebitenutil.DebugPrintAt(screen, debugText, 0, ResourceBarHeight)
}

// viewName returns a human-readable name for the current view.
func (gs *GameScene) viewName() string {
	switch gs.currentView {
	case input.ViewMap:
		return "Map (" + gs.mapView.ViewMode().String() + ")"
	case input.ViewBuilding:
		return "Buildings"
	case input.ViewUnit:
		return "Units"
	case input.ViewTech:
		return "Tech Tree"
	case input.ViewMission:
		return "Missions"
	case input.ViewProduction:
		return "Production"
	case input.ViewMultiAgent:
		return "Multi-Agent"
	default:
		return "Unknown"
	}
}

// OnEnter implements ui.Scene.
func (gs *GameScene) OnEnter() {
	// Start capability watcher for dynamic updates
	if gs.capabilityWatcher != nil {
		gs.capabilityWatcher.Start()
	}
	// Start production watcher for dynamic updates
	if gs.productionWatcher != nil {
		gs.productionWatcher.Start()
	}
}

// OnExit implements ui.Scene.
func (gs *GameScene) OnExit() {
	// Game scene exiting
}

// Close cleans up game scene resources.
func (gs *GameScene) Close() error {
	if gs.capabilityWatcher != nil {
		gs.capabilityWatcher.Stop()
	}
	if gs.productionWatcher != nil {
		gs.productionWatcher.Stop()
	}

	// Stop the main harness if running
	if gs.mainHarness != nil {
		if err := gs.mainHarness.Stop(); err != nil {
			// Log but don't fail on harness stop error
			_ = err
		}
	}
	if gs.harnessStop != nil {
		gs.harnessStop()
	}

	// Wait for event processing goroutine to complete
	gs.harnessWg.Wait()

	// Stop legacy interceptor
	if gs.interceptor != nil {
		return gs.interceptor.Stop()
	}
	return nil
}

// handleClaudeEvent processes events from the Claude interceptor (legacy).
func (gs *GameScene) handleClaudeEvent(event *claude.Event) {
	switch event.Type {
	case claude.EventFileRead:
		// Reveal fog for files that Claude reads
		if path, ok := event.Data["file_path"].(string); ok {
			absPath := gs.resolveFilePath(path)
			gs.mapView.RevealTile(absPath)
		}

	case claude.EventFileWrite, claude.EventFileEdit:
		// Highlight files that Claude writes/edits
		if path, ok := event.Data["file_path"].(string); ok {
			absPath := gs.resolveFilePath(path)
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

// Harness returns the main harness instance, if one is running.
func (gs *GameScene) Harness() harness.Harness {
	return gs.mainHarness
}

// HarnessRegistry returns the harness registry.
func (gs *GameScene) HarnessRegistry() *harness.Registry {
	return gs.registry
}

// SendPrompt sends a prompt to the main harness if one is running.
// Returns an error if no harness is running.
func (gs *GameScene) SendPrompt(prompt string) error {
	if gs.mainHarness == nil || !gs.mainHarness.IsRunning() {
		return fmt.Errorf("no harness running")
	}
	return gs.mainHarness.SendPrompt(prompt)
}

// SimulateHarnessEvent injects a simulated harness event for testing.
func (gs *GameScene) SimulateHarnessEvent(event harness.Event) {
	gs.handleHarnessEvent(&event)
}

// SimulateFileRead simulates a file read event.
// This works with both the legacy interceptor and the new harness system.
func (gs *GameScene) SimulateFileRead(path string) {
	// Use legacy interceptor for backwards compatibility
	if gs.interceptor != nil {
		gs.interceptor.SimulateFileRead(path)
	}
}

// SimulateFileWrite simulates a file write event.
func (gs *GameScene) SimulateFileWrite(path string) {
	if gs.interceptor != nil {
		gs.interceptor.SimulateFileWrite(path)
	}
}

// SimulateFileEdit simulates a file edit event.
func (gs *GameScene) SimulateFileEdit(path string) {
	if gs.interceptor != nil {
		gs.interceptor.SimulateFileEdit(path)
	}
}
