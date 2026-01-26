package harness

import (
	"context"
	"fmt"
	"sync"
)

// MockHarness is a test harness that records all interactions for verification.
// Use this in tests to verify that prompts are sent and events are processed.
type MockHarness struct {
	*BaseHarness

	mu      sync.Mutex
	prompts []string // All prompts sent via SendPrompt
	started bool
	config  Config
}

// NewMockHarness creates a new mock harness for testing.
func NewMockHarness() *MockHarness {
	return &MockHarness{
		BaseHarness: NewBaseHarness("mock"),
		prompts:     make([]string, 0),
	}
}

// Start records the config and marks the harness as running.
func (m *MockHarness) Start(ctx context.Context, config Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	m.started = false
	m.SetRunning(false)
	m.CloseEvents()
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
