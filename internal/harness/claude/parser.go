package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
)

// Parser parses Claude Code JSON output and emits harness events
type Parser struct {
	events chan<- harness.Event
	// DebugWriter, if set, receives debug output for parse failures and
	// unrecognized event types. Useful for troubleshooting Claude output changes.
	DebugWriter io.Writer
}

// NewParser creates a new Claude output parser
func NewParser(events chan<- harness.Event) *Parser {
	return &Parser{
		events: events,
	}
}

// SetDebugWriter sets the writer for debug output.
// Pass nil to disable debug logging.
func (p *Parser) SetDebugWriter(w io.Writer) {
	p.DebugWriter = w
}

// debugf writes a formatted debug message if DebugWriter is set.
func (p *Parser) debugf(format string, args ...interface{}) {
	if p.DebugWriter != nil {
		fmt.Fprintf(p.DebugWriter, "[parser] "+format+"\n", args...)
	}
}

// ParseLine parses a single line of JSON output
func (p *Parser) ParseLine(line []byte) {
	var raw map[string]interface{}
	if err := json.Unmarshal(line, &raw); err != nil {
		p.debugf("JSON parse error: %v (line: %s)", err, string(line))
		return
	}

	event := p.parseEvent(raw)
	if event != nil {
		p.events <- *event
	} else {
		p.debugf("unrecognized event structure: %s", string(line))
	}
}

// parseEvent converts raw JSON data to a harness Event
func (p *Parser) parseEvent(raw map[string]interface{}) *harness.Event {
	eventType := p.inferEventType(raw)
	if eventType == "" {
		return nil
	}

	event := harness.NewEvent(eventType).
		WithRawMap(raw).
		WithSource("claude-code").
		WithTimestamp(time.Now())

	// Extract tool information if present
	if toolName, ok := raw["tool"].(string); ok {
		event = event.WithTool(toolName)
	}

	// Extract tool input
	if input, ok := raw["input"].(map[string]interface{}); ok {
		event = event.WithToolInputMap(input)
	}

	// Handle different event types
	switch eventType {
	case harness.EventFileRead, harness.EventFileWrite, harness.EventFileEdit:
		p.extractFilePath(event, raw)

	case harness.EventBuildRun, harness.EventTestRun:
		p.extractCommand(event, raw)

	case harness.EventText:
		if text, ok := raw["text"].(string); ok {
			event = event.WithText(text)
		}
		if content, ok := raw["content"].(string); ok {
			event = event.WithText(content)
		}

	case harness.EventSubagentRun:
		p.extractSubagentInfo(event, raw)

	case harness.EventToolResult:
		if output, ok := raw["output"].(map[string]interface{}); ok {
			for k, v := range output {
				event = event.WithToolOutput(k, v)
			}
		}
		if result, ok := raw["result"].(map[string]interface{}); ok {
			for k, v := range result {
				event = event.WithToolOutput(k, v)
			}
		}
	}

	built := event.Build()
	return &built
}

// inferEventType determines the event type from JSON structure
func (p *Parser) inferEventType(raw map[string]interface{}) harness.EventType {
	// Check message type field (Claude Code JSON format)
	if msgType, ok := raw["type"].(string); ok {
		switch msgType {
		case "tool_use":
			return p.inferToolEventType(raw)
		case "tool_result":
			return harness.EventToolResult
		case "text", "content_block_delta", "content_block_start":
			return harness.EventText
		case "message_start":
			return harness.EventTurnStart
		case "message_stop", "message_delta":
			// Check if it's actually a turn complete
			if stopReason, ok := raw["stop_reason"].(string); ok && stopReason != "" {
				return harness.EventTurnComplete
			}
			return ""
		case "error":
			return harness.EventError
		}
	}

	// Fallback: check for tool field (alternate format)
	if toolName, ok := raw["tool"].(string); ok {
		return p.inferToolEventTypeFromName(toolName, raw)
	}

	// Check for content field
	if _, ok := raw["content"]; ok {
		return harness.EventText
	}

	// Check for text field
	if _, ok := raw["text"]; ok {
		return harness.EventText
	}

	return ""
}

// inferToolEventType determines the specific event type for a tool_use message
func (p *Parser) inferToolEventType(raw map[string]interface{}) harness.EventType {
	// Get tool name from nested structure
	var toolName string

	if name, ok := raw["name"].(string); ok {
		toolName = name
	} else if tool, ok := raw["tool"].(string); ok {
		toolName = tool
	}

	return p.inferToolEventTypeFromName(toolName, raw)
}

// inferToolEventTypeFromName maps tool names to event types
func (p *Parser) inferToolEventTypeFromName(toolName string, raw map[string]interface{}) harness.EventType {
	switch strings.ToLower(toolName) {
	case "read", "read_file":
		return harness.EventFileRead
	case "write", "write_file":
		return harness.EventFileWrite
	case "edit", "edit_file":
		return harness.EventFileEdit
	case "bash":
		return p.inferBashEventType(raw)
	case "task":
		return harness.EventSubagentRun
	default:
		return harness.EventToolUse
	}
}

// inferBashEventType determines if a Bash command is a build or test
func (p *Parser) inferBashEventType(raw map[string]interface{}) harness.EventType {
	var command string

	// Try input.command
	if input, ok := raw["input"].(map[string]interface{}); ok {
		if cmd, ok := input["command"].(string); ok {
			command = cmd
		}
	}

	// Try direct command field
	if cmd, ok := raw["command"].(string); ok {
		command = cmd
	}

	if command == "" {
		return harness.EventToolUse
	}

	cmdLower := strings.ToLower(command)
	words := strings.Fields(cmdLower)
	if len(words) == 0 {
		return harness.EventToolUse
	}

	firstWord := words[0]

	// Build tools that use "build" subcommand
	buildSubcommandTools := map[string]bool{
		"go": true, "cargo": true, "npm": true, "bazel": true,
		"gradle": true, "mvn": true, "dotnet": true,
	}

	// Standalone build tools (first word is the tool)
	standaloneBuildTools := map[string]bool{
		"make": true, "cmake": true, "msbuild": true, "ninja": true,
	}

	// Test tools that use "test" subcommand
	testSubcommandTools := map[string]bool{
		"go": true, "cargo": true, "npm": true, "bazel": true,
		"gradle": true, "mvn": true, "dotnet": true,
	}

	// Standalone test tools (first word is the tool)
	standaloneTestTools := map[string]bool{
		"pytest": true, "jest": true, "mocha": true, "vitest": true,
		"rspec": true, "phpunit": true,
	}

	// Check for standalone test tools first
	if standaloneTestTools[firstWord] {
		return harness.EventTestRun
	}

	// Check for standalone build tools
	if standaloneBuildTools[firstWord] {
		return harness.EventBuildRun
	}

	// Check for tools with subcommands
	if len(words) >= 2 {
		secondWord := words[1]

		// Check for test subcommand
		if testSubcommandTools[firstWord] && secondWord == "test" {
			return harness.EventTestRun
		}

		// Check for build subcommand (including "run build" for npm)
		if buildSubcommandTools[firstWord] {
			if secondWord == "build" {
				return harness.EventBuildRun
			}
			// npm run build
			if firstWord == "npm" && secondWord == "run" && len(words) >= 3 && words[2] == "build" {
				return harness.EventBuildRun
			}
		}
	}

	return harness.EventToolUse
}

// extractFilePath extracts file path from various JSON structures
func (p *Parser) extractFilePath(event *harness.EventBuilder, raw map[string]interface{}) {
	// Try input.file_path
	if input, ok := raw["input"].(map[string]interface{}); ok {
		if path, ok := input["file_path"].(string); ok {
			event.WithToolInput("file_path", path)
			return
		}
		if path, ok := input["path"].(string); ok {
			event.WithToolInput("file_path", path)
			return
		}
	}

	// Try direct file_path
	if path, ok := raw["file_path"].(string); ok {
		event.WithToolInput("file_path", path)
		return
	}

	// Try path
	if path, ok := raw["path"].(string); ok {
		event.WithToolInput("file_path", path)
		return
	}
}

// extractCommand extracts command from bash events
func (p *Parser) extractCommand(event *harness.EventBuilder, raw map[string]interface{}) {
	// Try input.command
	if input, ok := raw["input"].(map[string]interface{}); ok {
		if cmd, ok := input["command"].(string); ok {
			event.WithToolInput("command", cmd)
			return
		}
	}

	// Try direct command
	if cmd, ok := raw["command"].(string); ok {
		event.WithToolInput("command", cmd)
	}
}

// extractSubagentInfo extracts subagent/task information
func (p *Parser) extractSubagentInfo(event *harness.EventBuilder, raw map[string]interface{}) {
	// Extract task description
	if input, ok := raw["input"].(map[string]interface{}); ok {
		if desc, ok := input["description"].(string); ok {
			event.WithToolInput("description", desc)
		}
		if prompt, ok := input["prompt"].(string); ok {
			event.WithToolInput("prompt", prompt)
		}
		if agentType, ok := input["subagent_type"].(string); ok {
			event.WithToolInput("subagent_type", agentType)
		}
	}
}

// ParseJSON parses a complete JSON message (not line-delimited)
func (p *Parser) ParseJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	event := p.parseEvent(raw)
	if event != nil {
		p.events <- *event
	}
	return nil
}

// ParseToolUse creates a tool use event from explicit parameters
func (p *Parser) ParseToolUse(toolName string, input map[string]interface{}) {
	eventType := p.inferToolEventTypeFromName(toolName, map[string]interface{}{"input": input})

	event := harness.NewEvent(eventType).
		WithTool(toolName).
		WithToolInputMap(input).
		WithSource("claude-code").
		Build()

	p.events <- event
}

// ParseToolResult creates a tool result event
func (p *Parser) ParseToolResult(toolName string, output map[string]interface{}) {
	builder := harness.NewEvent(harness.EventToolResult).
		WithTool(toolName).
		WithSource("claude-code")

	for k, v := range output {
		builder = builder.WithToolOutput(k, v)
	}

	p.events <- builder.Build()
}
