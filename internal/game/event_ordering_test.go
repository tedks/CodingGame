package game

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
	"github.com/tedks/CodingGame/internal/ui"
)

// TestGameScene_OutOfOrderFileEvents verifies that file events arriving
// in unexpected order (Write before Read) don't cause panics or crashes.
//
// In practice, a harness might emit events in different orders based on
// how the AI agent is working, and the game scene should handle all orderings.
func TestGameScene_OutOfOrderFileEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "out-of-order-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := []string{
		tmpDir + "/file1.go",
		tmpDir + "/file2.go",
		tmpDir + "/file3.go",
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
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

	// Track events processed
	var eventCount int32
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil {
			atomic.AddInt32(&eventCount, 1)
		}
	}

	// Send events in "wrong" order: Write before Read
	mock.SimulateFileWrite(files[0])
	mock.SimulateFileRead(files[0])

	// Edit without Read
	mock.SimulateFileEdit(files[1])

	// Multiple Reads of same file
	mock.SimulateFileRead(files[2])
	mock.SimulateFileRead(files[2])
	mock.SimulateFileRead(files[2])

	// Write to file that was never read
	mock.SimulateFileWrite(files[2])

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	// All events should be processed without panic
	if atomic.LoadInt32(&eventCount) != 7 {
		t.Errorf("Expected 7 events processed, got %d", atomic.LoadInt32(&eventCount))
	}
}

// TestGameScene_EventFlood verifies that a large number of events (200+)
// can be processed without deadlock or dropped events.
func TestGameScene_EventFlood(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "event-flood-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
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

	// Track events processed
	var processedCount int32
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil {
			atomic.AddInt32(&processedCount, 1)
		}
	}

	const eventCount = 200

	// Send a flood of events
	for i := 0; i < eventCount; i++ {
		mock.SimulateText("event " + string(rune('0'+(i%10))))
	}

	// Wait for processing with timeout
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&processedCount) >= eventCount {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	processed := atomic.LoadInt32(&processedCount)
	if processed < eventCount {
		t.Errorf("Expected at least %d events processed, got %d (possible deadlock or event drop)", eventCount, processed)
	}
}

// TestGameScene_InterleavedEventTypes verifies that mixed event types
// (text, file read, file write, build, test) are all processed correctly.
func TestGameScene_InterleavedEventTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "interleaved-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := []string{
		tmpDir + "/main.go",
		tmpDir + "/util.go",
	}
	for _, f := range testFiles {
		if err := os.WriteFile(f, []byte("package main"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
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

	// Track events by type
	var mu sync.Mutex
	eventsByType := make(map[harness.EventType]int)
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil {
			mu.Lock()
			eventsByType[event.Type]++
			mu.Unlock()
		}
	}

	// Send interleaved events mimicking a real agent session
	mock.SimulateText("Let me analyze this code...")
	mock.SimulateFileRead(testFiles[0])
	mock.SimulateText("I see the main function...")
	mock.SimulateFileRead(testFiles[1])
	mock.SimulateText("Now I'll make a change...")
	mock.SimulateFileEdit(testFiles[0])
	mock.SimulateEvent(harness.NewEvent(harness.EventBuildRun).
		WithTool("Bash").
		WithToolInput("command", "go build ./...").
		WithSource("mock").
		Build())
	mock.SimulateText("Build succeeded, running tests...")
	mock.SimulateEvent(harness.NewEvent(harness.EventTestRun).
		WithTool("Bash").
		WithToolInput("command", "go test ./...").
		WithSource("mock").
		Build())
	mock.SimulateText("All tests pass!")
	mock.SimulateTurnComplete()

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Verify all event types were processed
	if eventsByType[harness.EventText] != 5 {
		t.Errorf("Expected 5 text events, got %d", eventsByType[harness.EventText])
	}
	if eventsByType[harness.EventFileRead] != 2 {
		t.Errorf("Expected 2 file read events, got %d", eventsByType[harness.EventFileRead])
	}
	if eventsByType[harness.EventFileEdit] != 1 {
		t.Errorf("Expected 1 file edit event, got %d", eventsByType[harness.EventFileEdit])
	}
	if eventsByType[harness.EventBuildRun] != 1 {
		t.Errorf("Expected 1 build event, got %d", eventsByType[harness.EventBuildRun])
	}
	if eventsByType[harness.EventTestRun] != 1 {
		t.Errorf("Expected 1 test event, got %d", eventsByType[harness.EventTestRun])
	}
	if eventsByType[harness.EventTurnComplete] != 1 {
		t.Errorf("Expected 1 turn complete event, got %d", eventsByType[harness.EventTurnComplete])
	}
}

// TestGameScene_EventOrderPreserved verifies that events are processed
// in the order they are sent.
func TestGameScene_EventOrderPreserved(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "event-order-*")
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

	// Track event order
	var mu sync.Mutex
	var receivedOrder []string
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil && event.Type == harness.EventText {
			mu.Lock()
			receivedOrder = append(receivedOrder, event.Text)
			mu.Unlock()
		}
	}

	// Send numbered events
	expectedOrder := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	for _, text := range expectedOrder {
		mock.SimulateText(text)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d events, got %d", len(expectedOrder), len(receivedOrder))
	}

	for i, expected := range expectedOrder {
		if receivedOrder[i] != expected {
			t.Errorf("Event %d: expected %q, got %q", i, expected, receivedOrder[i])
		}
	}
}

// TestGameScene_ConcurrentEventSources verifies that events from multiple
// concurrent senders are all processed without deadlock.
func TestGameScene_ConcurrentEventSources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent-events-*")
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

	// Track events
	var processedCount int32
	gs.harnessEventHook = func(event *harness.Event) {
		if event != nil {
			atomic.AddInt32(&processedCount, 1)
		}
	}

	const sendersCount = 5
	const eventsPerSender = 20
	expectedTotal := int32(sendersCount * eventsPerSender)

	var wg sync.WaitGroup

	// Launch multiple concurrent senders
	for s := 0; s < sendersCount; s++ {
		wg.Add(1)
		go func(senderID int) {
			defer wg.Done()
			for i := 0; i < eventsPerSender; i++ {
				mock.SimulateText("event from sender")
			}
		}(s)
	}

	// Wait for senders to complete
	wg.Wait()

	// Wait for processing with timeout
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&processedCount) >= expectedTotal {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	processed := atomic.LoadInt32(&processedCount)
	if processed < expectedTotal {
		t.Errorf("Expected at least %d events, got %d (possible deadlock)", expectedTotal, processed)
	}
}
