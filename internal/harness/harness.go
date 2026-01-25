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
	"io"
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

// BaseHarness provides common functionality for harness implementations
type BaseHarness struct {
	mu      sync.RWMutex
	name    string
	version string
	running bool
	events  chan Event
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
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
	return b.version
}

// SetVersion sets the harness version
func (b *BaseHarness) SetVersion(version string) {
	b.version = version
}

// IsRunning returns whether the harness is running
func (b *BaseHarness) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// SetRunning sets the running state
func (b *BaseHarness) SetRunning(running bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.running = running
}

// Events returns the events channel
func (b *BaseHarness) Events() <-chan Event {
	return b.events
}

// EventsWritable returns the writable events channel for implementations
func (b *BaseHarness) EventsWritable() chan<- Event {
	return b.events
}

// CloseEvents closes the events channel
func (b *BaseHarness) CloseEvents() {
	close(b.events)
}
