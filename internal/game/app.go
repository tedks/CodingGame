package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/harness"
	hclaude "github.com/tedks/CodingGame/internal/harness/claude"
	"github.com/tedks/CodingGame/internal/ui"
)

// App is the top-level application that manages scenes and implements ebiten.Game.
// It handles the transition from start screen to gameplay.
type App struct {
	width        int
	height       int
	sceneManager *ui.SceneManager

	// Shared harness registry for all scenes
	harnessRegistry *harness.Registry

	// Direct project path bypasses start screen
	directProjectPath string
}

// NewApp creates a new application instance.
// If projectPath is provided and non-empty, skip the start screen and go directly to game.
// If projectPath is empty or ".", show the start screen first.
func NewApp(projectPath string, width, height int) (*App, error) {
	app := &App{
		width:  width,
		height: height,
	}

	// Create shared harness registry and register available harnesses
	app.harnessRegistry = harness.NewRegistry()
	app.harnessRegistry.Register("claude-code", hclaude.NewHarness)

	// Determine if we should skip the start screen
	skipStartScreen := projectPath != "" && projectPath != "."

	if skipStartScreen {
		// Go directly to game with the provided project path
		app.directProjectPath = projectPath
		gameScene, err := NewGameScene(projectPath, width, height)
		if err != nil {
			return nil, err
		}
		gameScene.SetHarnessRegistry(app.harnessRegistry)
		app.sceneManager = ui.NewSceneManager(gameScene, width, height)
	} else {
		// Show start screen with shared registry
		startScreen := ui.NewStartScreen(width, height, func(config ui.GameConfig) {
			// When configuration is complete, transition to game
			app.onStartScreenComplete(config)
		})
		startScreen.SetHarnessRegistry(app.harnessRegistry)
		app.sceneManager = ui.NewSceneManager(startScreen, width, height)
	}

	return app, nil
}

// onStartScreenComplete is called when the user completes the start screen configuration.
func (a *App) onStartScreenComplete(config ui.GameConfig) {
	// Create and transition to the game scene
	gameScene, err := NewGameScene(config.ProjectPath, a.width, a.height)
	if err != nil {
		// TODO: Show error screen or return to start screen
		return
	}

	// Inject shared harness registry (must be before SetConfig)
	gameScene.SetHarnessRegistry(a.harnessRegistry)

	// Store config for later use (model selection, etc.)
	gameScene.SetConfig(config)

	a.sceneManager.SetScene(gameScene)
}

// Update implements ebiten.Game.
func (a *App) Update() error {
	return a.sceneManager.Update()
}

// Draw implements ebiten.Game.
func (a *App) Draw(screen *ebiten.Image) {
	a.sceneManager.Draw(screen)
}

// Layout implements ebiten.Game.
func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return a.width, a.height
}

// Close cleans up application resources.
func (a *App) Close() error {
	// If current scene is a GameScene, close it
	if gs, ok := a.sceneManager.Current().(*GameScene); ok {
		return gs.Close()
	}
	return nil
}
