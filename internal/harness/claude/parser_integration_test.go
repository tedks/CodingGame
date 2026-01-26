package claude

import (
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
)

// These tests verify the parser handles Claude's actual stream-json output format.
// The JSON samples are captured from real Claude CLI responses.

// Sample Claude stream-json output for a simple "Say hi" prompt
var claudeStreamJSONSamples = []struct {
	name         string
	json         string
	expectedType harness.EventType
}{
	{
		name: "system_init",
		json: `{"type":"system","subtype":"init","cwd":"/home/user/project","session_id":"abc-123","tools":["Task","Bash","Read"],"mcp_servers":[],"model":"claude-sonnet-4-5-20250929","permissionMode":"bypassPermissions","slash_commands":["compact"],"apiKeySource":"none","claude_code_version":"2.1.19","output_style":"default","agents":["Bash"],"skills":[],"plugins":[],"uuid":"xyz-456"}`,
		expectedType: harness.EventTurnStart,
	},
	{
		name: "assistant_text_response",
		json: `{"type":"assistant","message":{"model":"claude-sonnet-4-5-20250929","id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"Hello! How can I help you today?"}],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":2,"output_tokens":10}},"parent_tool_use_id":null,"session_id":"abc-123","uuid":"def-789"}`,
		expectedType: harness.EventText,
	},
	{
		name: "result_success",
		json: `{"type":"result","subtype":"success","is_error":false,"duration_ms":1500,"duration_api_ms":1400,"num_turns":1,"result":"Hello! How can I help you today?","session_id":"abc-123","total_cost_usd":0.001,"usage":{"input_tokens":2,"output_tokens":10},"modelUsage":{},"permission_denials":[],"uuid":"ghi-012"}`,
		expectedType: harness.EventTurnComplete,
	},
}

func TestParserClaudeStreamJSON(t *testing.T) {
	for _, tc := range claudeStreamJSONSamples {
		t.Run(tc.name, func(t *testing.T) {
			events := make(chan harness.Event, 10)
			parser := NewParser(events)

			parser.ParseLine([]byte(tc.json))

			select {
			case event := <-events:
				if event.Type != tc.expectedType {
					t.Errorf("Type = %v, want %v", event.Type, tc.expectedType)
				}
			case <-time.After(time.Second):
				t.Errorf("Timeout waiting for event from %s", tc.name)
			}
		})
	}
}

// Test that assistant message with tool_use content is correctly identified
func TestParserAssistantWithToolUse(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	// Claude's actual format for tool use within an assistant message
	json := `{"type":"assistant","message":{"model":"claude-sonnet-4-5-20250929","id":"msg_123","type":"message","role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Read","input":{"file_path":"/path/to/file.go"}}],"stop_reason":null},"session_id":"abc-123","uuid":"def-789"}`

	parser.ParseLine([]byte(json))

	select {
	case event := <-events:
		if event.Type != harness.EventFileRead {
			t.Errorf("Type = %v, want EventFileRead", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// Test that assistant message with end_turn stop_reason still emits text event
// (turn_complete comes from the separate "result" message)
func TestParserAssistantEndTurn(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	json := `{"type":"assistant","message":{"model":"claude-sonnet-4-5-20250929","id":"msg_123","type":"message","role":"assistant","content":[{"type":"text","text":"Done!"}],"stop_reason":"end_turn"},"session_id":"abc-123","uuid":"def-789"}`

	parser.ParseLine([]byte(json))

	select {
	case event := <-events:
		// Assistant messages always emit EventText, even with stop_reason
		// The turn_complete comes from the separate "result" message
		if event.Type != harness.EventText {
			t.Errorf("Type = %v, want EventText", event.Type)
		}
		if event.Text != "Done!" {
			t.Errorf("Text = %q, want 'Done!'", event.Text)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for event")
	}
}

// Test parsing multiple lines as would happen in real streaming
func TestParserMultipleStreamLines(t *testing.T) {
	events := make(chan harness.Event, 10)
	parser := NewParser(events)

	// Simulate the three lines Claude outputs for a simple response
	lines := []string{
		`{"type":"system","subtype":"init","cwd":"/tmp","session_id":"s1","tools":[],"mcp_servers":[],"model":"sonnet","uuid":"u1"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Hi!"}],"stop_reason":null},"session_id":"s1","uuid":"u2"}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"Hi!","session_id":"s1","uuid":"u3"}`,
	}

	for _, line := range lines {
		parser.ParseLine([]byte(line))
	}

	// Should get 3 events
	expectedTypes := []harness.EventType{
		harness.EventTurnStart,
		harness.EventText,
		harness.EventTurnComplete,
	}

	for i, expectedType := range expectedTypes {
		select {
		case event := <-events:
			if event.Type != expectedType {
				t.Errorf("Event %d: Type = %v, want %v", i, event.Type, expectedType)
			}
		case <-time.After(time.Second):
			t.Errorf("Timeout waiting for event %d", i)
		}
	}
}
