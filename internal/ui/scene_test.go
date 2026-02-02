package ui

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSceneManagerBasics(t *testing.T) {
	// Test NewSceneManager with nil
	sm := NewSceneManager(nil, 800, 600)

	if sm.Current() != nil {
		t.Error("expected current scene to be nil")
	}

	w, h := sm.Dimensions()
	if w != 800 || h != 600 {
		t.Errorf("expected dimensions 800x600, got %dx%d", w, h)
	}
}

func TestSceneManager_UpdateNil(t *testing.T) {
	sm := NewSceneManager(nil, 800, 600)

	// Should not panic or error with nil scene
	err := sm.Update()
	if err != nil {
		t.Errorf("Update() returned error: %v", err)
	}
}

func TestSceneManager_DrawNil(t *testing.T) {
	sm := NewSceneManager(nil, 800, 600)

	// Should not panic with nil scene
	sm.Draw(nil)
}

// Mock scene implementations for testing

type selfReturningScene struct{ updateCount int }

func (s *selfReturningScene) Update() (Scene, error) {
	s.updateCount++
	return s, nil
}
func (s *selfReturningScene) Draw(screen *ebiten.Image) {}
func (s *selfReturningScene) OnEnter()                  {}
func (s *selfReturningScene) OnExit()                   {}

type errorScene struct{ err error }

func (s *errorScene) Update() (Scene, error) { return nil, s.err }
func (s *errorScene) Draw(screen *ebiten.Image) {}
func (s *errorScene) OnEnter()                  {}
func (s *errorScene) OnExit()                   {}

type trackingScene struct {
	enteredCount int
	exitedCount  int
	nextScene    Scene
}

func (s *trackingScene) Update() (Scene, error) { return s.nextScene, nil }
func (s *trackingScene) Draw(screen *ebiten.Image) {}
func (s *trackingScene) OnEnter()                  { s.enteredCount++ }
func (s *trackingScene) OnExit()                   { s.exitedCount++ }

func TestSceneManager_SceneReturnsSelf(t *testing.T) {
	scene := &selfReturningScene{}
	sm := NewSceneManager(scene, 800, 600)

	for i := 0; i < 5; i++ {
		if err := sm.Update(); err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}
	}

	if scene.updateCount != 5 {
		t.Errorf("expected updateCount=5, got %d", scene.updateCount)
	}
	if sm.Current() != scene {
		t.Error("expected current scene to remain unchanged")
	}
}

func TestSceneManager_SetSceneToSame(t *testing.T) {
	scene := &trackingScene{}
	sm := NewSceneManager(scene, 800, 600)

	if scene.enteredCount != 1 {
		t.Errorf("expected enteredCount=1 after NewSceneManager, got %d", scene.enteredCount)
	}

	sm.SetScene(scene)

	// Documents current behavior: SetScene to same scene calls OnExit then OnEnter
	if scene.exitedCount != 1 {
		t.Errorf("SetScene(same): expected exitedCount=1, got %d", scene.exitedCount)
	}
	if scene.enteredCount != 2 {
		t.Errorf("SetScene(same): expected enteredCount=2, got %d", scene.enteredCount)
	}
}

func TestSceneManager_UpdateError(t *testing.T) {
	expectedErr := errors.New("test error")
	scene := &errorScene{err: expectedErr}
	sm := NewSceneManager(scene, 800, 600)

	err := sm.Update()
	if err == nil {
		t.Fatal("expected error from Update()")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestSceneManager_TransitionCallsLifecycle(t *testing.T) {
	scene1 := &trackingScene{}
	scene2 := &trackingScene{}
	scene1.nextScene = scene2

	sm := NewSceneManager(scene1, 800, 600)

	if scene1.enteredCount != 1 {
		t.Errorf("expected scene1 enteredCount=1, got %d", scene1.enteredCount)
	}

	if err := sm.Update(); err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if scene1.exitedCount != 1 {
		t.Errorf("expected scene1 exitedCount=1, got %d", scene1.exitedCount)
	}
	if scene2.enteredCount != 1 {
		t.Errorf("expected scene2 enteredCount=1, got %d", scene2.enteredCount)
	}
	if sm.Current() != scene2 {
		t.Error("expected current scene to be scene2")
	}
}

func TestSceneManager_TransitionToNil(t *testing.T) {
	scene := &trackingScene{}
	sm := NewSceneManager(scene, 800, 600)

	if err := sm.Update(); err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if scene.exitedCount != 0 {
		t.Errorf("expected exitedCount=0 when Update returns nil, got %d", scene.exitedCount)
	}
	if sm.Current() != scene {
		t.Error("expected current scene to remain unchanged")
	}
}

func TestSceneManager_SetSceneToNil(t *testing.T) {
	scene := &trackingScene{}
	sm := NewSceneManager(scene, 800, 600)

	sm.SetScene(nil)

	if scene.exitedCount != 1 {
		t.Errorf("expected exitedCount=1 after SetScene(nil), got %d", scene.exitedCount)
	}
	if sm.Current() != nil {
		t.Error("expected current scene to be nil")
	}
	if err := sm.Update(); err != nil {
		t.Fatalf("Update() with nil scene returned error: %v", err)
	}
}
