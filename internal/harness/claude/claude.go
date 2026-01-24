// Package claude provides a harness implementation for Claude Code CLI.
// It spawns Claude Code as a subprocess, parses its JSON output, and
// translates events into the unified harness event format.
package claude

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
)

// ClaudeHarness implements the Harness interface for Claude Code CLI
type ClaudeHarness struct {
	*harness.BaseHarness

	mu         sync.Mutex
	wg         sync.WaitGroup
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	cancelFunc context.CancelFunc
	done       chan struct{}
	parser     *Parser
}

// New creates a new Claude Code harness
func New() harness.Harness {
	return &ClaudeHarness{
		BaseHarness: harness.NewBaseHarness("claude-code"),
	}
}

// NewHarness is a factory function that returns a Harness interface
func NewHarness() harness.Harness {
	return New()
}

// Start begins the Claude Code subprocess
func (c *ClaudeHarness) Start(ctx context.Context, config harness.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.IsRunning() {
		return fmt.Errorf("harness already running")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Handle nil context
	if ctx == nil {
		ctx = context.Background()
	}

	// Create cancellable context
	ctx, c.cancelFunc = context.WithCancel(ctx)

	// Initialize done channel for goroutine signaling
	c.done = make(chan struct{})

	// Build command arguments
	args := c.buildArgs(config)

	// Create command
	c.cmd = exec.CommandContext(ctx, "claude", args...)
	c.cmd.Dir = config.WorkingDir

	// Set environment
	c.cmd.Env = os.Environ()
	for k, v := range config.Env {
		c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up pipes
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("creating stderr pipe: %w", err)
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("starting claude: %w", err)
	}

	// Get version from claude --version if not already set
	if c.Version() == "" {
		c.detectVersion()
	}

	// Create parser and start reading output
	c.parser = NewParser(c.EventsWritable())
	c.wg.Add(2)
	go c.readOutput()
	go c.readErrors()

	c.SetRunning(true)
	return nil
}

// buildArgs constructs the command line arguments for Claude
func (c *ClaudeHarness) buildArgs(config harness.Config) []string {
	args := []string{}

	// Always use JSON output format for parsing
	args = append(args, "--output-format", "json")

	// Model selection
	if config.Model != "" {
		args = append(args, "--model", config.Model)
	}

	// System prompt
	if config.SystemPrompt != "" {
		args = append(args, "--system-prompt", config.SystemPrompt)
	}

	// Max tokens
	if config.MaxTokens > 0 {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", config.MaxTokens))
	}

	// Verbose mode
	if config.Verbose {
		args = append(args, "--verbose")
	}

	// Resume session
	if config.ResumeSession != "" {
		args = append(args, "--resume", config.ResumeSession)
	}

	// Print mode for non-interactive use
	args = append(args, "--print")

	return args
}

// readOutput reads and parses stdout from Claude
func (c *ClaudeHarness) readOutput() {
	defer c.wg.Done()

	scanner := bufio.NewScanner(c.stdout)
	// Increase buffer size for large JSON messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		// Check if we should stop
		select {
		case <-c.done:
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse the JSON line and emit events
		c.parser.ParseLine(line)
	}

	if err := scanner.Err(); err != nil {
		// Check if we're shutting down before sending error
		select {
		case <-c.done:
			return
		default:
		}
		c.EventsWritable() <- harness.NewEvent(harness.EventError).
			WithError(fmt.Errorf("reading stdout: %w", err)).
			WithSource("claude-code").
			Build()
	}
}

// readErrors reads stderr and logs warnings
func (c *ClaudeHarness) readErrors() {
	defer c.wg.Done()

	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		// Check if we should stop
		select {
		case <-c.done:
			return
		default:
		}

		line := scanner.Text()
		if line != "" {
			// Check again before sending
			select {
			case <-c.done:
				return
			default:
			}
			// Emit as warning event
			c.EventsWritable() <- harness.NewEvent(harness.EventWarning).
				WithText(line).
				WithSource("claude-code").
				Build()
		}
	}
}

// detectVersion tries to get the Claude CLI version
func (c *ClaudeHarness) detectVersion() {
	cmd := exec.Command("claude", "--version")
	output, err := cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		c.SetVersion(version)
	}
}

// Stop terminates the Claude subprocess
func (c *ClaudeHarness) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.IsRunning() {
		return nil
	}

	// Signal goroutines to stop
	if c.done != nil {
		close(c.done)
	}

	// Cancel the context
	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	// Close stdin to signal EOF
	if c.stdin != nil {
		c.stdin.Close()
	}

	// Close stdout/stderr pipes to unblock scanner goroutines
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	// Wait for the process to exit with timeout
	if c.cmd != nil && c.cmd.Process != nil {
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- c.cmd.Wait()
		}()

		select {
		case err := <-waitDone:
			// Process exited
			if err != nil && !strings.Contains(err.Error(), "killed") {
				// Don't return error, just log it - process may have been killed by context
			}
		case <-time.After(5 * time.Second):
			// Timeout - force kill
			c.cmd.Process.Kill()
			<-waitDone // Wait for the goroutine to finish
		}
	}

	// Wait for reader goroutines to complete
	c.wg.Wait()

	// Now safe to close events channel
	c.CloseEvents()

	c.SetRunning(false)
	return nil
}

// SendPrompt sends a prompt to Claude via stdin
func (c *ClaudeHarness) SendPrompt(prompt string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.IsRunning() {
		return fmt.Errorf("harness not running")
	}

	if c.stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	// Write the prompt followed by newline
	_, err := fmt.Fprintln(c.stdin, prompt)
	if err != nil {
		return fmt.Errorf("sending prompt: %w", err)
	}

	return nil
}

// Capabilities returns what this harness supports
func (c *ClaudeHarness) Capabilities() harness.Capabilities {
	return harness.Capabilities{
		SupportedModels: []harness.Model{
			{ID: "opus", Name: "Claude Opus", Description: "Most capable model", Default: false},
			{ID: "sonnet", Name: "Claude Sonnet", Description: "Balanced performance", Default: true},
			{ID: "haiku", Name: "Claude Haiku", Description: "Fastest responses", Default: false},
		},
		SupportsHooks:     true,
		SupportsMCP:       true,
		SupportsStreaming: true,
		SupportsResume:    true,
	}
}

// SimulateEvent injects a simulated event for testing
func (c *ClaudeHarness) SimulateEvent(event harness.Event) {
	c.EventsWritable() <- event
}

// SimulateFileRead simulates a file read event for testing
func (c *ClaudeHarness) SimulateFileRead(path string) {
	c.EventsWritable() <- harness.NewEvent(harness.EventFileRead).
		WithTool("Read").
		WithToolInput("file_path", path).
		WithSource("claude-code").
		Build()
}

// SimulateFileWrite simulates a file write event for testing
func (c *ClaudeHarness) SimulateFileWrite(path string) {
	c.EventsWritable() <- harness.NewEvent(harness.EventFileWrite).
		WithTool("Write").
		WithToolInput("file_path", path).
		WithSource("claude-code").
		Build()
}

// SimulateFileEdit simulates a file edit event for testing
func (c *ClaudeHarness) SimulateFileEdit(path string) {
	c.EventsWritable() <- harness.NewEvent(harness.EventFileEdit).
		WithTool("Edit").
		WithToolInput("file_path", path).
		WithSource("claude-code").
		Build()
}
