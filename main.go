package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/game"
)

const (
	screenWidth  = 1280
	screenHeight = 720
)

func main() {
	// Parse command-line arguments for project path
	// If provided, skip the start screen and go directly to the project
	projectPath := ""
	if len(os.Args) > 1 {
		projectPath = os.Args[1]
	}

	// Initialize application
	app, err := game.NewApp(projectPath, screenWidth, screenHeight)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	defer app.Close()

	// Configure window
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("CodingGame")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Run game loop
	if err := ebiten.RunGame(app); err != nil {
		log.Fatalf("application error: %v", err)
	}
}
