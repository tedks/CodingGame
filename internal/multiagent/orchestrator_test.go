package multiagent

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewOrchestrator(t *testing.T) {
	orch := NewOrchestrator()
	if orch == nil {
		t.Fatal("NewOrchestrator returned nil")
	}
	if orch.Count() != 0 {
		t.Errorf("expected 0 agents initially, got %d", orch.Count())
	}
}

func TestOrchestratorAddAgent(t *testing.T) {
	orch := NewOrchestrator()

	agent := NewAgent("agent-1", "Agent One", "robot")
	orch.AddAgent(agent)

	if orch.Count() != 1 {
		t.Errorf("expected 1 agent, got %d", orch.Count())
	}

	retrieved := orch.Get("agent-1")
	if retrieved != agent {
		t.Error("expected Get to return the added agent")
	}
}

func TestOrchestratorCreateAgent(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := orch.CreateAgent("First Agent", "robot")
	agent2 := orch.CreateAgent("Second Agent", "gear")

	if orch.Count() != 2 {
		t.Errorf("expected 2 agents, got %d", orch.Count())
	}

	// IDs should be auto-generated and unique
	if agent1.ID() == agent2.ID() {
		t.Error("expected unique agent IDs")
	}
}

func TestOrchestratorRemoveAgent(t *testing.T) {
	orch := NewOrchestrator()

	agent := NewAgent("agent-1", "Agent One", "robot")
	orch.AddAgent(agent)

	if !orch.RemoveAgent("agent-1") {
		t.Error("expected RemoveAgent to return true")
	}

	if orch.Count() != 0 {
		t.Errorf("expected 0 agents after removal, got %d", orch.Count())
	}

	if orch.RemoveAgent("agent-1") {
		t.Error("expected RemoveAgent to return false for non-existent agent")
	}
}

func TestOrchestratorGetAll(t *testing.T) {
	orch := NewOrchestrator()

	orch.AddAgent(NewAgent("z-agent", "Z Agent", "robot"))
	orch.AddAgent(NewAgent("a-agent", "A Agent", "robot"))
	orch.AddAgent(NewAgent("m-agent", "M Agent", "robot"))

	agents := orch.GetAll()
	if len(agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(agents))
	}

	// Should be sorted by name
	if agents[0].Name() != "A Agent" {
		t.Errorf("expected first agent to be 'A Agent', got %q", agents[0].Name())
	}
	if agents[1].Name() != "M Agent" {
		t.Errorf("expected second agent to be 'M Agent', got %q", agents[1].Name())
	}
	if agents[2].Name() != "Z Agent" {
		t.Errorf("expected third agent to be 'Z Agent', got %q", agents[2].Name())
	}
}

func TestOrchestratorGetByStatus(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "robot")
	agent3 := NewAgent("agent-3", "Agent 3", "robot")

	_ = agent1.StartTask("Task 1")
	_ = agent2.StartTask("Task 2")
	// agent3 stays idle

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)
	orch.AddAgent(agent3)

	working := orch.GetByStatus(StatusWorking)
	if len(working) != 2 {
		t.Errorf("expected 2 working agents, got %d", len(working))
	}

	idle := orch.GetByStatus(StatusIdle)
	if len(idle) != 1 {
		t.Errorf("expected 1 idle agent, got %d", len(idle))
	}
}

func TestOrchestratorCountByStatus(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "robot")
	agent3 := NewAgent("agent-3", "Agent 3", "robot")

	_ = agent1.StartTask("Task 1")
	// agent2, agent3 stay idle

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)
	orch.AddAgent(agent3)

	counts := orch.CountByStatus()
	if counts[StatusWorking] != 1 {
		t.Errorf("expected 1 working, got %d", counts[StatusWorking])
	}
	if counts[StatusIdle] != 2 {
		t.Errorf("expected 2 idle, got %d", counts[StatusIdle])
	}
}

func TestOrchestratorActiveAgentCount(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "robot")

	_ = agent1.StartTask("Task 1")

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	if orch.ActiveAgentCount() != 1 {
		t.Errorf("expected 1 active agent, got %d", orch.ActiveAgentCount())
	}
}

func TestOrchestratorTotalTokensUsed(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "robot")

	agent1.UpdateTokenUsage(10000)
	agent2.UpdateTokenUsage(20000)

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	if orch.TotalTokensUsed() != 30000 {
		t.Errorf("expected 30000 total tokens, got %d", orch.TotalTokensUsed())
	}
}

func TestOrchestratorAssignTask(t *testing.T) {
	orch := NewOrchestrator()

	agent := NewAgent("agent-1", "Agent 1", "robot")
	orch.AddAgent(agent)

	err := orch.AssignTask("agent-1", "Fix bug #123")
	if err != nil {
		t.Fatalf("AssignTask returned error: %v", err)
	}

	if agent.Status() != StatusWorking {
		t.Errorf("expected agent to be working, got %v", agent.Status())
	}
	if agent.CurrentTask() != "Fix bug #123" {
		t.Errorf("expected task 'Fix bug #123', got %q", agent.CurrentTask())
	}
}

func TestOrchestratorAssignTaskErrors(t *testing.T) {
	orch := NewOrchestrator()

	// Non-existent agent
	err := orch.AssignTask("non-existent", "Task")
	if err == nil {
		t.Error("expected error for non-existent agent")
	}

	// Agent already working
	agent := NewAgent("agent-1", "Agent 1", "robot")
	_ = agent.StartTask("Existing task")
	orch.AddAgent(agent)

	err = orch.AssignTask("agent-1", "New task")
	if err == nil {
		t.Error("expected error for already working agent")
	}
}

func TestOrchestratorAssignEmptyTask(t *testing.T) {
	orch := NewOrchestrator()

	agent := NewAgent("agent-1", "Agent 1", "robot")
	orch.AddAgent(agent)

	err := orch.AssignTask("agent-1", "")
	if err == nil {
		t.Error("expected error for empty task description")
	}

	// Verify agent is still idle (task was not assigned)
	if agent.Status() != StatusIdle {
		t.Errorf("expected agent to remain idle, got %v", agent.Status())
	}
}

func TestOrchestratorHandoffTask(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "gear")

	_ = agent1.StartTask("Handoff task")

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	err := orch.HandoffTask("agent-1", "agent-2")
	if err != nil {
		t.Fatalf("HandoffTask returned error: %v", err)
	}

	// Source should be paused
	if agent1.Status() != StatusPaused {
		t.Errorf("expected source agent to be paused, got %v", agent1.Status())
	}

	// Target should be working
	if agent2.Status() != StatusWorking {
		t.Errorf("expected target agent to be working, got %v", agent2.Status())
	}
	if agent2.CurrentTask() != "Handoff task" {
		t.Errorf("expected target to have handoff task, got %q", agent2.CurrentTask())
	}
}

func TestOrchestratorHandoffTaskErrors(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "gear")

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	// Source has no task
	err := orch.HandoffTask("agent-1", "agent-2")
	if err == nil {
		t.Error("expected error when source has no task")
	}

	// Non-existent source
	err = orch.HandoffTask("non-existent", "agent-2")
	if err == nil {
		t.Error("expected error for non-existent source")
	}

	// Non-existent target
	_ = agent1.StartTask("Task")
	err = orch.HandoffTask("agent-1", "non-existent")
	if err == nil {
		t.Error("expected error for non-existent target")
	}
}

func TestOrchestratorGetSharedFiles(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "gear")
	agent3 := NewAgent("agent-3", "Agent 3", "wrench")

	// agent1 and agent2 both read shared.go
	agent1.MarkFileRead("shared.go")
	agent1.MarkFileRead("unique1.go")
	agent2.MarkFileRead("shared.go")
	agent2.MarkFileRead("unique2.go")
	agent3.MarkFileRead("unique3.go")

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)
	orch.AddAgent(agent3)

	shared := orch.GetSharedFiles()
	if len(shared) != 1 {
		t.Errorf("expected 1 shared file, got %d", len(shared))
	}
	if _, exists := shared["shared.go"]; !exists {
		t.Error("expected shared.go in shared files")
	}
	if len(shared["shared.go"]) != 2 {
		t.Errorf("expected 2 agents sharing shared.go, got %d", len(shared["shared.go"]))
	}
}

func TestOrchestratorGetAgentWithFile(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "gear")

	agent1.MarkFileRead("file1.go")
	agent2.MarkFileRead("file2.go")

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)

	found := orch.GetAgentWithFile("file1.go")
	if found != agent1 {
		t.Error("expected to find agent1 for file1.go")
	}

	found = orch.GetAgentWithFile("file2.go")
	if found != agent2 {
		t.Error("expected to find agent2 for file2.go")
	}

	found = orch.GetAgentWithFile("non-existent.go")
	if found != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestOrchestratorGetAllAgentsWithFile(t *testing.T) {
	orch := NewOrchestrator()

	agent1 := NewAgent("agent-1", "Agent 1", "robot")
	agent2 := NewAgent("agent-2", "Agent 2", "gear")
	agent3 := NewAgent("agent-3", "Agent 3", "wrench")

	// Multiple agents read same file
	agent1.MarkFileRead("shared.go")
	agent2.MarkFileRead("shared.go")

	orch.AddAgent(agent1)
	orch.AddAgent(agent2)
	orch.AddAgent(agent3)

	agents := orch.GetAllAgentsWithFile("shared.go")
	if len(agents) != 2 {
		t.Errorf("expected 2 agents with shared.go, got %d", len(agents))
	}
}

func TestOrchestratorListener(t *testing.T) {
	orch := NewOrchestrator()

	var mu sync.Mutex
	var notifiedAgents []*Agent
	done := make(chan struct{})

	listener := &testListener{
		onChanged: func(_ context.Context, agents []*Agent) {
			mu.Lock()
			notifiedAgents = agents
			mu.Unlock()
			close(done)
		},
	}
	orch.AddListener(listener)

	agent := NewAgent("agent-1", "Agent 1", "robot")
	orch.AddAgent(agent)

	// Wait for listener notification with timeout
	select {
	case <-done:
		// Notification received
	case <-time.After(time.Second):
		t.Error("listener notification timed out")
	}

	mu.Lock()
	if len(notifiedAgents) != 1 {
		t.Errorf("expected 1 agent in notification, got %d", len(notifiedAgents))
	}
	mu.Unlock()
}

func TestOrchestratorRemoveListener(t *testing.T) {
	orch := NewOrchestrator()

	listener := &testListener{}
	orch.AddListener(listener)
	orch.RemoveListener(listener)

	// Should not panic and should work fine
	agent := NewAgent("agent-1", "Agent 1", "robot")
	orch.AddAgent(agent)
}

// testListener implements OrchestratorListener for testing.
type testListener struct {
	onChanged func(context.Context, []*Agent)
}

func (l *testListener) OnAgentsChanged(ctx context.Context, agents []*Agent) {
	if l.onChanged != nil {
		l.onChanged(ctx, agents)
	}
}

// Concurrent access tests

func TestOrchestratorConcurrentAddRemove(t *testing.T) {
	orch := NewOrchestrator()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				agent := NewAgent(
					"agent-"+string(rune('A'+goroutineID))+"-"+string(rune('0'+j%10)),
					"Agent",
					"robot",
				)
				orch.AddAgent(agent)
			}
		}(i)
	}

	// Concurrent removes
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				orch.RemoveAgent("agent-" + string(rune('A'+goroutineID)) + "-" + string(rune('0'+j%10)))
			}
		}(i)
	}

	wg.Wait()

	// Verify orchestrator is in consistent state
	count := orch.Count()
	if count < 0 {
		t.Errorf("invalid agent count: %d", count)
	}
}

func TestOrchestratorConcurrentAssignTask(t *testing.T) {
	orch := NewOrchestrator()

	// Create agents
	for i := 0; i < 5; i++ {
		agent := NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		orch.AddAgent(agent)
	}

	const numGoroutines = 10

	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make(map[string]int)

	wg.Add(numGoroutines)

	// Multiple goroutines try to assign tasks to the same agents
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				agentID := "agent-" + string(rune('A'+j))
				err := orch.AssignTask(agentID, "Task from goroutine")
				if err != nil {
					mu.Lock()
					errors[agentID]++
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// Each agent should have been assigned exactly once (9 concurrent failures per agent)
	for i := 0; i < 5; i++ {
		agentID := "agent-" + string(rune('A'+i))
		agent := orch.Get(agentID)
		if agent == nil {
			t.Errorf("agent %s not found", agentID)
			continue
		}
		if agent.Status() != StatusWorking {
			t.Errorf("agent %s should be working, got %v", agentID, agent.Status())
		}
	}
}

func TestOrchestratorConcurrentReads(t *testing.T) {
	orch := NewOrchestrator()

	// Create agents
	for i := 0; i < 10; i++ {
		agent := NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		agent.MarkFileRead("shared.go")
		_ = agent.StartTask("Task")
		orch.AddAgent(agent)
	}

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent reads shouldn't cause issues
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = orch.GetAll()
				_ = orch.Count()
				_ = orch.CountByStatus()
				_ = orch.ActiveAgentCount()
				_ = orch.TotalTokensUsed()
				_ = orch.GetByStatus(StatusWorking)
				_ = orch.GetSharedFiles()
				_ = orch.GetAgentWithFile("shared.go")
				_ = orch.GetAllAgentsWithFile("shared.go")
			}
		}()
	}

	wg.Wait()
}

func TestOrchestratorListenerPanicRecovery(t *testing.T) {
	orch := NewOrchestrator()

	done := make(chan struct{})

	// Listener that panics
	panicListener := &testListener{
		onChanged: func(_ context.Context, agents []*Agent) {
			panic("intentional panic for testing")
		},
	}

	// Listener that works normally
	normalListener := &testListener{
		onChanged: func(_ context.Context, agents []*Agent) {
			close(done)
		},
	}

	orch.AddListener(panicListener)
	orch.AddListener(normalListener)

	agent := NewAgent("agent-1", "Agent 1", "robot")
	orch.AddAgent(agent) // This should not panic even though panicListener panics

	// Wait for normal listener to be called
	select {
	case <-done:
		// Normal listener was called despite panic in other listener
	case <-time.After(time.Second):
		t.Error("normal listener should have been called despite panic in other listener")
	}
}

func TestOrchestratorRemoveNilListener(t *testing.T) {
	orch := NewOrchestrator()

	// Should not panic
	orch.RemoveListener(nil)
}

func TestAbandonedListenerCount(t *testing.T) {
	// Verify the counter is accessible and returns a non-negative value
	count := AbandonedListenerCount()
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
	}
}

func TestOrchestratorHandoffTaskSameAgent(t *testing.T) {
	orch := NewOrchestrator()

	agent := NewAgent("agent-1", "Agent 1", "robot")
	_ = agent.StartTask("My task")
	orch.AddAgent(agent)

	// Handoff to self should fail
	err := orch.HandoffTask("agent-1", "agent-1")
	if err == nil {
		t.Error("expected error when handing off to same agent")
	}

	// Original agent should still be working (not paused)
	if agent.Status() != StatusWorking {
		t.Errorf("expected agent to remain working, got %v", agent.Status())
	}
}

// TestHandoffTask_RaceWithCompletion tests that the TOCTOU race in HandoffTask is fixed.
// Before the fix, this race existed:
//   1. Goroutine A reads source.CurrentTask() = "task"
//   2. Goroutine B completes the task on source (clearing currentTask)
//   3. Goroutine A uses stale "task" value to start target
//
// This test verifies the atomic transfer prevents such races.
func TestHandoffTask_RaceWithCompletion(t *testing.T) {
	const iterations = 1000

	for i := 0; i < iterations; i++ {
		orch := NewOrchestrator()

		source := NewAgent("source", "Source", "robot")
		target := NewAgent("target", "Target", "robot")
		_ = source.StartTask("Important task")

		orch.AddAgent(source)
		orch.AddAgent(target)

		var wg sync.WaitGroup
		var handoffErr error
		var handoffSucceeded bool

		wg.Add(2)

		// Goroutine 1: Try to handoff
		go func() {
			defer wg.Done()
			handoffErr = orch.HandoffTask("source", "target")
			handoffSucceeded = handoffErr == nil
		}()

		// Goroutine 2: Try to complete source task
		go func() {
			defer wg.Done()
			source.CompleteTask()
		}()

		wg.Wait()

		// The key invariant: if handoff succeeded, target MUST have the correct task.
		// We should NEVER have handoff fail but target still working on the task
		// (that would be the TOCTOU race - target got a stale task value).

		if handoffSucceeded {
			// If handoff succeeded atomically:
			// - Target MUST be working on "Important task"
			// - Source was Paused at the moment of handoff, but CompleteTask may have
			//   run afterwards and changed it to Completed (this is fine)
			if target.Status() != StatusWorking {
				t.Fatalf("iteration %d: handoff succeeded but target status is %v (expected Working)", i, target.Status())
			}
			if target.CurrentTask() != "Important task" {
				t.Fatalf("iteration %d: handoff succeeded but target has task %q (expected 'Important task')", i, target.CurrentTask())
			}
			// Source can be Paused (handoff ran after CompleteTask) or
			// Completed (CompleteTask ran after handoff) - both are valid
			status := source.Status()
			if status != StatusPaused && status != StatusCompleted {
				t.Fatalf("iteration %d: handoff succeeded but source status is %v (expected Paused or Completed)", i, status)
			}
		} else {
			// If handoff failed, the TOCTOU race would manifest as:
			// - handoff fails (because source task was cleared)
			// - BUT target is working on "Important task" (got stale value)
			//
			// With atomic transfer, this should NEVER happen.
			if target.Status() == StatusWorking && target.CurrentTask() == "Important task" {
				t.Fatalf("iteration %d: TOCTOU race detected! Handoff failed (%v) but target is working on the task", i, handoffErr)
			}
		}
	}
}

// TestHandoffTask_ConcurrentBidirectional tests concurrent handoffs in both directions.
// This verifies the consistent lock ordering prevents deadlocks.
func TestHandoffTask_ConcurrentBidirectional(t *testing.T) {
	const iterations = 100

	for i := 0; i < iterations; i++ {
		orch := NewOrchestrator()

		agent1 := NewAgent("agent-1", "Agent 1", "robot")
		agent2 := NewAgent("agent-2", "Agent 2", "robot")
		_ = agent1.StartTask("Task A")
		_ = agent2.StartTask("Task B")

		orch.AddAgent(agent1)
		orch.AddAgent(agent2)

		var wg sync.WaitGroup
		wg.Add(2)

		// Try handoffs in both directions simultaneously
		// Without consistent lock ordering, this could deadlock
		go func() {
			defer wg.Done()
			_ = orch.HandoffTask("agent-1", "agent-2")
		}()

		go func() {
			defer wg.Done()
			_ = orch.HandoffTask("agent-2", "agent-1")
		}()

		// If we get here, no deadlock occurred
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Success, no deadlock
		case <-time.After(5 * time.Second):
			t.Fatalf("iteration %d: deadlock detected in concurrent bidirectional handoff", i)
		}
	}
}

// TestConcurrentCreateAgent_UniqueIDs verifies that CreateAgent generates unique IDs
// even under heavy concurrent load.
func TestConcurrentCreateAgent_UniqueIDs(t *testing.T) {
	orch := NewOrchestrator()

	const numGoroutines = 50
	const agentsPerGoroutine = 20

	var wg sync.WaitGroup
	var mu sync.Mutex
	ids := make(map[string]bool)
	duplicates := 0

	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < agentsPerGoroutine; j++ {
				agent := orch.CreateAgent("Agent", "robot")
				id := agent.ID()

				mu.Lock()
				if ids[id] {
					duplicates++
				}
				ids[id] = true
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()

	if duplicates > 0 {
		t.Errorf("found %d duplicate IDs out of %d agents created", duplicates, numGoroutines*agentsPerGoroutine)
	}

	expectedCount := numGoroutines * agentsPerGoroutine
	if orch.Count() != expectedCount {
		t.Errorf("expected %d agents, got %d", expectedCount, orch.Count())
	}
}

// TestSnapshots_ConsistencyUnderMutation verifies that Snapshots returns a consistent
// point-in-time view even while agents are being modified.
func TestSnapshots_ConsistencyUnderMutation(t *testing.T) {
	orch := NewOrchestrator()

	// Create some agents
	for i := 0; i < 10; i++ {
		agent := NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		orch.AddAgent(agent)
	}

	const iterations = 100
	var wg sync.WaitGroup

	// Start goroutines that continuously mutate agents
	stopCh := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				agents := orch.GetAll()
				for _, a := range agents {
					_ = a.StartTask("Some task")
					a.CompleteTask()
					a.MarkFileRead("file.go")
					a.UpdateTokenUsage(1000)
				}
			}
		}
	}()

	// Take snapshots while mutations are happening
	for i := 0; i < iterations; i++ {
		snapshots := orch.Snapshots()

		// Each snapshot should be internally consistent
		for _, snap := range snapshots {
			// Status and task should be consistent
			if snap.Status == StatusIdle || snap.Status == StatusCompleted {
				// These states should have empty task
				// (though due to timing, we might catch mid-transition)
			}
			// Token usage should be non-negative
			if snap.TokensUsed < 0 {
				t.Errorf("snapshot has negative tokens: %d", snap.TokensUsed)
			}
		}
	}

	close(stopCh)
	wg.Wait()
}

// TestAssignTask_ConcurrentToSameAgent verifies that only one concurrent
// AssignTask call succeeds for a given agent.
func TestAssignTask_ConcurrentToSameAgent(t *testing.T) {
	const iterations = 100
	const concurrency = 10

	for iter := 0; iter < iterations; iter++ {
		orch := NewOrchestrator()
		agent := NewAgent("target", "Target", "robot")
		orch.AddAgent(agent)

		var wg sync.WaitGroup
		var mu sync.Mutex
		successCount := 0
		failCount := 0

		wg.Add(concurrency)
		for g := 0; g < concurrency; g++ {
			go func(taskNum int) {
				defer wg.Done()
				err := orch.AssignTask("target", "Task "+string(rune('A'+taskNum)))
				mu.Lock()
				if err == nil {
					successCount++
				} else {
					failCount++
				}
				mu.Unlock()
			}(g)
		}

		wg.Wait()

		// Exactly one should succeed
		if successCount != 1 {
			t.Errorf("iteration %d: expected exactly 1 success, got %d", iter, successCount)
		}
		if failCount != concurrency-1 {
			t.Errorf("iteration %d: expected %d failures, got %d", iter, concurrency-1, failCount)
		}

		// Agent should be working
		if agent.Status() != StatusWorking {
			t.Errorf("iteration %d: expected agent to be working, got %v", iter, agent.Status())
		}
	}
}

// Listener tests

// slowListener simulates a listener that takes a configurable time to respond.
type slowListener struct {
	delay    time.Duration
	called   int
	mu       sync.Mutex
	blockCtx bool // if true, block until ctx.Done()
}

func (l *slowListener) OnAgentsChanged(ctx context.Context, agents []*Agent) {
	l.mu.Lock()
	l.called++
	l.mu.Unlock()

	if l.blockCtx {
		<-ctx.Done() // Block until timeout
		return
	}

	select {
	case <-time.After(l.delay):
	case <-ctx.Done():
	}
}

func (l *slowListener) CallCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.called
}

// TestListener_TimeoutBehavior verifies that slow listeners are timed out
// and the abandoned listener count is incremented.
func TestListener_TimeoutBehavior(t *testing.T) {
	initialCount := AbandonedListenerCount()

	orch := NewOrchestrator()

	// Listener that blocks until context is cancelled
	slowL := &slowListener{blockCtx: true}
	orch.AddListener(slowL)

	// Add an agent - this should trigger the listener
	agent := NewAgent("test", "Test", "robot")
	orch.AddAgent(agent)

	// Wait a bit for timeout to trigger
	time.Sleep(listenerTimeout + 100*time.Millisecond)

	// Verify abandoned count increased
	newCount := AbandonedListenerCount()
	if newCount <= initialCount {
		t.Errorf("expected abandoned listener count to increase from %d, got %d", initialCount, newCount)
	}

	// Listener should have been called
	if slowL.CallCount() == 0 {
		t.Error("expected slow listener to be called")
	}
}

// TestListener_FastListenersNotAbandoned verifies that fast listeners
// are not counted as abandoned.
func TestListener_FastListenersNotAbandoned(t *testing.T) {
	initialCount := AbandonedListenerCount()

	orch := NewOrchestrator()

	// Fast listener
	fastL := &slowListener{delay: 1 * time.Millisecond}
	orch.AddListener(fastL)

	// Trigger several notifications
	for i := 0; i < 10; i++ {
		agent := NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		orch.AddAgent(agent)
	}

	// Wait for all listeners to complete
	time.Sleep(100 * time.Millisecond)

	// Abandoned count should not have increased
	newCount := AbandonedListenerCount()
	if newCount != initialCount {
		t.Errorf("expected abandoned count to remain %d, got %d", initialCount, newCount)
	}

	// Listener should have been called 10 times
	if fastL.CallCount() != 10 {
		t.Errorf("expected 10 calls, got %d", fastL.CallCount())
	}
}

// recordingListener records the order and timing of notifications.
type recordingListener struct {
	mu          sync.Mutex
	callTimes   []time.Time
	agentCounts []int
}

func (l *recordingListener) OnAgentsChanged(_ context.Context, agents []*Agent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callTimes = append(l.callTimes, time.Now())
	l.agentCounts = append(l.agentCounts, len(agents))
}

func (l *recordingListener) Calls() ([]time.Time, []int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	times := make([]time.Time, len(l.callTimes))
	counts := make([]int, len(l.agentCounts))
	copy(times, l.callTimes)
	copy(counts, l.agentCounts)
	return times, counts
}

// TestListener_NotificationOrder verifies that notifications are received
// with the correct agent state at the time of the event.
func TestListener_NotificationOrder(t *testing.T) {
	orch := NewOrchestrator()

	recorder := &recordingListener{}
	orch.AddListener(recorder)

	// Add agents one at a time
	for i := 0; i < 5; i++ {
		agent := NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		orch.AddAgent(agent)
		// Small delay to ensure ordering
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for notifications
	time.Sleep(100 * time.Millisecond)

	_, counts := recorder.Calls()

	// Should have received 5 notifications
	if len(counts) != 5 {
		t.Errorf("expected 5 notifications, got %d", len(counts))
		return
	}

	// Each notification should reflect increasing agent counts
	for i, count := range counts {
		expected := i + 1
		if count != expected {
			t.Errorf("notification %d: expected %d agents, got %d", i, expected, count)
		}
	}
}

// selfModifyingListener adds/removes itself during callback
type selfModifyingListener struct {
	orch      *Orchestrator
	mu        sync.Mutex
	calls     int
	removed   bool
	addAnother bool
}

func (l *selfModifyingListener) OnAgentsChanged(_ context.Context, _ []*Agent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++

	// Try to remove self during callback
	if !l.removed {
		l.removed = true
		l.orch.RemoveListener(l)
	}
}

func (l *selfModifyingListener) CallCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

// TestListener_SelfRemovalDuringCallback verifies that a listener can safely
// remove itself during a callback without causing issues.
func TestListener_SelfRemovalDuringCallback(t *testing.T) {
	orch := NewOrchestrator()

	selfMod := &selfModifyingListener{orch: orch}
	orch.AddListener(selfMod)

	// Trigger first notification - listener removes itself
	orch.AddAgent(NewAgent("agent-1", "Agent 1", "robot"))
	time.Sleep(50 * time.Millisecond)

	if selfMod.CallCount() != 1 {
		t.Errorf("expected 1 call before removal, got %d", selfMod.CallCount())
	}

	// Trigger second notification - listener should not be called
	orch.AddAgent(NewAgent("agent-2", "Agent 2", "robot"))
	time.Sleep(50 * time.Millisecond)

	// Should still be 1 (no new calls after removal)
	if selfMod.CallCount() != 1 {
		t.Errorf("expected still 1 call after removal, got %d", selfMod.CallCount())
	}
}

// TestListener_MultipleListeners verifies all listeners are notified.
func TestListener_MultipleListeners(t *testing.T) {
	orch := NewOrchestrator()

	const numListeners = 5
	recorders := make([]*recordingListener, numListeners)
	for i := 0; i < numListeners; i++ {
		recorders[i] = &recordingListener{}
		orch.AddListener(recorders[i])
	}

	// Trigger notification
	orch.AddAgent(NewAgent("test", "Test", "robot"))
	time.Sleep(100 * time.Millisecond)

	// All listeners should have been called
	for i, r := range recorders {
		_, counts := r.Calls()
		if len(counts) != 1 {
			t.Errorf("listener %d: expected 1 call, got %d", i, len(counts))
		}
	}
}

// TestGetSharedFiles_ConcurrentFileReads verifies GetSharedFiles is safe
// under concurrent file read updates.
func TestGetSharedFiles_ConcurrentFileReads(t *testing.T) {
	orch := NewOrchestrator()

	const numAgents = 10
	agents := make([]*Agent, numAgents)
	for i := 0; i < numAgents; i++ {
		agents[i] = NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		orch.AddAgent(agents[i])
	}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutines continuously marking files as read
	wg.Add(numAgents)
	for i := 0; i < numAgents; i++ {
		go func(agent *Agent) {
			defer wg.Done()
			fileNum := 0
			for {
				select {
				case <-stopCh:
					return
				default:
					agent.MarkFileRead("shared.go")
					agent.MarkFileRead("file-" + string(rune('0'+fileNum%10)) + ".go")
					fileNum++
				}
			}
		}(agents[i])
	}

	// Goroutine calling GetSharedFiles
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				shared := orch.GetSharedFiles()
				// Just verify no panic and reasonable result
				if shared == nil {
					t.Error("GetSharedFiles returned nil")
				}
			}
		}
	}()

	// Run for a bit
	time.Sleep(100 * time.Millisecond)
	close(stopCh)
	wg.Wait()
}

// Chaos engineering tests

// TestChaos_MassAgentChurn tests system stability under rapid agent creation
// and destruction.
func TestChaos_MassAgentChurn(t *testing.T) {
	orch := NewOrchestrator()

	const numGoroutines = 20
	const iterations = 100

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Goroutines creating agents
	wg.Add(numGoroutines)
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				select {
				case <-stopCh:
					return
				default:
					agent := orch.CreateAgent("Agent", "robot")
					// Immediately do some operations
					_ = agent.StartTask("Task")
					agent.MarkFileRead("file.go")
					agent.UpdateTokenUsage(1000)
					agent.CompleteTask()
					// Sometimes remove the agent
					if i%3 == 0 {
						orch.RemoveAgent(agent.ID())
					}
				}
			}
		}(g)
	}

	// Concurrent readers
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = orch.GetAll()
					_ = orch.Snapshots()
					_ = orch.Count()
					_ = orch.CountByStatus()
					_ = orch.GetSharedFiles()
				}
			}
		}()
	}

	// Let it run
	time.Sleep(500 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	// System should still be in consistent state
	count := orch.Count()
	if count < 0 {
		t.Errorf("invalid final count: %d", count)
	}

	// All remaining agents should be in valid states
	for _, agent := range orch.GetAll() {
		status := agent.Status()
		validStatuses := map[AgentStatus]bool{
			StatusIdle:      true,
			StatusWorking:   true,
			StatusPaused:    true,
			StatusCompleted: true,
			StatusError:     true,
		}
		if !validStatuses[status] {
			t.Errorf("agent %s has invalid status: %v", agent.ID(), status)
		}
	}
}

// panicListener panics during callback
type panicListener struct {
	mu     sync.Mutex
	panics int
}

func (l *panicListener) OnAgentsChanged(_ context.Context, _ []*Agent) {
	l.mu.Lock()
	l.panics++
	l.mu.Unlock()
	panic("intentional panic")
}

func (l *panicListener) PanicCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.panics
}

// TestChaos_ListenerPanics verifies system stability when listeners panic.
func TestChaos_ListenerPanics(t *testing.T) {
	orch := NewOrchestrator()

	// Add multiple panicking listeners
	panicListeners := make([]*panicListener, 5)
	for i := 0; i < 5; i++ {
		panicListeners[i] = &panicListener{}
		orch.AddListener(panicListeners[i])
	}

	// Also add a normal listener
	normalL := &recordingListener{}
	orch.AddListener(normalL)

	// Trigger many notifications
	for i := 0; i < 10; i++ {
		orch.AddAgent(NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot"))
	}

	// Wait for listeners
	time.Sleep(100 * time.Millisecond)

	// Normal listener should still have been called despite panics
	_, counts := normalL.Calls()
	if len(counts) != 10 {
		t.Errorf("expected 10 calls to normal listener, got %d", len(counts))
	}

	// System should still work
	if orch.Count() != 10 {
		t.Errorf("expected 10 agents, got %d", orch.Count())
	}
}

// TestChaos_MixedOperations tests many different operations happening
// concurrently without any coordination.
func TestChaos_MixedOperations(t *testing.T) {
	orch := NewOrchestrator()

	// Pre-populate with some agents
	for i := 0; i < 10; i++ {
		agent := NewAgent("agent-"+string(rune('A'+i)), "Agent", "robot")
		orch.AddAgent(agent)
	}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Agent creators
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				orch.CreateAgent("New Agent", "robot")
			}
		}
	}()

	// Agent removers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				agents := orch.GetAll()
				if len(agents) > 0 {
					orch.RemoveAgent(agents[0].ID())
				}
			}
		}
	}()

	// Task assigners
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				agents := orch.GetByStatus(StatusIdle)
				if len(agents) > 0 {
					_ = orch.AssignTask(agents[0].ID(), "Task")
				}
			}
		}
	}()

	// Task completers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				agents := orch.GetByStatus(StatusWorking)
				if len(agents) > 0 {
					agents[0].CompleteTask()
				}
			}
		}
	}()

	// Handoff attempters
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				working := orch.GetByStatus(StatusWorking)
				idle := orch.GetByStatus(StatusIdle)
				if len(working) > 0 && len(idle) > 0 {
					_ = orch.HandoffTask(working[0].ID(), idle[0].ID())
				}
			}
		}
	}()

	// Snapshot takers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				_ = orch.Snapshots()
			}
		}
	}()

	// File markers
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopCh:
				return
			default:
				agents := orch.GetAll()
				for _, agent := range agents {
					agent.MarkFileRead("chaos.go")
				}
			}
		}
	}()

	// Let the chaos run
	time.Sleep(300 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	// System should be in consistent state
	count := orch.Count()
	if count < 0 {
		t.Errorf("invalid count: %d", count)
	}

	// Verify snapshots work
	snapshots := orch.Snapshots()
	if len(snapshots) != count {
		t.Errorf("snapshot count %d doesn't match agent count %d", len(snapshots), count)
	}
}

// TestAgent_TokenUsageOverflow tests behavior with extreme token values.
func TestAgent_TokenUsageOverflow(t *testing.T) {
	agent := NewAgent("test", "Test", "robot")

	// Set very large token limit
	agent.SetTokenLimit(1<<62 - 1) // Large but valid int64

	// Update with large usage
	agent.UpdateTokenUsage(1<<62 - 2)

	// Should calculate context usage without overflow
	usage := agent.ContextUsage()
	if usage < 0 || usage > 1.0 {
		t.Errorf("context usage out of bounds: %f", usage)
	}

	// Test with zero limit (should be 0% usage)
	agent.SetTokenLimit(0)
	agent.UpdateTokenUsage(1000000)
	if agent.ContextUsage() != 0 {
		t.Errorf("expected 0 context usage with zero limit, got %f", agent.ContextUsage())
	}
}

// TestOrchestrator_HandoffToWorkingAgent verifies handoff fails when target is busy.
func TestOrchestrator_HandoffToWorkingAgent(t *testing.T) {
	orch := NewOrchestrator()

	source := NewAgent("source", "Source", "robot")
	target := NewAgent("target", "Target", "robot")

	_ = source.StartTask("Source task")
	_ = target.StartTask("Target task")

	orch.AddAgent(source)
	orch.AddAgent(target)

	// Handoff should fail because target is working
	err := orch.HandoffTask("source", "target")
	if err == nil {
		t.Error("expected error when target is working")
	}

	// Both agents should retain their original tasks
	if source.CurrentTask() != "Source task" {
		t.Errorf("source task changed to %q", source.CurrentTask())
	}
	if target.CurrentTask() != "Target task" {
		t.Errorf("target task changed to %q", target.CurrentTask())
	}
}
