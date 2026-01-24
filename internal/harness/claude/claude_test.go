package claude

import (
	"os"
	"testing"

	"github.com/tedks/CodingGame/internal/harness"
)

func TestNew(t *testing.T) {
	h := New()

	if h.Name() != "claude-code" {
		t.Errorf("Name() = %q, want claude-code", h.Name())
	}
	if h.IsRunning() {
		t.Error("IsRunning() should be false initially")
	}
}

func TestNewHarness(t *testing.T) {
	// Verify it returns a Harness interface
	var h harness.Harness = NewHarness()
	if h.Name() != "claude-code" {
		t.Errorf("Name() = %q, want claude-code", h.Name())
	}
}

func TestCapabilities(t *testing.T) {
	h := New()
	caps := h.Capabilities()

	if len(caps.SupportedModels) == 0 {
		t.Error("Should have supported models")
	}

	// Verify models include opus, sonnet, haiku
	modelIDs := make(map[string]bool)
	for _, m := range caps.SupportedModels {
		modelIDs[m.ID] = true
	}

	if !modelIDs["opus"] {
		t.Error("Should support opus model")
	}
	if !modelIDs["sonnet"] {
		t.Error("Should support sonnet model")
	}
	if !modelIDs["haiku"] {
		t.Error("Should support haiku model")
	}

	if !caps.SupportsHooks {
		t.Error("Should support hooks")
	}
	if !caps.SupportsMCP {
		t.Error("Should support MCP")
	}
	if !caps.SupportsStreaming {
		t.Error("Should support streaming")
	}
	if !caps.SupportsResume {
		t.Error("Should support resume")
	}
}

func TestBuildArgs(t *testing.T) {
	h := New().(*ClaudeHarness)

	tests := []struct {
		name     string
		config   harness.Config
		contains []string
	}{
		{
			name:   "default",
			config: harness.NewConfig("/tmp"),
			contains: []string{
				"--output-format", "json",
				"--print",
			},
		},
		{
			name:   "with model",
			config: harness.NewConfig("/tmp").WithModel("opus"),
			contains: []string{
				"--model", "opus",
			},
		},
		{
			name:   "with system prompt",
			config: harness.NewConfig("/tmp").WithSystemPrompt("You are a security expert"),
			contains: []string{
				"--system-prompt", "You are a security expert",
			},
		},
		{
			name: "with max tokens",
			config: harness.Config{
				WorkingDir: "/tmp",
				MaxTokens:  4096,
			},
			contains: []string{
				"--max-tokens", "4096",
			},
		},
		{
			name: "with verbose",
			config: harness.Config{
				WorkingDir: "/tmp",
				Verbose:    true,
			},
			contains: []string{
				"--verbose",
			},
		},
		{
			name: "with resume",
			config: harness.Config{
				WorkingDir:    "/tmp",
				ResumeSession: "abc123",
			},
			contains: []string{
				"--resume", "abc123",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := h.buildArgs(tc.config)

			for i := 0; i < len(tc.contains); i++ {
				found := false
				for _, arg := range args {
					if arg == tc.contains[i] {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Args should contain %q, got %v", tc.contains[i], args)
				}
			}
		})
	}
}

func TestSimulateEvents(t *testing.T) {
	h := New().(*ClaudeHarness)

	// Test SimulateFileRead
	go func() {
		h.SimulateFileRead("/test/file.go")
	}()

	event := <-h.Events()
	if event.Type != harness.EventFileRead {
		t.Errorf("Type = %v, want EventFileRead", event.Type)
	}
	if event.FilePath() != "/test/file.go" {
		t.Errorf("FilePath() = %q, want /test/file.go", event.FilePath())
	}

	// Test SimulateFileWrite
	go func() {
		h.SimulateFileWrite("/test/output.go")
	}()

	event = <-h.Events()
	if event.Type != harness.EventFileWrite {
		t.Errorf("Type = %v, want EventFileWrite", event.Type)
	}

	// Test SimulateFileEdit
	go func() {
		h.SimulateFileEdit("/test/edit.go")
	}()

	event = <-h.Events()
	if event.Type != harness.EventFileEdit {
		t.Errorf("Type = %v, want EventFileEdit", event.Type)
	}
}

func TestSimulateEvent(t *testing.T) {
	h := New().(*ClaudeHarness)

	testEvent := harness.NewEvent(harness.EventBuildRun).
		WithTool("Bash").
		WithToolInput("command", "make build").
		Build()

	go func() {
		h.SimulateEvent(testEvent)
	}()

	event := <-h.Events()
	if event.Type != harness.EventBuildRun {
		t.Errorf("Type = %v, want EventBuildRun", event.Type)
	}
	if event.Command() != "make build" {
		t.Errorf("Command() = %q, want 'make build'", event.Command())
	}
}

func TestSendPromptNotRunning(t *testing.T) {
	h := New().(*ClaudeHarness)

	err := h.SendPrompt("test prompt")
	if err == nil {
		t.Error("SendPrompt should return error when not running")
	}
}

func TestStopNotRunning(t *testing.T) {
	h := New().(*ClaudeHarness)

	err := h.Stop()
	if err != nil {
		t.Errorf("Stop() should not error when not running: %v", err)
	}
}

func TestStartInvalidConfig(t *testing.T) {
	h := New().(*ClaudeHarness)

	// Empty working dir should fail validation
	config := harness.Config{}
	err := h.Start(nil, config)
	if err == nil {
		t.Error("Start should fail with invalid config")
	}
}

func TestStartNonExistentDir(t *testing.T) {
	h := New().(*ClaudeHarness)

	config := harness.NewConfig("/nonexistent/path/that/does/not/exist")
	err := h.Start(nil, config)
	if err == nil {
		t.Error("Start should fail with non-existent directory")
	}
}

// Integration test - only runs if claude CLI is installed
func TestStartStopIntegration(t *testing.T) {
	// Skip if claude is not installed
	if _, err := os.Stat("/usr/bin/claude"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/local/bin/claude"); os.IsNotExist(err) {
			t.Skip("claude CLI not installed")
		}
	}

	// Also skip in CI environments where claude may not be configured
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI")
	}

	h := New().(*ClaudeHarness)

	tmpDir, err := os.MkdirTemp("", "claude-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := harness.NewConfig(tmpDir).WithModel("sonnet")

	// This will fail if claude isn't authenticated, which is fine for testing
	// We're mainly testing that the subprocess management works
	_ = h.Start(nil, config)

	// Stop should work regardless of whether start succeeded
	err = h.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}
