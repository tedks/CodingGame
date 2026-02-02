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
	if err := agent.StartTask("Fix bug #123"); err != nil {
		t.Fatalf("StartTask returned error: %v", err)
	}
	if agent.Status() != StatusWorking {
		t.Errorf("expected Status Working, got %v", agent.Status())
	}
	if agent.CurrentTask() != "Fix bug #123" {
		t.Errorf("expected CurrentTask 'Fix bug #123', got %q", agent.CurrentTask())
	}

	// Try to start another task while working - should fail
	if err := agent.StartTask("Another task"); err == nil {
		t.Error("expected error when starting task while already working")
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
	// Verify currentTask is cleared after completion
	if agent.CurrentTask() != "" {
		t.Errorf("expected empty CurrentTask after completion, got %q", agent.CurrentTask())
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

// State machine exhaustive tests
//
// The agent state machine is:
//   Idle -> Working (StartTask)
//   Working -> Paused (PauseTask)
//   Working -> Completed (CompleteTask)
//   Paused -> Working (ResumeTask)
//   Any state -> Error (SetError)
//   Any state -> Idle (Reset)

// TestAgentStateMachine_AllTransitions tests every valid state transition.
func TestAgentStateMachine_AllTransitions(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Agent)
		action       func(*Agent) error
		expectStatus AgentStatus
		expectError  bool
	}{
		// Idle -> Working
		{
			name:         "Idle -> Working via StartTask",
			setup:        func(a *Agent) {},
			action:       func(a *Agent) error { return a.StartTask("task") },
			expectStatus: StatusWorking,
			expectError:  false,
		},
		// Working -> Paused
		{
			name:         "Working -> Paused via PauseTask",
			setup:        func(a *Agent) { _ = a.StartTask("task") },
			action:       func(a *Agent) error { a.PauseTask(); return nil },
			expectStatus: StatusPaused,
			expectError:  false,
		},
		// Working -> Completed
		{
			name:         "Working -> Completed via CompleteTask",
			setup:        func(a *Agent) { _ = a.StartTask("task") },
			action:       func(a *Agent) error { a.CompleteTask(); return nil },
			expectStatus: StatusCompleted,
			expectError:  false,
		},
		// Paused -> Working
		{
			name: "Paused -> Working via ResumeTask",
			setup: func(a *Agent) {
				_ = a.StartTask("task")
				a.PauseTask()
			},
			action:       func(a *Agent) error { a.ResumeTask(); return nil },
			expectStatus: StatusWorking,
			expectError:  false,
		},
		// Paused -> Working via StartTask (new task)
		{
			name: "Paused -> Working via StartTask (new task)",
			setup: func(a *Agent) {
				_ = a.StartTask("task")
				a.PauseTask()
			},
			action:       func(a *Agent) error { return a.StartTask("new task") },
			expectStatus: StatusWorking,
			expectError:  false,
		},
		// Completed -> Working
		{
			name: "Completed -> Working via StartTask",
			setup: func(a *Agent) {
				_ = a.StartTask("task")
				a.CompleteTask()
			},
			action:       func(a *Agent) error { return a.StartTask("new task") },
			expectStatus: StatusWorking,
			expectError:  false,
		},
		// Error -> Working
		{
			name: "Error -> Working via StartTask",
			setup: func(a *Agent) {
				a.SetError(&testError{"some error"})
			},
			action:       func(a *Agent) error { return a.StartTask("recovery task") },
			expectStatus: StatusWorking,
			expectError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agent := NewAgent("test", "Test", "robot")
			tc.setup(agent)
			err := tc.action(agent)

			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if agent.Status() != tc.expectStatus {
				t.Errorf("expected status %v, got %v", tc.expectStatus, agent.Status())
			}
		})
	}
}

// TestAgentStateMachine_InvalidTransitions tests transitions that should be no-ops.
func TestAgentStateMachine_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Agent) AgentStatus
		action       func(*Agent)
		expectNoOp   bool // status should remain unchanged
	}{
		// PauseTask from non-Working states (should be no-op)
		{
			name:       "PauseTask when Idle (no-op)",
			setup:      func(a *Agent) AgentStatus { return a.Status() },
			action:     func(a *Agent) { a.PauseTask() },
			expectNoOp: true,
		},
		{
			name: "PauseTask when Paused (no-op)",
			setup: func(a *Agent) AgentStatus {
				_ = a.StartTask("task")
				a.PauseTask()
				return a.Status()
			},
			action:     func(a *Agent) { a.PauseTask() },
			expectNoOp: true,
		},
		{
			name: "PauseTask when Completed (no-op)",
			setup: func(a *Agent) AgentStatus {
				_ = a.StartTask("task")
				a.CompleteTask()
				return a.Status()
			},
			action:     func(a *Agent) { a.PauseTask() },
			expectNoOp: true,
		},
		{
			name: "PauseTask when Error (no-op)",
			setup: func(a *Agent) AgentStatus {
				a.SetError(&testError{"error"})
				return a.Status()
			},
			action:     func(a *Agent) { a.PauseTask() },
			expectNoOp: true,
		},
		// ResumeTask from non-Paused states (should be no-op)
		{
			name:       "ResumeTask when Idle (no-op)",
			setup:      func(a *Agent) AgentStatus { return a.Status() },
			action:     func(a *Agent) { a.ResumeTask() },
			expectNoOp: true,
		},
		{
			name: "ResumeTask when Working (no-op)",
			setup: func(a *Agent) AgentStatus {
				_ = a.StartTask("task")
				return a.Status()
			},
			action:     func(a *Agent) { a.ResumeTask() },
			expectNoOp: true,
		},
		{
			name: "ResumeTask when Completed (no-op)",
			setup: func(a *Agent) AgentStatus {
				_ = a.StartTask("task")
				a.CompleteTask()
				return a.Status()
			},
			action:     func(a *Agent) { a.ResumeTask() },
			expectNoOp: true,
		},
		{
			name: "ResumeTask when Error (no-op)",
			setup: func(a *Agent) AgentStatus {
				a.SetError(&testError{"error"})
				return a.Status()
			},
			action:     func(a *Agent) { a.ResumeTask() },
			expectNoOp: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agent := NewAgent("test", "Test", "robot")
			initialStatus := tc.setup(agent)
			tc.action(agent)

			if tc.expectNoOp && agent.Status() != initialStatus {
				t.Errorf("expected no-op (status to remain %v), but got %v", initialStatus, agent.Status())
			}
		})
	}
}

// TestAgentStateMachine_StartTaskErrors tests that StartTask fails when already working.
func TestAgentStateMachine_StartTaskErrors(t *testing.T) {
	agent := NewAgent("test", "Test", "robot")
	_ = agent.StartTask("first task")

	// StartTask should fail when already working
	err := agent.StartTask("second task")
	if err == nil {
		t.Error("expected error when starting task while working")
	}

	// Original task should be preserved
	if agent.CurrentTask() != "first task" {
		t.Errorf("expected task 'first task', got %q", agent.CurrentTask())
	}
}

// TestAgentStateMachine_SetErrorFromEveryState tests SetError transitions from all states.
func TestAgentStateMachine_SetErrorFromEveryState(t *testing.T) {
	states := []struct {
		name  string
		setup func(*Agent)
	}{
		{"Idle", func(a *Agent) {}},
		{"Working", func(a *Agent) { _ = a.StartTask("task") }},
		{"Paused", func(a *Agent) { _ = a.StartTask("task"); a.PauseTask() }},
		{"Completed", func(a *Agent) { _ = a.StartTask("task"); a.CompleteTask() }},
		{"Error", func(a *Agent) { a.SetError(&testError{"first error"}) }},
	}

	for _, state := range states {
		t.Run("SetError from "+state.name, func(t *testing.T) {
			agent := NewAgent("test", "Test", "robot")
			state.setup(agent)

			testErr := &testError{"test error"}
			agent.SetError(testErr)

			if agent.Status() != StatusError {
				t.Errorf("expected status Error, got %v", agent.Status())
			}
			if agent.LastError() != testErr {
				t.Error("expected LastError to return the set error")
			}
		})
	}
}

// TestAgentStateMachine_ResetFromEveryState tests Reset transitions from all states.
func TestAgentStateMachine_ResetFromEveryState(t *testing.T) {
	states := []struct {
		name  string
		setup func(*Agent)
	}{
		{"Idle", func(a *Agent) {}},
		{"Working", func(a *Agent) {
			_ = a.StartTask("task")
			a.MarkFileRead("file.go")
			a.UpdateTokenUsage(5000)
		}},
		{"Paused", func(a *Agent) {
			_ = a.StartTask("task")
			a.MarkFileRead("file.go")
			a.PauseTask()
		}},
		{"Completed", func(a *Agent) {
			_ = a.StartTask("task")
			a.MarkFileRead("file.go")
			a.CompleteTask()
		}},
		{"Error", func(a *Agent) {
			_ = a.StartTask("task")
			a.MarkFileRead("file.go")
			a.SetError(&testError{"error"})
		}},
	}

	for _, state := range states {
		t.Run("Reset from "+state.name, func(t *testing.T) {
			agent := NewAgent("test", "Test", "robot")
			state.setup(agent)

			agent.Reset()

			if agent.Status() != StatusIdle {
				t.Errorf("expected status Idle, got %v", agent.Status())
			}
			if agent.CurrentTask() != "" {
				t.Errorf("expected empty task, got %q", agent.CurrentTask())
			}
			if agent.FileCount() != 0 {
				t.Errorf("expected 0 files, got %d", agent.FileCount())
			}
			if agent.TokensUsed() != 0 {
				t.Errorf("expected 0 tokens, got %d", agent.TokensUsed())
			}
			if agent.ContextUsage() != 0 {
				t.Errorf("expected 0 context usage, got %f", agent.ContextUsage())
			}
			if agent.LastError() != nil {
				t.Error("expected nil LastError after reset")
			}
		})
	}
}

// TestAgentSnapshot_Immutability verifies that snapshots are truly immutable.
func TestAgentSnapshot_Immutability(t *testing.T) {
	agent := NewAgent("test", "Test", "robot")
	_ = agent.StartTask("original task")
	agent.UpdateTokenUsage(5000)

	snap := agent.Snapshot()

	// Modify the original agent
	agent.CompleteTask()
	_ = agent.StartTask("new task")
	agent.UpdateTokenUsage(10000)

	// Snapshot should retain original values
	if snap.Status != StatusWorking {
		t.Errorf("snapshot status changed: expected Working, got %v", snap.Status)
	}
	if snap.CurrentTask != "original task" {
		t.Errorf("snapshot task changed: expected 'original task', got %q", snap.CurrentTask)
	}
	if snap.TokensUsed != 5000 {
		t.Errorf("snapshot tokens changed: expected 5000, got %d", snap.TokensUsed)
	}
}

// TestAgentTokenUsage_ZeroLimit tests behavior when token limit is zero.
func TestAgentTokenUsage_ZeroLimit(t *testing.T) {
	agent := NewAgent("test", "Test", "robot")
	agent.SetTokenLimit(0)
	agent.UpdateTokenUsage(1000)

	// With zero limit, context usage should be 0 (no artificial constraint)
	if agent.ContextUsage() != 0 {
		t.Errorf("expected 0 context usage with zero limit, got %f", agent.ContextUsage())
	}
}

// TestAgentFilesRead_CopySemantics verifies FilesRead returns a copy.
func TestAgentFilesRead_CopySemantics(t *testing.T) {
	agent := NewAgent("test", "Test", "robot")
	agent.MarkFileRead("file1.go")
	agent.MarkFileRead("file2.go")

	files := agent.FilesRead()

	// Modify the returned map
	files["file3.go"] = files["file1.go"]
	delete(files, "file1.go")

	// Original agent should be unaffected
	if agent.FileCount() != 2 {
		t.Errorf("expected 2 files in agent, got %d", agent.FileCount())
	}
	if !agent.HasRead("file1.go") {
		t.Error("agent should still have file1.go")
	}
}
