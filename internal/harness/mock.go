package harness

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockHarness is a test harness that records all interactions for verification.
// Use this in tests to verify that prompts are sent and events are processed.
type MockHarness struct {
	*BaseHarness

	mu        sync.Mutex
	prompts   []string // All prompts sent via SendPrompt
	started   bool
	config    Config
	closeOnce sync.Once

	// Test control fields
	startDelay time.Duration // Delay before Start returns
	stopDelay  time.Duration // Delay before Stop returns
	startErr   error         // Error to return from Start (if set)

	// Event collection for tests
	collectedEvents []Event
	eventCond       *sync.Cond // For WaitForEvent
}

// NewMockHarness creates a new mock harness for testing.
func NewMockHarness() *MockHarness {
	m := &MockHarness{
		BaseHarness:     NewBaseHarness("mock"),
		prompts:         make([]string, 0),
		collectedEvents: make([]Event, 0),
	}
	m.eventCond = sync.NewCond(&m.mu)
	return m
}

// Start records the config and marks the harness as running.
func (m *MockHarness) Start(ctx context.Context, config Config) error {
	m.mu.Lock()
	startDelay := m.startDelay
	startErr := m.startErr
	m.mu.Unlock()

	// Simulate start delay for testing race conditions
	if startDelay > 0 {
		time.Sleep(startDelay)
	}

	// Return configured error if set
	if startErr != nil {
		return startErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsStopped() {
		return fmt.Errorf("mock harness already stopped")
	}
	if m.started {
		return fmt.Errorf("mock harness already started")
	}

	if err := config.Validate(); err != nil {
		return err
	}

	m.config = config
	m.started = true
	m.SetRunning(true)
	return nil
}

// Stop marks the harness as stopped.
func (m *MockHarness) Stop() error {
	m.mu.Lock()
	stopDelay := m.stopDelay
	m.mu.Unlock()

	// Simulate stop delay for testing race conditions
	if stopDelay > 0 {
		time.Sleep(stopDelay)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsStopped() {
		return nil
	}

	m.started = false
	m.SetRunning(false)
	m.closeOnce.Do(func() {
		m.CloseEvents()
	})
	// Signal any waiters that events are no longer coming
	m.eventCond.Broadcast()
	return nil
}

// SendPrompt records the prompt for later verification.
func (m *MockHarness) SendPrompt(prompt string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return fmt.Errorf("mock harness not running")
	}

	m.prompts = append(m.prompts, prompt)
	return nil
}

// Capabilities returns mock capabilities.
func (m *MockHarness) Capabilities() Capabilities {
	return Capabilities{
		SupportedModels: []Model{
			{ID: "mock", Name: "Mock Model", Default: true},
		},
		SupportsHooks:     true,
		SupportsMCP:       true,
		SupportsStreaming: true,
		SupportsResume:    false,
	}
}

// --- Test verification methods ---

// Prompts returns all prompts that were sent to this harness.
func (m *MockHarness) Prompts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.prompts))
	copy(result, m.prompts)
	return result
}

// LastPrompt returns the most recent prompt, or empty string if none.
func (m *MockHarness) LastPrompt() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.prompts) == 0 {
		return ""
	}
	return m.prompts[len(m.prompts)-1]
}

// PromptCount returns the number of prompts sent.
func (m *MockHarness) PromptCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.prompts)
}

// ClearPrompts clears the recorded prompts.
func (m *MockHarness) ClearPrompts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prompts = m.prompts[:0]
}

// Config returns the config passed to Start.
func (m *MockHarness) StartConfig() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config
}

// SimulateEvent sends an event to consumers (for testing event handlers).
func (m *MockHarness) SimulateEvent(event Event) {
	if !m.IsRunning() || m.EventsClosed() {
		return
	}
	m.EventsWritable() <- event
}

// SimulateTurnComplete sends a turn complete event.
func (m *MockHarness) SimulateTurnComplete() {
	m.SimulateEvent(NewEvent(EventTurnComplete).WithSource("mock").Build())
}

// SimulateText sends a text event with the given content.
func (m *MockHarness) SimulateText(text string) {
	m.SimulateEvent(NewEvent(EventText).WithText(text).WithSource("mock").Build())
}

// --- Extended test control methods ---

// SetStartDelay configures a delay before Start() returns.
// This is useful for testing race conditions during startup.
func (m *MockHarness) SetStartDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startDelay = delay
}

// SetStopDelay configures a delay before Stop() returns.
// This is useful for testing race conditions during shutdown.
func (m *MockHarness) SetStopDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopDelay = delay
}

// SetStartError configures an error to be returned from Start().
// This is useful for testing harness start failure handling.
func (m *MockHarness) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startErr = err
}

// CollectEvent records an event for later inspection.
// Call this from a test's event handler to collect events.
func (m *MockHarness) CollectEvent(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collectedEvents = append(m.collectedEvents, event)
	m.eventCond.Broadcast()
}

// CollectedEvents returns all events recorded via CollectEvent.
func (m *MockHarness) CollectedEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Event, len(m.collectedEvents))
	copy(result, m.collectedEvents)
	return result
}

// CollectedEventCount returns the number of collected events.
func (m *MockHarness) CollectedEventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.collectedEvents)
}

// ClearCollectedEvents clears the collected events.
func (m *MockHarness) ClearCollectedEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collectedEvents = m.collectedEvents[:0]
}

// WaitForEventCount waits until at least count events have been collected,
// or the timeout expires. Returns error if timeout occurs.
func (m *MockHarness) WaitForEventCount(count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	m.mu.Lock()
	defer m.mu.Unlock()

	for len(m.collectedEvents) < count {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return fmt.Errorf("timeout waiting for %d events, got %d", count, len(m.collectedEvents))
		}

		// Use a channel-based timeout since sync.Cond doesn't support timeout directly
		done := make(chan struct{})
		go func() {
			defer close(done)
			time.Sleep(remaining)
		}()

		// Wait with periodic checks
		m.mu.Unlock()
		select {
		case <-done:
		case <-time.After(10 * time.Millisecond):
		}
		m.mu.Lock()
	}
	return nil
}

// WaitForEvent waits for an event of the specified type to be collected,
// or the timeout expires. Returns the event or error if timeout occurs.
func (m *MockHarness) WaitForEvent(eventType EventType, timeout time.Duration) (*Event, error) {
	deadline := time.Now().Add(timeout)

	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		// Check if we already have the event
		for i := range m.collectedEvents {
			if m.collectedEvents[i].Type == eventType {
				return &m.collectedEvents[i], nil
			}
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("timeout waiting for event type %v", eventType)
		}

		// Wait with periodic checks
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
	}
}

// SimulateEventAfter sends an event to consumers after the specified delay.
// The event is sent asynchronously in a goroutine.
func (m *MockHarness) SimulateEventAfter(event Event, delay time.Duration) {
	go func() {
		time.Sleep(delay)
		m.SimulateEvent(event)
	}()
}

// SimulateFileRead simulates a file read event.
func (m *MockHarness) SimulateFileRead(filePath string) {
	m.SimulateEvent(NewEvent(EventFileRead).
		WithTool("Read").
		WithToolInput("file_path", filePath).
		WithSource("mock").
		Build())
}

// SimulateFileWrite simulates a file write event.
func (m *MockHarness) SimulateFileWrite(filePath string) {
	m.SimulateEvent(NewEvent(EventFileWrite).
		WithTool("Write").
		WithToolInput("file_path", filePath).
		WithSource("mock").
		Build())
}

// SimulateFileEdit simulates a file edit event.
func (m *MockHarness) SimulateFileEdit(filePath string) {
	m.SimulateEvent(NewEvent(EventFileEdit).
		WithTool("Edit").
		WithToolInput("file_path", filePath).
		WithSource("mock").
		Build())
}
