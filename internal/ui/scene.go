// Package ui provides user interface components for CodingGame including
// menus, text input, and scene management. It follows keyboard-first design
// principles with optional mouse support.
package ui

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene represents a distinct game state/screen that handles its own
// update and draw logic. Examples: StartScreen, GamePlay, Settings.
type Scene interface {
	// Update handles input and updates state. Returns the next scene
	// to transition to, or nil to stay on current scene.
	Update() (Scene, error)

	// Draw renders the scene to the screen.
	Draw(screen *ebiten.Image)

	// OnEnter is called when transitioning into this scene.
	OnEnter()

	// OnExit is called when transitioning out of this scene.
	OnExit()
}

// SceneManager manages scene transitions and the current active scene.
type SceneManager struct {
	current Scene
	width   int
	height  int
}

// NewSceneManager creates a new scene manager with the given initial scene.
func NewSceneManager(initial Scene, width, height int) *SceneManager {
	sm := &SceneManager{
		current: initial,
		width:   width,
		height:  height,
	}
	if initial != nil {
		initial.OnEnter()
	}
	return sm
}

// Update updates the current scene and handles transitions.
func (sm *SceneManager) Update() error {
	if sm.current == nil {
		return nil
	}

	next, err := sm.current.Update()
	if err != nil {
		return err
	}

	if next != nil && next != sm.current {
		sm.current.OnExit()
		sm.current = next
		sm.current.OnEnter()
	}

	return nil
}

// Draw renders the current scene.
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	if sm.current != nil {
		sm.current.Draw(screen)
	}
}

// Current returns the current active scene.
func (sm *SceneManager) Current() Scene {
	return sm.current
}

// SetScene transitions to a new scene immediately.
func (sm *SceneManager) SetScene(scene Scene) {
	if sm.current != nil {
		sm.current.OnExit()
	}
	sm.current = scene
	if scene != nil {
		scene.OnEnter()
	}
}

// Dimensions returns the screen dimensions.
func (sm *SceneManager) Dimensions() (width, height int) {
	return sm.width, sm.height
}
