package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// EventType represents the type of Claude event
type EventType string

const (
	EventToolUse     EventType = "tool_use"
	EventToolResult  EventType = "tool_result"
	EventText        EventType = "text"
	EventFileRead    EventType = "file_read"
	EventFileWrite   EventType = "file_write"
	EventFileEdit    EventType = "file_edit"
	EventBuildRun    EventType = "build_run"
	EventTestRun     EventType = "test_run"
	EventSubagentRun EventType = "subagent_run"
)

// Event represents a Claude tool use or result event
type Event struct {
	Type      EventType
	Timestamp time.Time
	Data      map[string]interface{}
}

// EventHandler is called when a Claude event occurs
type EventHandler func(*Event)

// Interceptor intercepts Claude Code tool calls via JSON output
type Interceptor struct {
	mu       sync.RWMutex
	cmd      *exec.Cmd
	handlers []EventHandler
	running  bool
	events   chan *Event
}

// New creates a new Claude interceptor
func New() *Interceptor {
	return &Interceptor{
		handlers: make([]EventHandler, 0),
		events:   make(chan *Event, 100),
	}
}

// AddHandler registers an event handler
func (i *Interceptor) AddHandler(handler EventHandler) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.handlers = append(i.handlers, handler)
}

// Start begins intercepting Claude output
// For now, this is a stub that will be connected to actual Claude subprocess
func (i *Interceptor) Start() error {
	i.mu.Lock()
	if i.running {
		i.mu.Unlock()
		return fmt.Errorf("interceptor already running")
	}
	i.running = true
	i.mu.Unlock()

	// Start event dispatcher goroutine
	go i.dispatchEvents()

	// Future: Start Claude subprocess with --output-format json
	// cmd := exec.Command("claude", "--output-format", "json")
	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	//     return err
	// }
	// if err := cmd.Start(); err != nil {
	//     return err
	// }
	// i.cmd = cmd
	// go i.parseOutput(stdout)

	return nil
}

// Stop stops the interceptor
func (i *Interceptor) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.running {
		return nil
	}

	i.running = false
	close(i.events)

	if i.cmd != nil && i.cmd.Process != nil {
		return i.cmd.Process.Kill()
	}

	return nil
}

// dispatchEvents dispatches events to registered handlers
func (i *Interceptor) dispatchEvents() {
	for event := range i.events {
		i.mu.RLock()
		handlers := make([]EventHandler, len(i.handlers))
		copy(handlers, i.handlers)
		i.mu.RUnlock()

		// Call all handlers with this event
		for _, handler := range handlers {
			handler(event)
		}
	}
}

// parseOutput parses JSON output from Claude subprocess
func (i *Interceptor) parseOutput(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse JSON line
		var data map[string]interface{}
		if err := json.Unmarshal(line, &data); err != nil {
			continue // Skip malformed lines
		}

		// Determine event type and create event
		event := i.parseEvent(data)
		if event != nil {
			i.events <- event
		}
	}
}

// parseEvent converts raw JSON data to a typed Event
func (i *Interceptor) parseEvent(data map[string]interface{}) *Event {
	// Determine event type from JSON structure
	eventType := i.inferEventType(data)
	if eventType == "" {
		return nil
	}

	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// inferEventType infers the event type from JSON data
func (i *Interceptor) inferEventType(data map[string]interface{}) EventType {
	// Check for tool_use or tool_result in the data
	if toolName, ok := data["tool"].(string); ok {
		switch toolName {
		case "Read", "read", "read_file":
			return EventFileRead
		case "Write", "write", "write_file":
			return EventFileWrite
		case "Edit", "edit", "edit_file":
			return EventFileEdit
		case "Bash", "bash":
			// Check if it's a build or test command
			if command, ok := data["command"].(string); ok {
				if containsAny(command, []string{"build", "make", "cargo", "go build"}) {
					return EventBuildRun
				}
				if containsAny(command, []string{"test", "pytest", "go test", "npm test"}) {
					return EventTestRun
				}
			}
			return EventToolUse
		case "Task", "task":
			return EventSubagentRun
		default:
			return EventToolUse
		}
	}

	// Check for text content
	if _, ok := data["text"]; ok {
		return EventText
	}

	return ""
}

// containsAny checks if a string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	// Simple case-insensitive check
	sLower := toLower(s)
	substrLower := toLower(substr)
	return indexOf(sLower, substrLower) >= 0
}

// toLower converts a string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// indexOf finds the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// SimulateFileRead simulates a file read event (for testing)
func (i *Interceptor) SimulateFileRead(path string) {
	i.events <- &Event{
		Type:      EventFileRead,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool":      "Read",
			"file_path": path,
		},
	}
}

// SimulateFileWrite simulates a file write event (for testing)
func (i *Interceptor) SimulateFileWrite(path string) {
	i.events <- &Event{
		Type:      EventFileWrite,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool":      "Write",
			"file_path": path,
		},
	}
}

// SimulateFileEdit simulates a file edit event (for testing)
func (i *Interceptor) SimulateFileEdit(path string) {
	i.events <- &Event{
		Type:      EventFileEdit,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"tool":      "Edit",
			"file_path": path,
		},
	}
}
