package harness

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestableHarness extends MockHarness with event simulation capabilities
type TestableHarness struct {
	*BaseHarness
	startCalled  bool
	stopCalled   bool
	lastPrompt   string
	mu           sync.Mutex
	closeOnce    sync.Once
	simulateFunc func() // Called after SendPrompt to simulate events
}

func NewTestableHarness() *TestableHarness {
	return &TestableHarness{
		BaseHarness: NewBaseHarness("testable"),
	}
}

func (t *TestableHarness) Start(ctx context.Context, config Config) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.startCalled = true
	t.SetRunning(true)
	return nil
}

func (t *TestableHarness) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopCalled = true
	t.SetRunning(false)
	// Use sync.Once to prevent double-close panic
	t.closeOnce.Do(func() {
		t.CloseEvents()
	})
	return nil
}

func (t *TestableHarness) SendPrompt(prompt string) error {
	t.mu.Lock()
	t.lastPrompt = prompt
	simFunc := t.simulateFunc
	t.mu.Unlock()

	// Simulate events after sending prompt
	if simFunc != nil {
		go simFunc()
	}
	return nil
}

func (t *TestableHarness) Capabilities() Capabilities {
	return Capabilities{
		SupportedModels: []Model{
			{ID: "test", Name: "Test Model", Default: true},
		},
	}
}

// SimulateEvent sends an event to the events channel
func (t *TestableHarness) SimulateEvent(event Event) {
	t.EventsWritable() <- event
}

// SetSimulateFunc sets a function to be called after SendPrompt
func (t *TestableHarness) SetSimulateFunc(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.simulateFunc = f
}

// TestFullLifecycle tests the complete harness lifecycle:
// start → send prompt → receive events → stop
func TestFullLifecycle(t *testing.T) {
	h := NewTestableHarness()

	// Set up event simulation
	eventsSent := make(chan struct{})
	h.SetSimulateFunc(func() {
		// Simulate a typical Claude response: text events + turn complete
		h.SimulateEvent(NewEvent(EventText).
			WithText("Analyzing the code...").
			WithSource("test").
			Build())

		h.SimulateEvent(NewEvent(EventFileRead).
			WithTool("Read").
			WithToolInput("file_path", "/test/file.go").
			WithSource("test").
			Build())

		h.SimulateEvent(NewEvent(EventText).
			WithText("Analysis complete.").
			WithSource("test").
			Build())

		h.SimulateEvent(NewEvent(EventTurnComplete).
			WithSource("test").
			Build())

		close(eventsSent)
	})

	// Start
	config := NewConfig("/tmp")
	if err := h.Start(context.Background(), config); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !h.IsRunning() {
		t.Error("Should be running after Start()")
	}

	// Send prompt
	if err := h.SendPrompt("Analyze this code"); err != nil {
		t.Fatalf("SendPrompt() error = %v", err)
	}

	// Collect events
	var events []Event
	timeout := time.After(2 * time.Second)

collectLoop:
	for {
		select {
		case event, ok := <-h.Events():
			if !ok {
				break collectLoop
			}
			events = append(events, event)
			if event.Type == EventTurnComplete {
				break collectLoop
			}
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

	// Verify events
	if len(events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(events))
	}
	if events[0].Type != EventText {
		t.Errorf("First event should be EventText, got %v", events[0].Type)
	}
	if events[1].Type != EventFileRead {
		t.Errorf("Second event should be EventFileRead, got %v", events[1].Type)
	}
	if events[3].Type != EventTurnComplete {
		t.Errorf("Last event should be EventTurnComplete, got %v", events[3].Type)
	}

	// Wait for simulation to complete
	<-eventsSent

	// Stop
	if err := h.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if h.IsRunning() {
		t.Error("Should not be running after Stop()")
	}
}

// TestLifecycleWithContextCancellation tests that context cancellation works
func TestLifecycleWithContextCancellation(t *testing.T) {
	h := NewTestableHarness()

	ctx, cancel := context.WithCancel(context.Background())

	config := NewConfig("/tmp")
	if err := h.Start(ctx, config); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Cancel context
	cancel()

	// Stop should still work
	if err := h.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

// TestConcurrentStartStop tests that concurrent Start/Stop calls are safe
func TestConcurrentStartStop(t *testing.T) {
	h := NewTestableHarness()
	config := NewConfig("/tmp")

	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// Start multiple goroutines trying to start/stop
	for i := 0; i < 10; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			if err := h.Start(context.Background(), config); err != nil {
				// Start may fail if already running, that's OK
				_ = err
			}
		}()

		go func() {
			defer wg.Done()
			if err := h.Stop(); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for unexpected errors
	for err := range errors {
		t.Errorf("Unexpected error during concurrent operations: %v", err)
	}
}

// TestEventChannelBuffering tests that the event channel handles burst traffic
func TestEventChannelBuffering(t *testing.T) {
	h := NewTestableHarness()

	config := NewConfig("/tmp")
	if err := h.Start(context.Background(), config); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Send many events quickly (less than buffer size of 100)
	for i := 0; i < 50; i++ {
		h.SimulateEvent(NewEvent(EventText).
			WithText("Event").
			WithSource("test").
			Build())
	}

	// Should be able to read all events without blocking
	timeout := time.After(1 * time.Second)
	count := 0

	for count < 50 {
		select {
		case <-h.Events():
			count++
		case <-timeout:
			t.Fatalf("Timeout reading events, only got %d of 50", count)
		}
	}

	if err := h.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

// TestEventChannelClosedAfterStop tests that events channel is closed after Stop
func TestEventChannelClosedAfterStop(t *testing.T) {
	h := NewTestableHarness()

	config := NewConfig("/tmp")
	if err := h.Start(context.Background(), config); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := h.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Events channel should be closed
	select {
	case _, ok := <-h.Events():
		if ok {
			t.Error("Expected events channel to be closed after Stop()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Events channel should be closed and readable immediately")
	}
}

// TestMultipleStopCalls tests that multiple Stop() calls are safe
func TestMultipleStopCalls(t *testing.T) {
	h := NewTestableHarness()

	config := NewConfig("/tmp")
	if err := h.Start(context.Background(), config); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Multiple stops should be safe
	for i := 0; i < 5; i++ {
		if err := h.Stop(); err != nil {
			t.Errorf("Stop() call %d error = %v", i, err)
		}
	}
}

// TestEventFiltering tests that events contain expected data
func TestEventFiltering(t *testing.T) {
	h := NewTestableHarness()

	config := NewConfig("/tmp")
	if err := h.Start(context.Background(), config); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Send events with specific data
	h.SimulateEvent(NewEvent(EventFileRead).
		WithTool("Read").
		WithToolInput("file_path", "/path/to/file.go").
		WithSource("test").
		Build())

	h.SimulateEvent(NewEvent(EventBuildRun).
		WithTool("Bash").
		WithToolInput("command", "go build ./...").
		WithSource("test").
		Build())

	// Read and verify events
	timeout := time.After(1 * time.Second)

	select {
	case event := <-h.Events():
		if event.Type != EventFileRead {
			t.Errorf("Expected EventFileRead, got %v", event.Type)
		}
		if event.FilePath() != "/path/to/file.go" {
			t.Errorf("Expected file path '/path/to/file.go', got %q", event.FilePath())
		}
	case <-timeout:
		t.Fatal("Timeout waiting for EventFileRead")
	}

	select {
	case event := <-h.Events():
		if event.Type != EventBuildRun {
			t.Errorf("Expected EventBuildRun, got %v", event.Type)
		}
		if event.Command() != "go build ./..." {
			t.Errorf("Expected command 'go build ./...', got %q", event.Command())
		}
	case <-timeout:
		t.Fatal("Timeout waiting for EventBuildRun")
	}

	if err := h.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}
