package game

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/harness"
	"github.com/tedks/CodingGame/internal/input"
	"github.com/tedks/CodingGame/internal/testutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// TestGameScene_RapidModeSwitch verifies that rapidly switching modes
// 100 times always ends in Normal mode.
func TestGameScene_RapidModeSwitch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rapid-mode-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	source := testutil.NewTestInputSource()
	gs.SetInputSource(source)

	// Alternate between Insert and Normal modes 100 times
	// The pattern is: Enter (Insert), Escape (Normal), Enter, Escape, ...
	// After 100 switches (50 pairs), we should be in Normal mode
	for i := 0; i < 50; i++ {
		source.QueueKeyPress(ebiten.KeyEnter)  // Enter Insert mode
		source.QueueKeyPress(ebiten.KeyEscape) // Exit to Normal mode
	}

	// Process all events
	for source.HasPendingEvents() {
		source.AdvanceFrame()
		gs.Update()
	}

	// Should be in Normal mode
	if gs.inputHandler.Mode() != input.ModeNormal {
		t.Errorf("Expected ModeNormal after 100 mode switches, got %v", gs.inputHandler.Mode())
	}
}

// TestGameScene_RapidViewSwitch verifies that rapidly switching views
// leaves us on the expected final view.
func TestGameScene_RapidViewSwitch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rapid-view-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	source := testutil.NewTestInputSource()
	gs.SetInputSource(source)

	// Cycle through all views multiple times
	viewKeys := []ebiten.Key{
		ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
		ebiten.Key5, ebiten.Key6, ebiten.Key7,
	}

	for cycle := 0; cycle < 10; cycle++ {
		for _, key := range viewKeys {
			source.QueueKeyPress(key)
		}
	}

	// Process all events
	for source.HasPendingEvents() {
		source.AdvanceFrame()
		gs.Update()
	}

	// Should be on View 7 (MultiAgent) after cycling
	if gs.currentView != input.ViewMultiAgent {
		t.Errorf("Expected ViewMultiAgent after rapid cycling, got %v", gs.currentView)
	}
}

// TestGameScene_CloseDuringUpdate verifies that calling Close() while
// Update() is potentially running doesn't cause panic.
func TestGameScene_CloseDuringUpdate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "close-during-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}

	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })
	gs.SetHarnessRegistry(registry)

	gs.SetConfig(ui.GameConfig{
		Harness:     "mock",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	source := testutil.NewTestInputSource()
	gs.SetInputSource(source)

	// Queue many events to process
	for i := 0; i < 100; i++ {
		source.QueueKeyHold(ebiten.KeyH, 1)
		mock.SimulateText("event")
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Goroutine that calls Update() repeatedly
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				source.AdvanceFrame()
				gs.Update()
				time.Sleep(time.Microsecond)
			}
		}
	}()

	// Give Update some time to run
	time.Sleep(10 * time.Millisecond)

	// Close should not panic even if Update is running
	close(done)
	err = gs.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	wg.Wait()
}

// TestGameScene_ConcurrentPromptSubmit verifies that concurrent prompt
// submissions don't cause race conditions.
func TestGameScene_ConcurrentPromptSubmit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent-prompt-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })
	gs.SetHarnessRegistry(registry)

	gs.SetConfig(ui.GameConfig{
		Harness:     "mock",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	var wg sync.WaitGroup
	const goroutines = 10
	const promptsPerGoroutine = 10

	// Launch multiple goroutines that submit prompts concurrently
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < promptsPerGoroutine; i++ {
				if gs.onPromptSubmit != nil {
					gs.onPromptSubmit("prompt")
				}
			}
		}(g)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent prompt submissions deadlocked")
	}

	// Verify all prompts were received
	expectedCount := goroutines * promptsPerGoroutine
	actualCount := mock.PromptCount()
	if actualCount != expectedCount {
		t.Errorf("Expected %d prompts, got %d", expectedCount, actualCount)
	}
}

// TestGameScene_StressEvents verifies that high event throughput
// doesn't cause issues.
func TestGameScene_StressEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stress-events-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })
	gs.SetHarnessRegistry(registry)

	gs.SetConfig(ui.GameConfig{
		Harness:     "mock",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	source := testutil.NewTestInputSource()
	gs.SetInputSource(source)

	const eventCount = 500

	// Flood with events from harness
	go func() {
		for i := 0; i < eventCount; i++ {
			mock.SimulateText("stress event")
		}
	}()

	// Also process input events simultaneously
	go func() {
		for i := 0; i < 100; i++ {
			source.QueueKeyHold(ebiten.KeyH, 1)
			source.AdvanceFrame()
		}
	}()

	// Run Update() many times
	for i := 0; i < 100; i++ {
		gs.Update()
		time.Sleep(time.Microsecond)
	}

	// Test passes if no panic or deadlock
}

// TestGameScene_RapidConfigChange verifies that rapidly changing config
// while events are processing doesn't cause issues.
func TestGameScene_RapidConfigChange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rapid-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	registry := harness.NewRegistry()

	// Create multiple harnesses
	harnesses := make([]*harness.MockHarness, 5)
	for i := 0; i < 5; i++ {
		harnesses[i] = harness.NewMockHarness()
		name := string(rune('a' + i))
		registry.Register(name, func(h *harness.MockHarness) harness.HarnessFactory {
			return func() harness.Harness { return h }
		}(harnesses[i]))
	}

	gs.SetHarnessRegistry(registry)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Goroutine that sends events
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				for _, h := range harnesses {
					h.SimulateText("event")
				}
				time.Sleep(time.Microsecond)
			}
		}
	}()

	// Rapidly change config
	for i := 0; i < 20; i++ {
		name := string(rune('a' + (i % 5)))
		gs.SetConfig(ui.GameConfig{
			Harness:     name,
			Model:       "mock",
			ProjectPath: tmpDir,
		})
		time.Sleep(time.Microsecond)
	}

	close(done)
	wg.Wait()

	// Test passes if no panic or deadlock
}

// TestGameScene_ChaosInput verifies that random-ish input sequences
// don't cause crashes.
func TestGameScene_ChaosInput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chaos-input-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	source := testutil.NewTestInputSource()
	gs.SetInputSource(source)

	// Generate chaos input - mix of keys, mouse, text
	keys := []ebiten.Key{
		ebiten.KeyH, ebiten.KeyJ, ebiten.KeyK, ebiten.KeyL,
		ebiten.KeyEnter, ebiten.KeyEscape,
		ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
		ebiten.Key5, ebiten.Key6, ebiten.Key7,
		ebiten.KeyEqual, ebiten.KeyMinus,
	}

	for i := 0; i < 200; i++ {
		switch i % 5 {
		case 0:
			source.QueueKeyPress(keys[i%len(keys)])
		case 1:
			source.QueueKeyHold(keys[i%len(keys)], 1)
		case 2:
			source.QueueMouseMove(i*3%800, i*2%600)
		case 3:
			source.QueueMouseClick(ebiten.MouseButtonLeft)
		case 4:
			source.QueueCharInput(rune('a' + i%26))
		}
	}

	// Process all events
	for source.HasPendingEvents() {
		source.AdvanceFrame()
		gs.Update()
	}

	// Test passes if no panic
}
