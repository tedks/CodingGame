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
	projectPath := "."
	if len(os.Args) > 1 {
		projectPath = os.Args[1]
	}

	// Initialize game
	g, err := game.New(projectPath, screenWidth, screenHeight)
	if err != nil {
		log.Fatalf("failed to initialize game: %v", err)
	}

	// Configure window
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("CodingGame")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Run game loop
	if err := ebiten.RunGame(g); err != nil {
		log.Fatalf("game error: %v", err)
	}
}
