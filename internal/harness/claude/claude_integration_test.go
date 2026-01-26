package claude

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
)

// truncate returns s truncated to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// These tests require the Claude CLI to be installed and authenticated.
// Run with: bazel test --test_tag_filters=integration //internal/harness/claude:claude_test

func TestClaudeHarness_RealCLI_SimplePrompt(t *testing.T) {
	// Skip if claude CLI is not available
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found in PATH")
	}

	// Use a short temp directory to avoid ENAMETOOLONG errors
	// (Bazel sandbox paths are very long)
	workingDir, err := os.MkdirTemp("/tmp", "claude-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(workingDir)

	h := New().(*ClaudeHarness)

	config := harness.NewConfig(workingDir).
		WithModel("sonnet")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start the harness
	if err := h.Start(ctx, config); err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	// Send a simple prompt
	if err := h.SendPrompt("Say hello in exactly 3 words."); err != nil {
		t.Fatalf("Failed to send prompt: %v", err)
	}

	// Collect events with timeout
	var events []harness.Event
	eventTimeout := time.After(30 * time.Second)

	for {
		select {
		case event, ok := <-h.Events():
			if !ok {
				// Channel closed, we're done
				goto verify
			}
			events = append(events, event)
			// Log more details for debugging
			if event.Text != "" {
				t.Logf("Received event: type=%s, text=%q", event.Type, truncate(event.Text, 100))
			} else if event.Error != nil {
				t.Logf("Received event: type=%s, error=%v", event.Type, event.Error)
			} else {
				t.Logf("Received event: type=%s", event.Type)
			}

		case <-eventTimeout:
			t.Fatal("Timeout waiting for events")
		}
	}

verify:
	// Verify we received expected event types
	if len(events) == 0 {
		t.Fatal("No events received")
	}

	// Should have at least: turn_start, text, turn_complete
	var hasTurnStart, hasText, hasTurnComplete bool
	for _, e := range events {
		switch e.Type {
		case harness.EventTurnStart:
			hasTurnStart = true
		case harness.EventText:
			hasText = true
		case harness.EventTurnComplete:
			hasTurnComplete = true
		}
	}

	if !hasTurnStart {
		t.Error("Missing turn_start event")
	}
	if !hasText {
		t.Error("Missing text event")
	}
	if !hasTurnComplete {
		t.Error("Missing turn_complete event")
	}

	t.Logf("Received %d events total", len(events))
}

func TestClaudeHarness_RealCLI_EventTypes(t *testing.T) {
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude CLI not found in PATH")
	}

	// Use a short temp directory to avoid ENAMETOOLONG errors
	workingDir, err := os.MkdirTemp("/tmp", "claude-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(workingDir)

	h := New().(*ClaudeHarness)
	config := harness.NewConfig(workingDir).WithModel("sonnet")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := h.Start(ctx, config); err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	// Send prompt that should trigger a file read
	if err := h.SendPrompt("What is in the file parser.go? Just say 'It defines a Parser type' - don't actually read it."); err != nil {
		t.Fatalf("Failed to send prompt: %v", err)
	}

	// Collect events
	var events []harness.Event
	eventTimeout := time.After(30 * time.Second)

	for {
		select {
		case event, ok := <-h.Events():
			if !ok {
				goto done
			}
			events = append(events, event)
		case <-eventTimeout:
			t.Fatal("Timeout waiting for events")
		}
	}

done:
	// Log all event types received
	for i, e := range events {
		t.Logf("Event %d: type=%s, tool=%s", i, e.Type, e.Tool)
	}

	if len(events) < 2 {
		t.Errorf("Expected at least 2 events, got %d", len(events))
	}
}
