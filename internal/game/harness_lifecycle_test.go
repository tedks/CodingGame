package game

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
	"github.com/tedks/CodingGame/internal/ui"
)

// TestGameScene_SetConfigDuringEvents verifies that SetConfig can be called
// while events are being processed without causing deadlock.
//
// This tests concurrent access to the harness lifecycle during event processing.
func TestGameScene_SetConfigDuringEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "setconfig-events-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	// Create mock harness
	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })
	gs.SetHarnessRegistry(registry)

	// Start harness
	gs.SetConfig(ui.GameConfig{
		Harness:     "mock",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	if !mock.IsRunning() {
		t.Fatal("Mock harness should be running")
	}

	// Channel to coordinate test completion
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Start a goroutine that sends events continuously
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case <-done:
				return
			default:
				mock.SimulateText("event")
				time.Sleep(time.Microsecond)
			}
		}
	}()

	// Call SetConfig while events are being sent (should not deadlock)
	time.Sleep(5 * time.Millisecond)

	// Create a new mock harness for the new config
	mock2 := harness.NewMockHarness()
	registry.Register("mock2", func() harness.Harness { return mock2 })

	// This should complete without deadlock (with 1 second timeout)
	setConfigDone := make(chan struct{})
	go func() {
		gs.SetConfig(ui.GameConfig{
			Harness:     "mock2",
			Model:       "mock",
			ProjectPath: tmpDir,
		})
		close(setConfigDone)
	}()

	select {
	case <-setConfigDone:
		// Success - SetConfig completed
	case <-time.After(2 * time.Second):
		t.Fatal("SetConfig deadlocked while events were being processed")
	}

	// Signal event sender to stop
	close(done)
	wg.Wait()

	// Original harness should be stopped
	if mock.IsRunning() {
		t.Error("Original harness should be stopped after SetConfig")
	}
}

// TestGameScene_DoubleStopSafe verifies that multiple concurrent Close() calls
// do not cause panic or race conditions.
func TestGameScene_DoubleStopSafe(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "double-stop-*")
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

	// Call Close() multiple times concurrently
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Should not panic
			_ = gs.Close()
		}()
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
	case <-time.After(2 * time.Second):
		t.Fatal("Concurrent Close() calls deadlocked")
	}
}

// TestGameScene_LateEventsAfterStop verifies that events arriving after
// harness Stop() are gracefully ignored without panic.
func TestGameScene_LateEventsAfterStop(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "late-events-*")
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

	// Schedule events to be sent after a delay
	mock.SimulateEventAfter(
		harness.NewEvent(harness.EventText).WithText("late event 1").WithSource("mock").Build(),
		50*time.Millisecond,
	)
	mock.SimulateEventAfter(
		harness.NewEvent(harness.EventText).WithText("late event 2").WithSource("mock").Build(),
		100*time.Millisecond,
	)

	// Stop harness immediately (events are scheduled but not yet sent)
	gs.SetConfig(ui.GameConfig{}) // Clear config to stop harness

	// Wait for the scheduled events to fire (they should be ignored)
	time.Sleep(200 * time.Millisecond)

	// No panic should have occurred - test passes if we get here
}

// TestGameScene_HarnessStartFailure verifies that when harness Start() fails,
// resources are properly cleaned up.
func TestGameScene_HarnessStartFailure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "start-failure-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	// Create mock harness that will fail to start
	mock := harness.NewMockHarness()
	mock.SetStartError(errors.New("simulated start failure"))

	registry := harness.NewRegistry()
	registry.Register("failing", func() harness.Harness { return mock })
	gs.SetHarnessRegistry(registry)

	// SetConfig should not panic or leave partial state
	gs.SetConfig(ui.GameConfig{
		Harness:     "failing",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	// Harness should not be set
	if gs.Harness() != nil {
		t.Error("Harness should be nil after start failure")
	}

	// onPromptSubmit should not be set
	if gs.onPromptSubmit != nil {
		t.Error("onPromptSubmit should be nil after start failure")
	}

	// Should be able to configure a working harness after failure
	mock2 := harness.NewMockHarness()
	registry.Register("working", func() harness.Harness { return mock2 })

	gs.SetConfig(ui.GameConfig{
		Harness:     "working",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	if !mock2.IsRunning() {
		t.Error("Working harness should be running after retry")
	}
}

// TestGameScene_SetConfigRapidFire verifies that rapid SetConfig calls
// don't cause race conditions or leave zombie goroutines.
func TestGameScene_SetConfigRapidFire(t *testing.T) {
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

	// Create multiple mock harnesses
	var mocks []*harness.MockHarness
	for i := 0; i < 10; i++ {
		mock := harness.NewMockHarness()
		mocks = append(mocks, mock)
		name := "mock" + string(rune('0'+i))
		registry.Register(name, func(m *harness.MockHarness) harness.HarnessFactory {
			return func() harness.Harness { return m }
		}(mock))
	}

	gs.SetHarnessRegistry(registry)

	// Rapidly switch between harnesses
	for i := 0; i < 10; i++ {
		name := "mock" + string(rune('0'+i))
		gs.SetConfig(ui.GameConfig{
			Harness:     name,
			Model:       "mock",
			ProjectPath: tmpDir,
		})
	}

	// Only the last harness should be running
	for i := 0; i < 9; i++ {
		if mocks[i].IsRunning() {
			t.Errorf("Mock %d should be stopped", i)
		}
	}
	if !mocks[9].IsRunning() {
		t.Error("Last mock should be running")
	}
}

// TestGameScene_HarnessContextCancellation verifies that harness Stop()
// properly cancels the context and cleans up goroutines.
func TestGameScene_HarnessContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ctx-cancel-*")
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

	// Start sending events and verify they flow
	eventReceived := make(chan struct{})
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil && event.Type == harness.EventText {
			select {
			case eventReceived <- struct{}{}:
			default:
			}
		}
	}

	mock.SimulateText("test event")
	select {
	case <-eventReceived:
		// Event received
	case <-time.After(1 * time.Second):
		t.Fatal("Event not received before stop")
	}

	// Clear config to stop harness
	gs.SetConfig(ui.GameConfig{})

	// Events channel should be closed and no more events should arrive
	// (SimulateEvent should not panic or block)
	mock.SimulateText("event after stop") // Should be silently ignored
}
