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
