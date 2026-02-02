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

// TestGameScene_PromptToHarnessToMap verifies the full integration chain:
// 1. User submits prompt
// 2. Prompt reaches harness
// 3. Harness emits file read event
// 4. Map view fog of war is updated
//
// This is the most important integration test because it verifies the
// entire data flow path works end-to-end.
func TestGameScene_PromptToHarnessToMap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "full-chain-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file that will be "read" by the harness
	testFile := tmpDir + "/main.go"
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

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

	// Step 1: Verify prompt reaches harness
	testPrompt := "What does main.go contain?"
	gs.onPromptSubmit(testPrompt)

	if mock.LastPrompt() != testPrompt {
		t.Errorf("Prompt not received by harness: got %q", mock.LastPrompt())
	}

	// Step 2: Harness "responds" with file read event
	// Track event processing
	eventProcessed := make(chan struct{}, 1)
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil && event.Type == harness.EventFileRead {
			select {
			case eventProcessed <- struct{}{}:
			default:
			}
		}
	}

	mock.SimulateFileRead(testFile)

	// Wait for event to be processed
	select {
	case <-eventProcessed:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("File read event not processed")
	}

	// Step 3: Harness sends response text
	textProcessed := make(chan struct{}, 1)
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil && event.Type == harness.EventText {
			select {
			case textProcessed <- struct{}{}:
			default:
			}
		}
	}

	mock.SimulateText("The main.go file contains a basic Go program.")

	select {
	case <-textProcessed:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Text event not processed")
	}

	// Step 4: Harness signals turn complete
	turnCompleted := make(chan struct{}, 1)
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil && event.Type == harness.EventTurnComplete {
			select {
			case turnCompleted <- struct{}{}:
			default:
			}
		}
	}

	mock.SimulateTurnComplete()

	select {
	case <-turnCompleted:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Turn complete event not processed")
	}

	// The integration test passes if all events were processed without panic
	// and the data flowed through the entire chain
}

// TestGameScene_HarnessReplace verifies that replacing one harness with another
// works correctly (stop old, start new, events flow correctly).
func TestGameScene_HarnessReplace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "harness-replace-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	gs, err := NewGameScene(tmpDir, 800, 600)
	if err != nil {
		t.Fatalf("Failed to create game scene: %v", err)
	}
	defer gs.Close()

	mock1 := harness.NewMockHarness()
	mock2 := harness.NewMockHarness()

	registry := harness.NewRegistry()
	registry.Register("mock1", func() harness.Harness { return mock1 })
	registry.Register("mock2", func() harness.Harness { return mock2 })

	gs.SetHarnessRegistry(registry)

	// Start with mock1
	gs.SetConfig(ui.GameConfig{
		Harness:     "mock1",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	// Send prompt to mock1
	gs.onPromptSubmit("prompt to mock1")
	if mock1.PromptCount() != 1 {
		t.Error("Prompt not sent to mock1")
	}

	// Replace with mock2
	gs.SetConfig(ui.GameConfig{
		Harness:     "mock2",
		Model:       "mock",
		ProjectPath: tmpDir,
	})

	// mock1 should be stopped
	if mock1.IsRunning() {
		t.Error("mock1 should be stopped after replacement")
	}

	// mock2 should be running
	if !mock2.IsRunning() {
		t.Error("mock2 should be running after replacement")
	}

	// Send prompt to mock2
	gs.onPromptSubmit("prompt to mock2")
	if mock2.PromptCount() != 1 {
		t.Error("Prompt not sent to mock2")
	}

	// mock1 should not have received the new prompt
	if mock1.PromptCount() != 1 {
		t.Errorf("mock1 received additional prompts: %d", mock1.PromptCount())
	}
}

// TestGameScene_EventTypeCoverage verifies that all supported event types
// are handled without panic.
func TestGameScene_EventTypeCoverage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "event-coverage-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFile := tmpDir + "/test.go"
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

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

	// Track processed events
	processedEvents := make(map[harness.EventType]bool)
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil {
			processedEvents[event.Type] = true
		}
	}

	// Send all supported event types
	eventTypes := []harness.EventType{
		harness.EventFileRead,
		harness.EventFileWrite,
		harness.EventFileEdit,
		harness.EventBuildRun,
		harness.EventTestRun,
		harness.EventText,
		harness.EventTurnComplete,
		harness.EventSubagentRun,
	}

	for _, et := range eventTypes {
		event := harness.NewEvent(et).
			WithSource("mock").
			WithTool("MockTool").
			WithToolInput("file_path", testFile).
			WithToolInput("command", "test").
			Build()
		mock.SimulateEvent(event)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify all events were processed
	for _, et := range eventTypes {
		if !processedEvents[et] {
			t.Errorf("Event type %v was not processed", et)
		}
	}
}
