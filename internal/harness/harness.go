// Package harness provides an abstraction layer for AI coding agent CLIs.
// It supports multiple harness providers (Claude Code, Codex, Gemini, etc.)
// and enables both main agent execution and advisor subagent spawning.
//
// The harness system is designed to be provider-agnostic, translating
// provider-specific JSON output into unified events that drive game
// visualizations like fog of war reveals, tile highlights, and metrics.
package harness

import (
	"context"
	"sync"
)

// Harness represents an interface to an AI coding agent CLI.
// Each harness implementation handles a specific provider's CLI
// (e.g., claude for Claude Code, codex for OpenAI Codex).
type Harness interface {
	// Identity returns information about this harness
	Name() string    // e.g., "claude-code", "codex", "gemini"
	Version() string // CLI version if available

	// Lifecycle manages the harness subprocess
	Start(ctx context.Context, config Config) error
	Stop() error
	IsRunning() bool

	// Communication with the agent
	SendPrompt(prompt string) error
	Events() <-chan Event

	// Capabilities reports what this harness supports
	Capabilities() Capabilities
}

// Capabilities describes what features a harness supports
type Capabilities struct {
	// SupportedModels lists available models for this harness
	SupportedModels []Model

	// SupportsHooks indicates if the harness supports pre/post tool hooks
	SupportsHooks bool

	// SupportsMCP indicates if the harness supports Model Context Protocol
	SupportsMCP bool

	// SupportsStreaming indicates if events are streamed in real-time
	SupportsStreaming bool

	// SupportsResume indicates if the harness can resume conversations
	SupportsResume bool
}

// Model represents an AI model available in a harness
type Model struct {
	ID          string // e.g., "opus", "sonnet", "gpt-5-codex"
	Name        string // Human-readable name
	Description string // Brief description
	Default     bool   // Whether this is the default model
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	Name    string            // Server name
	Command string            // Command to run
	Args    []string          // Command arguments
	Env     map[string]string // Environment variables
}

// HarnessFactory creates a new instance of a harness
type HarnessFactory func() Harness

// BaseHarness provides common functionality for harness implementations.
//
// # Thread Safety
//
// BaseHarness is designed for concurrent use with the following guarantees:
//
//   - IsRunning() and SetRunning() are protected by a mutex and safe to call
//     from multiple goroutines.
//   - Events() returns a receive-only channel that can be read by one consumer.
//     Multiple consumers reading from the same channel will each receive a
//     subset of events (fan-out behavior).
//   - EventsWritable() returns a send-only channel. Multiple goroutines may
//     send events concurrently, but implementers must ensure the channel is
//     not closed while sends are in progress.
//   - CloseEvents() should be called after all senders have stopped. It is
//     safe to call more than once.
//
// # Event Ordering
//
// Events are delivered in the order they are sent to the channel. There is no
// guaranteed ordering between events from different goroutines (e.g., stdout
// vs stderr readers). The events channel has a buffer of 100 events; if the
// consumer falls behind, senders will block.
//
// # Lifecycle
//
// A harness progresses through these states:
//  1. Created (NewBaseHarness) - not running, events channel open
//  2. Started (SetRunning(true)) - running, events flowing
//  3. Stopped (SetRunning(false), CloseEvents) - not running, events channel closed
//
// Once stopped, a BaseHarness cannot be restarted. Create a new instance instead.
type BaseHarness struct {
	mu        sync.RWMutex
	name      string
	version   string
	running   bool
	stopped   bool
	events    chan Event
	closeOnce sync.Once
	closed    bool
}

// NewBaseHarness creates a new base harness with the given name
func NewBaseHarness(name string) *BaseHarness {
	return &BaseHarness{
		name:   name,
		events: make(chan Event, 100),
	}
}

// Name returns the harness name
func (b *BaseHarness) Name() string {
	return b.name
}

// Version returns the harness version
func (b *BaseHarness) Version() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.version
}

// SetVersion sets the harness version
func (b *BaseHarness) SetVersion(version string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.version = version
}

// IsRunning returns whether the harness is running
func (b *BaseHarness) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// IsStopped returns whether the harness has been stopped.
// Once stopped, a harness cannot be restarted.
func (b *BaseHarness) IsStopped() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.stopped
}

// SetRunning sets the running state.
// Setting running to false marks the harness as stopped.
func (b *BaseHarness) SetRunning(running bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if running {
		if b.stopped {
			return
		}
		b.running = true
		return
	}
	b.running = false
	b.stopped = true
}

// Events returns the events channel
func (b *BaseHarness) Events() <-chan Event {
	return b.events
}

// EventsWritable returns the writable events channel for implementations
func (b *BaseHarness) EventsWritable() chan<- Event {
	return b.events
}

// EventsClosed reports whether the events channel has been closed.
func (b *BaseHarness) EventsClosed() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.closed
}

// CloseEvents closes the events channel
func (b *BaseHarness) CloseEvents() {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.closed = true
		close(b.events)
	})
}
