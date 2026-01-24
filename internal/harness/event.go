package harness

import "time"

// EventType represents the type of harness event
type EventType string

const (
	// Tool events
	EventToolUse    EventType = "tool_use"    // Agent is using a tool
	EventToolResult EventType = "tool_result" // Tool returned a result

	// Text events
	EventText         EventType = "text"          // Agent produced text output
	EventTurnStart    EventType = "turn_start"    // New turn/response started
	EventTurnComplete EventType = "turn_complete" // Turn/response completed

	// Derived events (inferred from tool use)
	EventFileRead    EventType = "file_read"    // File was read
	EventFileWrite   EventType = "file_write"   // File was written
	EventFileEdit    EventType = "file_edit"    // File was edited
	EventBuildRun    EventType = "build_run"    // Build command executed
	EventTestRun     EventType = "test_run"     // Test command executed
	EventSubagentRun EventType = "subagent_run" // Subagent/task spawned

	// Status events
	EventError   EventType = "error"   // An error occurred
	EventWarning EventType = "warning" // A warning was generated
)

// Event represents a unified event from any harness
type Event struct {
	// Type is the event type
	Type EventType

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Tool information (for tool events)
	Tool       string                 // Tool name (Read, Write, Bash, etc.)
	ToolInput  map[string]interface{} // Tool input parameters
	ToolOutput map[string]interface{} // Tool result (for tool_result events)

	// Text content (for text events)
	Text string

	// Error information (for error events)
	Error error

	// Raw data from the harness (for debugging/extension)
	Raw map[string]interface{}

	// Source identifies which harness produced this event
	Source string
}

// IsFileEvent returns true if this event relates to a file operation
func (e *Event) IsFileEvent() bool {
	return e.Type == EventFileRead || e.Type == EventFileWrite || e.Type == EventFileEdit
}

// IsFileRead returns true if this is a file read event
func (e *Event) IsFileRead() bool {
	return e.Type == EventFileRead
}

// IsFileWrite returns true if this is a file write event
func (e *Event) IsFileWrite() bool {
	return e.Type == EventFileWrite
}

// IsFileEdit returns true if this is a file edit event
func (e *Event) IsFileEdit() bool {
	return e.Type == EventFileEdit
}

// IsBuildOrTest returns true if this is a build or test event
func (e *Event) IsBuildOrTest() bool {
	return e.Type == EventBuildRun || e.Type == EventTestRun
}

// FilePath extracts the file path from a file event
func (e *Event) FilePath() string {
	// Check tool input first
	if path, ok := e.ToolInput["file_path"].(string); ok {
		return path
	}
	if path, ok := e.ToolInput["path"].(string); ok {
		return path
	}
	// Check raw data as fallback
	if path, ok := e.Raw["file_path"].(string); ok {
		return path
	}
	if path, ok := e.Raw["path"].(string); ok {
		return path
	}
	return ""
}

// Command extracts the command from a bash/build/test event
func (e *Event) Command() string {
	if cmd, ok := e.ToolInput["command"].(string); ok {
		return cmd
	}
	if cmd, ok := e.Raw["command"].(string); ok {
		return cmd
	}
	return ""
}

// EventBuilder provides a fluent interface for creating events
type EventBuilder struct {
	event Event
}

// NewEvent creates a new event builder
func NewEvent(eventType EventType) *EventBuilder {
	return &EventBuilder{
		event: Event{
			Type:       eventType,
			Timestamp:  time.Now(),
			ToolInput:  make(map[string]interface{}),
			ToolOutput: make(map[string]interface{}),
			Raw:        make(map[string]interface{}),
		},
	}
}

// WithTool sets the tool name
func (b *EventBuilder) WithTool(tool string) *EventBuilder {
	b.event.Tool = tool
	return b
}

// WithToolInput sets a tool input parameter
func (b *EventBuilder) WithToolInput(key string, value interface{}) *EventBuilder {
	b.event.ToolInput[key] = value
	return b
}

// WithToolInputMap sets all tool input parameters
func (b *EventBuilder) WithToolInputMap(input map[string]interface{}) *EventBuilder {
	for k, v := range input {
		b.event.ToolInput[k] = v
	}
	return b
}

// WithToolOutput sets a tool output parameter
func (b *EventBuilder) WithToolOutput(key string, value interface{}) *EventBuilder {
	b.event.ToolOutput[key] = value
	return b
}

// WithText sets the text content
func (b *EventBuilder) WithText(text string) *EventBuilder {
	b.event.Text = text
	return b
}

// WithError sets the error
func (b *EventBuilder) WithError(err error) *EventBuilder {
	b.event.Error = err
	return b
}

// WithRaw sets a raw data value
func (b *EventBuilder) WithRaw(key string, value interface{}) *EventBuilder {
	b.event.Raw[key] = value
	return b
}

// WithRawMap sets all raw data
func (b *EventBuilder) WithRawMap(raw map[string]interface{}) *EventBuilder {
	for k, v := range raw {
		b.event.Raw[k] = v
	}
	return b
}

// WithSource sets the source harness
func (b *EventBuilder) WithSource(source string) *EventBuilder {
	b.event.Source = source
	return b
}

// WithTimestamp sets a specific timestamp
func (b *EventBuilder) WithTimestamp(t time.Time) *EventBuilder {
	b.event.Timestamp = t
	return b
}

// Build returns the constructed event
func (b *EventBuilder) Build() Event {
	return b.event
}
