package game

import (
	"os"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
	"github.com/tedks/CodingGame/internal/ui"
)

// TestGameScene_PromptSentToHarness verifies that when a prompt is submitted
// through the game scene, it actually reaches the harness via SendPrompt.
//
// This test catches the bug where the prompt callback wasn't wired up.
func TestGameScene_PromptSentToHarness(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "harness-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create game scene
	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	// Create a mock harness
	mock := harness.NewMockHarness()

	// Create registry with mock harness
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })

	// Inject registry and configure to use mock harness
	gs.SetHarnessRegistry(registry)
	gs.SetConfig(ui.GameConfig{
		Harness:     "mock",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	// Verify harness is running
	if !mock.IsRunning() {
		t.Fatal("Mock harness should be running after SetConfig")
	}

	// Simulate prompt submission through the game scene's callback
	testPrompt := "What files are in this project?"

	// The onPromptSubmit callback should be set by startHarness
	if gs.onPromptSubmit == nil {
		t.Fatal("onPromptSubmit callback not set - prompt wiring is broken")
	}

	// Call the callback directly (simulating what happens when prompt panel submits)
	gs.onPromptSubmit(testPrompt)

	// Verify the prompt was sent to the harness
	if mock.PromptCount() != 1 {
		t.Errorf("Expected 1 prompt sent, got %d", mock.PromptCount())
	}
	if mock.LastPrompt() != testPrompt {
		t.Errorf("Expected prompt %q, got %q", testPrompt, mock.LastPrompt())
	}
}

// TestGameScene_HarnessEventsProcessed verifies that events from the harness
// are received and processed by the game scene.
func TestGameScene_HarnessEventsProcessed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harness-events-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file for the event to reference
	testFile := tmpDir + "/test.go"
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	eventProcessed := make(chan *harness.Event, 1)
	gs.harnessEventHook = func(event *harness.Event) {
		if event == nil || event.Type != harness.EventFileRead {
			return
		}
		select {
		case eventProcessed <- event:
		default:
		}
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

	// Send a file read event from the mock harness
	mock.SimulateEvent(harness.NewEvent(harness.EventFileRead).
		WithTool("Read").
		WithToolInput("file_path", testFile).
		WithSource("mock").
		Build())

	select {
	case <-eventProcessed:
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for harness event to be processed")
	}

	// The event should have been processed (we can't easily verify the effect
	// without exposing more internal state, but at least we verify no panics)
}

// TestGameScene_MultiplePrompts verifies that multiple prompts can be sent.
func TestGameScene_MultiplePrompts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "multi-prompt-*")
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

	// Send multiple prompts
	prompts := []string{
		"First prompt",
		"Second prompt",
		"Third prompt",
	}

	for _, p := range prompts {
		gs.onPromptSubmit(p)
	}

	// Verify all prompts were sent
	if mock.PromptCount() != len(prompts) {
		t.Errorf("Expected %d prompts, got %d", len(prompts), mock.PromptCount())
	}

	received := mock.Prompts()
	for i, expected := range prompts {
		if received[i] != expected {
			t.Errorf("Prompt %d: expected %q, got %q", i, expected, received[i])
		}
	}
}

// TestGameScene_NoHarnessConfigured verifies graceful behavior when no harness is set.
func TestGameScene_NoHarnessConfigured(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "no-harness-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	// Don't set config - no harness configured
	// The callback should be nil
	if gs.onPromptSubmit != nil {
		t.Error("onPromptSubmit should be nil when no harness is configured")
	}
}
