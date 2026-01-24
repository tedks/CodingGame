package claude

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
)

func TestParserFileRead(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "tool_use",
		"name": "Read",
		"input": map[string]interface{}{
			"file_path": "/path/to/file.go",
		},
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventFileRead {
			t.Errorf("Type = %v, want EventFileRead", event.Type)
		}
		if event.FilePath() != "/path/to/file.go" {
			t.Errorf("FilePath() = %q, want /path/to/file.go", event.FilePath())
		}
		if event.Source != "claude-code" {
			t.Errorf("Source = %q, want claude-code", event.Source)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserFileWrite(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "tool_use",
		"name": "Write",
		"input": map[string]interface{}{
			"file_path": "/path/to/output.go",
			"content":   "package main",
		},
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventFileWrite {
			t.Errorf("Type = %v, want EventFileWrite", event.Type)
		}
		if event.FilePath() != "/path/to/output.go" {
			t.Errorf("FilePath() = %q, want /path/to/output.go", event.FilePath())
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserFileEdit(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "tool_use",
		"name": "Edit",
		"input": map[string]interface{}{
			"file_path":  "/path/to/edit.go",
			"old_string": "foo",
			"new_string": "bar",
		},
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventFileEdit {
			t.Errorf("Type = %v, want EventFileEdit", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserBashBuild(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "tool_use",
		"name": "Bash",
		"input": map[string]interface{}{
			"command": "bazel build //...",
		},
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventBuildRun {
			t.Errorf("Type = %v, want EventBuildRun", event.Type)
		}
		if event.Command() != "bazel build //..." {
			t.Errorf("Command() = %q, want 'bazel build //...'", event.Command())
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserBashTest(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	testCases := []struct {
		name    string
		command string
	}{
		{"go test", "go test ./..."},
		{"pytest", "pytest tests/"},
		{"npm test", "npm test"},
		{"bazel test", "bazel test //..."},
		{"cargo test", "cargo test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := map[string]interface{}{
				"type": "tool_use",
				"name": "Bash",
				"input": map[string]interface{}{
					"command": tc.command,
				},
			}
			data, _ := json.Marshal(input)
			parser.ParseLine(data)

			select {
			case event := <-events:
				if event.Type != harness.EventTestRun {
					t.Errorf("Type = %v, want EventTestRun for command %q", event.Type, tc.command)
				}
			case <-time.After(time.Second):
				t.Error("Timeout waiting for event")
			}
		})
	}
}

func TestParserTextEvent(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "text",
		"text": "Hello, I'm Claude.",
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventText {
			t.Errorf("Type = %v, want EventText", event.Type)
		}
		if event.Text != "Hello, I'm Claude." {
			t.Errorf("Text = %q, want 'Hello, I'm Claude.'", event.Text)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserToolResult(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "tool_result",
		"tool": "Read",
		"output": map[string]interface{}{
			"content": "package main",
		},
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventToolResult {
			t.Errorf("Type = %v, want EventToolResult", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserSubagentRun(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "tool_use",
		"name": "Task",
		"input": map[string]interface{}{
			"description":   "Analyze code",
			"prompt":        "Review this file for security issues",
			"subagent_type": "Explore",
		},
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventSubagentRun {
			t.Errorf("Type = %v, want EventSubagentRun", event.Type)
		}
		if event.ToolInput["description"] != "Analyze code" {
			t.Errorf("description = %v, want 'Analyze code'", event.ToolInput["description"])
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserTurnStart(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type": "message_start",
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventTurnStart {
			t.Errorf("Type = %v, want EventTurnStart", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserTurnComplete(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	input := map[string]interface{}{
		"type":        "message_stop",
		"stop_reason": "end_turn",
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventTurnComplete {
			t.Errorf("Type = %v, want EventTurnComplete", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParserInvalidJSON(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	// Invalid JSON should be silently skipped
	parser.ParseLine([]byte("not valid json"))

	select {
	case <-events:
		t.Error("Should not emit event for invalid JSON")
	case <-time.After(100 * time.Millisecond):
		// Expected: no event
	}
}

func TestParserUnknownEventType(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	// Unknown structure should not produce event
	input := map[string]interface{}{
		"unknown_field": "value",
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case <-events:
		t.Error("Should not emit event for unknown structure")
	case <-time.After(100 * time.Millisecond):
		// Expected: no event
	}
}

func TestParserAlternateToolFormat(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	// Test alternate format with "tool" instead of "name"
	input := map[string]interface{}{
		"tool":      "Read",
		"file_path": "/direct/path.go",
	}
	data, _ := json.Marshal(input)
	parser.ParseLine(data)

	select {
	case event := <-events:
		if event.Type != harness.EventFileRead {
			t.Errorf("Type = %v, want EventFileRead", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParseToolUse(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	parser.ParseToolUse("Read", map[string]interface{}{
		"file_path": "/test/file.go",
	})

	select {
	case event := <-events:
		if event.Type != harness.EventFileRead {
			t.Errorf("Type = %v, want EventFileRead", event.Type)
		}
		if event.Tool != "Read" {
			t.Errorf("Tool = %q, want Read", event.Tool)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

func TestParseToolResult(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	parser.ParseToolResult("Read", map[string]interface{}{
		"content": "file content",
	})

	select {
	case event := <-events:
		if event.Type != harness.EventToolResult {
			t.Errorf("Type = %v, want EventToolResult", event.Type)
		}
		if event.Tool != "Read" {
			t.Errorf("Tool = %q, want Read", event.Tool)
		}
		if event.ToolOutput["content"] != "file content" {
			t.Errorf("ToolOutput[content] = %v, want 'file content'", event.ToolOutput["content"])
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}
