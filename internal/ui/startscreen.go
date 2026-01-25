package ui

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tedks/CodingGame/internal/harness"
	"github.com/tedks/CodingGame/internal/input"
)

// GameConfig holds the configuration selected during the start screen flow.
type GameConfig struct {
	Harness     string // e.g., "claude-code", "codex", "gemini"
	Model       string // e.g., "opus", "sonnet", "haiku"
	ProjectPath string // Path to the project directory
}

// StartScreenState represents the current step in the start screen flow.
type StartScreenState int

const (
	StateMainMenu StartScreenState = iota
	StateHarnessSelect
	StateModelSelect
	StateProjectSelect
)

// StartScreen implements the game's start/title screen with harness,
// model, and project selection flow.
type StartScreen struct {
	width, height int
	state         StartScreenState
	config        GameConfig

	// Menus for each step
	mainMenu    *Menu
	harnessMenu *Menu
	modelMenu   *Menu
	projectMenu *Menu

	// Text input for custom project path
	projectInput     string
	projectInputMode bool

	// Recent projects (would be loaded from config)
	recentProjects []string

	// Callback when configuration is complete
	onComplete func(config GameConfig)

	// For title animation
	frameCount int

	// Input source (for testing injection)
	inputSource input.InputSource

	// Harness registry for dynamic harness/model info
	registry *harness.Registry
}

// NewStartScreen creates a new start screen.
func NewStartScreen(width, height int, onComplete func(config GameConfig)) *StartScreen {
	ss := &StartScreen{
		width:          width,
		height:         height,
		state:          StateMainMenu,
		onComplete:     onComplete,
		inputSource:    input.DefaultSource,
		registry:       harness.NewRegistry(),
		recentProjects: []string{
			// These would be loaded from a config file
		},
	}

	ss.initMenus()
	return ss
}

// SetInputSource sets the input source for testing.
// Pass nil to reset to the default Ebitengine source.
func (ss *StartScreen) SetInputSource(source input.InputSource) {
	if source == nil {
		ss.inputSource = input.DefaultSource
	} else {
		ss.inputSource = source
	}
	// Also set on all menus
	ss.mainMenu.SetInputSource(source)
	ss.harnessMenu.SetInputSource(source)
	ss.modelMenu.SetInputSource(source)
	ss.projectMenu.SetInputSource(source)
}

func (ss *StartScreen) initMenus() {
	// Main menu
	ss.mainMenu = NewMenu("", []*MenuItem{
		NewMenuItem("NEW GAME"),
		NewMenuItem("CONTINUE"),
	})
	ss.mainMenu.CancelAllowed = false
	ss.mainMenu.Width = 200

	// Harness selection - build from registry
	ss.harnessMenu = ss.buildHarnessMenu()
	ss.harnessMenu.Width = 280

	// Model selection - initially for first harness, updates when harness selected
	ss.modelMenu = ss.buildModelMenu("claude-code")
	ss.modelMenu.Width = 280

	// Project selection
	projectItems := []*MenuItem{
		NewMenuItem("Enter project path..."),
	}
	// Add recent projects
	for _, path := range ss.recentProjects {
		// Shorten path for display
		label := path
		if len(label) > 40 {
			label = "..." + label[len(label)-37:]
		}
		projectItems = append(projectItems, NewMenuItemWithValue(label, path))
	}
	ss.projectMenu = NewMenu("SELECT PROJECT", projectItems)
	ss.projectMenu.Width = 400
}

// buildHarnessMenu creates the harness selection menu from the registry.
func (ss *StartScreen) buildHarnessMenu() *Menu {
	var items []*MenuItem

	// Get all harness definitions from registry
	for _, name := range ss.registry.Defined() {
		info := ss.registry.Info(name)
		if info == nil {
			continue
		}

		// Build display label
		label := info.DisplayName
		if !info.Installed {
			label += " (not installed)"
		} else if !info.Registered {
			label += " (not supported)"
		}

		item := NewMenuItemWithValue(label, info.Name)
		// Only enable if both installed AND has a registered factory
		item.Enabled = info.Installed && info.Registered

		items = append(items, item)
	}

	// If no harnesses defined, add a placeholder
	if len(items) == 0 {
		items = append(items, NewMenuItem("No harnesses available"))
		items[0].Enabled = false
	}

	return NewMenu("SELECT HARNESS", items)
}

// buildModelMenu creates the model selection menu for a given harness.
func (ss *StartScreen) buildModelMenu(harnessName string) *Menu {
	var items []*MenuItem

	models := ss.registry.Models(harnessName)
	defaultModel := ss.registry.DefaultModel(harnessName)

	for _, model := range models {
		label := model.Name
		if model.ID == defaultModel {
			label += " (Recommended)"
		}

		items = append(items, NewMenuItemWithValue(label, model.ID))
	}

	// If no models defined, add a placeholder
	if len(items) == 0 {
		items = append(items, NewMenuItem("No models available"))
		items[0].Enabled = false
	}

	return NewMenu("SELECT MODEL", items)
}

// updateModelMenuForHarness rebuilds the model menu for the selected harness.
func (ss *StartScreen) updateModelMenuForHarness(harnessName string) {
	ss.modelMenu = ss.buildModelMenu(harnessName)
	ss.modelMenu.Width = 280
	ss.modelMenu.SetInputSource(ss.inputSource)
}

// Update handles input and updates state.
func (ss *StartScreen) Update() (Scene, error) {
	ss.frameCount++

	// Handle project path text input mode
	if ss.projectInputMode {
		return ss.handleProjectInput()
	}

	switch ss.state {
	case StateMainMenu:
		return ss.updateMainMenu()
	case StateHarnessSelect:
		return ss.updateHarnessSelect()
	case StateModelSelect:
		return ss.updateModelSelect()
	case StateProjectSelect:
		return ss.updateProjectSelect()
	}

	return nil, nil
}

func (ss *StartScreen) updateMainMenu() (Scene, error) {
	ss.mainMenu.Center(ss.width, ss.height)

	selected, _, err := ss.mainMenu.Update()
	if err != nil {
		return nil, err
	}

	if selected == "NEW GAME" {
		ss.state = StateHarnessSelect
	} else if selected == "CONTINUE" {
		// TODO: Load last session
		ss.state = StateHarnessSelect
	}

	return nil, nil
}

func (ss *StartScreen) updateHarnessSelect() (Scene, error) {
	ss.harnessMenu.Center(ss.width, ss.height)

	selected, cancelled, err := ss.harnessMenu.Update()
	if err != nil {
		return nil, err
	}

	if cancelled {
		ss.state = StateMainMenu
		return nil, nil
	}

	if selected != "" {
		ss.config.Harness = selected
		// Update model menu for the selected harness
		ss.updateModelMenuForHarness(selected)
		ss.state = StateModelSelect
	}

	return nil, nil
}

func (ss *StartScreen) updateModelSelect() (Scene, error) {
	ss.modelMenu.Center(ss.width, ss.height)

	selected, cancelled, err := ss.modelMenu.Update()
	if err != nil {
		return nil, err
	}

	if cancelled {
		ss.state = StateHarnessSelect
		return nil, nil
	}

	if selected != "" {
		ss.config.Model = selected
		ss.state = StateProjectSelect
	}

	return nil, nil
}

func (ss *StartScreen) updateProjectSelect() (Scene, error) {
	ss.projectMenu.Center(ss.width, ss.height)

	selected, cancelled, err := ss.projectMenu.Update()
	if err != nil {
		return nil, err
	}

	if cancelled {
		ss.state = StateModelSelect
		return nil, nil
	}

	if selected != "" {
		if selected == "Enter project path..." {
			ss.projectInputMode = true
			ss.projectInput = ""
		} else {
			ss.config.ProjectPath = selected
			ss.completeConfiguration()
		}
	}

	return nil, nil
}

func (ss *StartScreen) handleProjectInput() (Scene, error) {
	// Handle text input character by character
	chars := ss.inputSource.AppendInputChars(nil)
	ss.projectInput += string(chars)

	// Handle backspace
	if ss.inputSource.IsKeyJustPressed(ebiten.KeyBackspace) && len(ss.projectInput) > 0 {
		ss.projectInput = ss.projectInput[:len(ss.projectInput)-1]
	}

	// Handle Enter to confirm
	if ss.inputSource.IsKeyJustPressed(ebiten.KeyEnter) {
		path := strings.TrimSpace(ss.projectInput)
		if path != "" {
			ss.config.ProjectPath = path
			ss.projectInputMode = false
			ss.completeConfiguration()
		}
	}

	// Handle Escape to cancel
	if ss.inputSource.IsKeyJustPressed(ebiten.KeyEscape) {
		ss.projectInputMode = false
	}

	return nil, nil
}

func (ss *StartScreen) completeConfiguration() {
	if ss.onComplete != nil {
		ss.onComplete(ss.config)
	}
}

// Draw renders the start screen.
func (ss *StartScreen) Draw(screen *ebiten.Image) {
	// Draw background
	screen.Fill(color.RGBA{15, 15, 25, 255})

	// Draw title
	ss.drawTitle(screen)

	// Draw current menu or input
	if ss.projectInputMode {
		ss.drawProjectInput(screen)
	} else {
		switch ss.state {
		case StateMainMenu:
			ss.mainMenu.Draw(screen)
		case StateHarnessSelect:
			ss.harnessMenu.Draw(screen)
		case StateModelSelect:
			ss.modelMenu.Draw(screen)
		case StateProjectSelect:
			ss.projectMenu.Draw(screen)
		}
	}

	// Draw navigation hint
	ss.drawHint(screen)
}

func (ss *StartScreen) drawTitle(screen *ebiten.Image) {
	title := "CODINGGAME"

	// Calculate position for centered title
	titleWidth := len(title) * 6 // Approximate character width
	x := (ss.width - titleWidth) / 2
	y := ss.height / 6

	// Draw title with simple animation
	ebitenutil.DebugPrintAt(screen, title, x, y)

	// Draw subtitle
	subtitle := "Strategy Interface for Claude Code"
	subtitleWidth := len(subtitle) * 6
	ebitenutil.DebugPrintAt(screen, subtitle, (ss.width-subtitleWidth)/2, y+20)
}

func (ss *StartScreen) drawProjectInput(screen *ebiten.Image) {
	// Draw input box
	boxWidth := 500
	boxHeight := 80
	boxX := (ss.width - boxWidth) / 2
	boxY := (ss.height - boxHeight) / 2

	// Background
	vector.DrawFilledRect(
		screen,
		float32(boxX),
		float32(boxY),
		float32(boxWidth),
		float32(boxHeight),
		color.RGBA{30, 30, 45, 230},
		false,
	)

	// Border
	vector.StrokeRect(
		screen,
		float32(boxX),
		float32(boxY),
		float32(boxWidth),
		float32(boxHeight),
		2,
		color.RGBA{80, 80, 120, 255},
		false,
	)

	// Label
	ebitenutil.DebugPrintAt(screen, "Enter project path:", boxX+10, boxY+10)

	// Input text with cursor
	cursor := ""
	if (ss.frameCount/30)%2 == 0 {
		cursor = "_"
	}
	inputText := ss.projectInput + cursor
	ebitenutil.DebugPrintAt(screen, inputText, boxX+10, boxY+35)

	// Hint
	ebitenutil.DebugPrintAt(screen, "[Enter] Confirm  [Escape] Cancel", boxX+10, boxY+60)
}

func (ss *StartScreen) drawHint(screen *ebiten.Image) {
	hint := "[Up/Down or j/k] Navigate  [Enter] Select"
	if ss.state != StateMainMenu {
		hint += "  [Escape] Back"
	}
	ebitenutil.DebugPrintAt(screen, hint, 10, ss.height-20)
}

// OnEnter is called when entering the start screen.
func (ss *StartScreen) OnEnter() {
	ss.state = StateMainMenu
	ss.mainMenu.SelectedIndex = 0
}

// OnExit is called when leaving the start screen.
func (ss *StartScreen) OnExit() {
	// Nothing to clean up
}

// SetRecentProjects updates the list of recent projects.
func (ss *StartScreen) SetRecentProjects(projects []string) {
	ss.recentProjects = projects
	ss.initMenus() // Rebuild menus with updated projects
}
