package multiagent

import (
	"testing"
)

func TestNewAgent(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	if agent.ID() != "agent-1" {
		t.Errorf("expected ID 'agent-1', got %q", agent.ID())
	}
	if agent.Name() != "Test Agent" {
		t.Errorf("expected Name 'Test Agent', got %q", agent.Name())
	}
	if agent.Icon() != "robot" {
		t.Errorf("expected Icon 'robot', got %q", agent.Icon())
	}
	if agent.Status() != StatusIdle {
		t.Errorf("expected Status Idle, got %v", agent.Status())
	}
}

func TestAgentPosition(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Set position
	agent.SetPosition(100.5, 200.5)

	x, y := agent.Position()
	if x != 100.5 || y != 200.5 {
		t.Errorf("expected position (100.5, 200.5), got (%f, %f)", x, y)
	}
}

func TestAgentFilesRead(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Initially no files read
	if agent.FileCount() != 0 {
		t.Errorf("expected 0 files read initially, got %d", agent.FileCount())
	}
	if agent.HasRead("test.go") {
		t.Error("expected HasRead to return false for unread file")
	}

	// Mark files as read
	agent.MarkFileRead("test.go")
	agent.MarkFileRead("main.go")

	if agent.FileCount() != 2 {
		t.Errorf("expected 2 files read, got %d", agent.FileCount())
	}
	if !agent.HasRead("test.go") {
		t.Error("expected HasRead to return true for read file")
	}

	// Get files read
	files := agent.FilesRead()
	if len(files) != 2 {
		t.Errorf("expected 2 files in FilesRead(), got %d", len(files))
	}
}

func TestAgentTokenUsage(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Set token limit
	agent.SetTokenLimit(100000)
	if agent.TokenLimit() != 100000 {
		t.Errorf("expected TokenLimit 100000, got %d", agent.TokenLimit())
	}

	// Update usage
	agent.UpdateTokenUsage(50000)

	if agent.TokensUsed() != 50000 {
		t.Errorf("expected TokensUsed 50000, got %d", agent.TokensUsed())
	}

	usage := agent.ContextUsage()
	if usage < 0.49 || usage > 0.51 {
		t.Errorf("expected ContextUsage ~0.5, got %f", usage)
	}
}

func TestAgentTaskLifecycle(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Start task
	agent.StartTask("Fix bug #123")
	if agent.Status() != StatusWorking {
		t.Errorf("expected Status Working, got %v", agent.Status())
	}
	if agent.CurrentTask() != "Fix bug #123" {
		t.Errorf("expected CurrentTask 'Fix bug #123', got %q", agent.CurrentTask())
	}

	// Pause task
	agent.PauseTask()
	if agent.Status() != StatusPaused {
		t.Errorf("expected Status Paused, got %v", agent.Status())
	}

	// Resume task
	agent.ResumeTask()
	if agent.Status() != StatusWorking {
		t.Errorf("expected Status Working after resume, got %v", agent.Status())
	}

	// Complete task
	agent.CompleteTask()
	if agent.Status() != StatusCompleted {
		t.Errorf("expected Status Completed, got %v", agent.Status())
	}
}

func TestAgentError(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Set error
	testErr := &testError{msg: "connection failed"}
	agent.SetError(testErr)

	if agent.Status() != StatusError {
		t.Errorf("expected Status Error, got %v", agent.Status())
	}
	if agent.LastError() != testErr {
		t.Error("expected LastError to return the error")
	}
}

func TestAgentReset(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Set up state
	agent.StartTask("Some task")
	agent.MarkFileRead("test.go")
	agent.UpdateTokenUsage(50000)

	// Reset
	agent.Reset()

	if agent.Status() != StatusIdle {
		t.Errorf("expected Status Idle after reset, got %v", agent.Status())
	}
	if agent.CurrentTask() != "" {
		t.Errorf("expected empty CurrentTask after reset, got %q", agent.CurrentTask())
	}
	if agent.FileCount() != 0 {
		t.Errorf("expected 0 files after reset, got %d", agent.FileCount())
	}
	if agent.TokensUsed() != 0 {
		t.Errorf("expected 0 tokens after reset, got %d", agent.TokensUsed())
	}
}

func TestAgentPauseResumeEdgeCases(t *testing.T) {
	agent := NewAgent("agent-1", "Test Agent", "robot")

	// Pause when idle should do nothing
	agent.PauseTask()
	if agent.Status() != StatusIdle {
		t.Errorf("expected Status Idle after pause when idle, got %v", agent.Status())
	}

	// Resume when idle should do nothing
	agent.ResumeTask()
	if agent.Status() != StatusIdle {
		t.Errorf("expected Status Idle after resume when idle, got %v", agent.Status())
	}
}

// testError implements error for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
