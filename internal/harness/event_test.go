package harness

import (
	"errors"
	"testing"
	"time"
)

func TestEventTypes(t *testing.T) {
	// Verify all event types are distinct
	types := []EventType{
		EventToolUse, EventToolResult, EventText, EventTurnStart, EventTurnComplete,
		EventFileRead, EventFileWrite, EventFileEdit, EventBuildRun, EventTestRun,
		EventSubagentRun, EventError, EventWarning,
	}

	seen := make(map[EventType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("duplicate event type: %s", et)
		}
		seen[et] = true
	}
}

func TestEventIsFileEvent(t *testing.T) {
	tests := []struct {
		eventType EventType
		isFile    bool
	}{
		{EventFileRead, true},
		{EventFileWrite, true},
		{EventFileEdit, true},
		{EventToolUse, false},
		{EventText, false},
		{EventBuildRun, false},
	}

	for _, tt := range tests {
		e := Event{Type: tt.eventType}
		if e.IsFileEvent() != tt.isFile {
			t.Errorf("Event{Type: %s}.IsFileEvent() = %v, want %v", tt.eventType, e.IsFileEvent(), tt.isFile)
		}
	}
}

func TestEventFilePath(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "file_path in ToolInput",
			event: Event{
				ToolInput: map[string]interface{}{"file_path": "/path/to/file.go"},
			},
			expected: "/path/to/file.go",
		},
		{
			name: "path in ToolInput",
			event: Event{
				ToolInput: map[string]interface{}{"path": "/another/path.go"},
			},
			expected: "/another/path.go",
		},
		{
			name: "file_path in Raw",
			event: Event{
				ToolInput: map[string]interface{}{},
				Raw:       map[string]interface{}{"file_path": "/raw/path.go"},
			},
			expected: "/raw/path.go",
		},
		{
			name: "no path",
			event: Event{
				ToolInput: map[string]interface{}{},
				Raw:       map[string]interface{}{},
			},
			expected: "",
		},
		{
			name:     "nil maps",
			event:    Event{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.FilePath(); got != tt.expected {
				t.Errorf("FilePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEventCommand(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		{
			name: "command in ToolInput",
			event: Event{
				ToolInput: map[string]interface{}{"command": "go test ./..."},
			},
			expected: "go test ./...",
		},
		{
			name: "command in Raw",
			event: Event{
				ToolInput: map[string]interface{}{},
				Raw:       map[string]interface{}{"command": "bazel build //..."},
			},
			expected: "bazel build //...",
		},
		{
			name:     "no command",
			event:    Event{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.Command(); got != tt.expected {
				t.Errorf("Command() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEventBuilder(t *testing.T) {
	testErr := errors.New("test error")
	now := time.Now()

	event := NewEvent(EventFileRead).
		WithTool("Read").
		WithToolInput("file_path", "/test/file.go").
		WithToolOutput("content", "package main").
		WithText("Reading file").
		WithError(testErr).
		WithRaw("custom", "value").
		WithSource("claude-code").
		WithTimestamp(now).
		Build()

	if event.Type != EventFileRead {
		t.Errorf("Type = %v, want %v", event.Type, EventFileRead)
	}
	if event.Tool != "Read" {
		t.Errorf("Tool = %v, want Read", event.Tool)
	}
	if event.ToolInput["file_path"] != "/test/file.go" {
		t.Errorf("ToolInput[file_path] = %v, want /test/file.go", event.ToolInput["file_path"])
	}
	if event.ToolOutput["content"] != "package main" {
		t.Errorf("ToolOutput[content] = %v, want 'package main'", event.ToolOutput["content"])
	}
	if event.Text != "Reading file" {
		t.Errorf("Text = %v, want 'Reading file'", event.Text)
	}
	if event.Error != testErr {
		t.Errorf("Error = %v, want %v", event.Error, testErr)
	}
	if event.Raw["custom"] != "value" {
		t.Errorf("Raw[custom] = %v, want value", event.Raw["custom"])
	}
	if event.Source != "claude-code" {
		t.Errorf("Source = %v, want claude-code", event.Source)
	}
	if !event.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", event.Timestamp, now)
	}
}

func TestEventBuilderWithMaps(t *testing.T) {
	input := map[string]interface{}{
		"file_path": "/test.go",
		"offset":    100,
	}
	raw := map[string]interface{}{
		"tool_id":  "abc123",
		"sequence": 1,
	}

	event := NewEvent(EventToolUse).
		WithToolInputMap(input).
		WithRawMap(raw).
		Build()

	if event.ToolInput["file_path"] != "/test.go" {
		t.Errorf("ToolInput[file_path] = %v, want /test.go", event.ToolInput["file_path"])
	}
	if event.ToolInput["offset"] != 100 {
		t.Errorf("ToolInput[offset] = %v, want 100", event.ToolInput["offset"])
	}
	if event.Raw["tool_id"] != "abc123" {
		t.Errorf("Raw[tool_id] = %v, want abc123", event.Raw["tool_id"])
	}
	if event.Raw["sequence"] != 1 {
		t.Errorf("Raw[sequence] = %v, want 1", event.Raw["sequence"])
	}
}

func TestEventIsBuildOrTest(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  bool
	}{
		{EventBuildRun, true},
		{EventTestRun, true},
		{EventToolUse, false},
		{EventFileRead, false},
	}

	for _, tt := range tests {
		e := Event{Type: tt.eventType}
		if got := e.IsBuildOrTest(); got != tt.expected {
			t.Errorf("Event{Type: %s}.IsBuildOrTest() = %v, want %v", tt.eventType, got, tt.expected)
		}
	}
}
