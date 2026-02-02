package multiagent

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// listenerTimeout is the maximum time a listener callback can take before being considered stuck.
// Listener goroutines that exceed this will be abandoned (the goroutine itself may still run,
// but we won't block waiting for it).
const listenerTimeout = 5 * time.Second

// abandonedListenerCount tracks the number of listener callbacks that timed out.
// This counter is useful for debugging listener performance issues. When a listener
// callback takes longer than listenerTimeout, the orchestrator stops waiting but
// the goroutine may still be running in the background if the listener ignores ctx.
var abandonedListenerCount int64

// AbandonedListenerCount returns the number of listener callbacks that have
// timed out. This is useful for debugging listener performance issues and detecting
// potential goroutine leaks from slow or stuck listeners.
func AbandonedListenerCount() int64 {
	return atomic.LoadInt64(&abandonedListenerCount)
}

// OrchestratorListener receives notifications about agent changes.
type OrchestratorListener interface {
	// OnAgentsChanged is called when agents are added, removed, or updated.
	// The provided slice is read-only and must not be modified.
	// For an immutable, point-in-time view, call Agent.Snapshot on each entry.
	// Listeners are called asynchronously in goroutines.
	// The context is canceled when the listener times out.
	// Implementations should return promptly when ctx.Done() is closed.
	OnAgentsChanged(ctx context.Context, agents []*Agent)
}

// Orchestrator manages multiple concurrent agents and their task assignments.
// It provides a central point for:
// - Adding/removing agents
// - Assigning tasks to agents
// - Monitoring agent status
// - Coordinating agent handoffs
//
// # Concurrency
//
// mu protects: agents, listeners, nextID.
// The agents map is keyed by agent.ID and IDs are immutable after creation.
// Listener callbacks are invoked without holding mu; a snapshot of listeners is
// taken to avoid holding the lock across user code.
type Orchestrator struct {
	mu sync.RWMutex

	agents    map[string]*Agent
	listeners []OrchestratorListener

	// Agent ID counter for generating unique IDs
	nextID int
}

// NewOrchestrator creates a new multi-agent orchestrator.
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		agents: make(map[string]*Agent),
	}
}

// AddAgent adds a new agent to the orchestrator.
// Returns the agent for method chaining.
func (o *Orchestrator) AddAgent(agent *Agent) *Agent {
	o.mu.Lock()
	o.agents[agent.ID()] = agent
	listeners := make([]OrchestratorListener, len(o.listeners))
	copy(listeners, o.listeners)
	o.mu.Unlock()

	o.notifyListeners(listeners)
	return agent
}

// CreateAgent creates and adds a new agent with auto-generated ID.
func (o *Orchestrator) CreateAgent(name, icon string) *Agent {
	o.mu.Lock()
	id := fmt.Sprintf("agent-%d", o.nextID)
	o.nextID++
	o.mu.Unlock()

	agent := NewAgent(id, name, icon)
	return o.AddAgent(agent)
}

// RemoveAgent removes an agent from the orchestrator.
func (o *Orchestrator) RemoveAgent(id string) bool {
	o.mu.Lock()
	if _, exists := o.agents[id]; !exists {
		o.mu.Unlock()
		return false
	}
	delete(o.agents, id)
	listeners := make([]OrchestratorListener, len(o.listeners))
	copy(listeners, o.listeners)
	o.mu.Unlock()

	o.notifyListeners(listeners)
	return true
}

// Get returns an agent by ID, or nil if not found.
func (o *Orchestrator) Get(id string) *Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.agents[id]
}

// GetAll returns all agents sorted by name.
// The returned slice contains live agent pointers; use Agent.Snapshot for a
// read-only copy of agent state.
func (o *Orchestrator) GetAll() []*Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	agents := make([]*Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name() < agents[j].Name()
	})

	return agents
}

// Snapshots returns immutable snapshots of all agents sorted by name.
func (o *Orchestrator) Snapshots() []AgentSnapshot {
	o.mu.RLock()
	agents := make([]*Agent, 0, len(o.agents))
	for _, agent := range o.agents {
		agents = append(agents, agent)
	}
	o.mu.RUnlock()

	snapshots := make([]AgentSnapshot, len(agents))
	for i, agent := range agents {
		snapshots[i] = agent.Snapshot()
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Name < snapshots[j].Name
	})
	return snapshots
}

// Count returns the total number of agents.
func (o *Orchestrator) Count() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.agents)
}

// GetByStatus returns agents with a specific status.
func (o *Orchestrator) GetByStatus(status AgentStatus) []*Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var agents []*Agent
	for _, agent := range o.agents {
		if agent.Status() == status {
			agents = append(agents, agent)
		}
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name() < agents[j].Name()
	})

	return agents
}

// CountByStatus returns the count of agents in each status.
func (o *Orchestrator) CountByStatus() map[AgentStatus]int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	counts := make(map[AgentStatus]int)
	for _, agent := range o.agents {
		counts[agent.Status()]++
	}
	return counts
}

// ActiveAgentCount returns the number of agents currently working.
func (o *Orchestrator) ActiveAgentCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	count := 0
	for _, agent := range o.agents {
		if agent.Status() == StatusWorking {
			count++
		}
	}
	return count
}

// TotalTokensUsed returns the sum of tokens used by all agents.
func (o *Orchestrator) TotalTokensUsed() int64 {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var total int64
	for _, agent := range o.agents {
		total += agent.TokensUsed()
	}
	return total
}

// AssignTask assigns a task to a specific agent.
// Returns an error if the agent is not found, already working, or task description is empty.
// The operation is atomic - uses agent's StartTask which checks and sets status atomically.
func (o *Orchestrator) AssignTask(agentID string, taskDescription string) error {
	if taskDescription == "" {
		return fmt.Errorf("task description cannot be empty")
	}

	o.mu.RLock()
	agent, exists := o.agents[agentID]
	listeners := make([]OrchestratorListener, len(o.listeners))
	copy(listeners, o.listeners)
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// StartTask is atomic - returns error if agent is already working
	if err := agent.StartTask(taskDescription); err != nil {
		return err
	}

	o.notifyListeners(listeners)
	return nil
}

// HandoffTask transfers a task from one agent to another.
// The source agent is paused and the target agent starts the task.
// The operation is atomic - both agents are locked in consistent order to prevent
// TOCTOU races and deadlocks.
func (o *Orchestrator) HandoffTask(fromAgentID, toAgentID string) error {
	if fromAgentID == toAgentID {
		return fmt.Errorf("cannot handoff task to same agent: %s", fromAgentID)
	}

	o.mu.RLock()
	fromAgent, fromExists := o.agents[fromAgentID]
	toAgent, toExists := o.agents[toAgentID]
	listeners := make([]OrchestratorListener, len(o.listeners))
	copy(listeners, o.listeners)
	o.mu.RUnlock()

	if !fromExists {
		return fmt.Errorf("source agent not found: %s", fromAgentID)
	}
	if !toExists {
		return fmt.Errorf("target agent not found: %s", toAgentID)
	}

	// Atomic transfer using consistent lock ordering
	_, err := transferTask(fromAgent, toAgent)
	if err != nil {
		return err
	}

	o.notifyListeners(listeners)
	return nil
}

// transferTask atomically transfers a task from one agent to another.
// It locks both agents in consistent order (by ID) to prevent deadlocks.
// Returns the transferred task description, or error if transfer fails.
//
// # TOCTOU Race Prevention
//
// This function prevents the Time-Of-Check-Time-Of-Use race that would occur
// if we read source.CurrentTask() then called target.StartTask() separately.
// Between those calls, another goroutine could complete or modify the source's task.
// By holding both locks, we ensure the check and use happen atomically.
func transferTask(from, to *Agent) (string, error) {
	// Lock in consistent order by ID to prevent deadlock.
	// Without consistent ordering, goroutine A locking (agent1, agent2) and
	// goroutine B locking (agent2, agent1) could deadlock.
	first, second := from, to
	if from.id > to.id {
		first, second = to, from
	}

	first.mu.Lock()
	defer first.mu.Unlock()
	second.mu.Lock()
	defer second.mu.Unlock()

	// Now check AND transfer atomically
	task := from.currentTask
	if task == "" {
		return "", fmt.Errorf("source agent %s has no active task", from.id)
	}

	if to.status == StatusWorking {
		return "", fmt.Errorf("target agent %s is already working on: %s", to.id, to.currentTask)
	}

	// Transfer: start target, pause source
	to.status = StatusWorking
	to.currentTask = task
	to.lastActivity = time.Now()
	to.lastError = nil

	from.status = StatusPaused
	from.lastActivity = time.Now()

	return task, nil
}

// AddListener registers a listener for orchestrator events.
func (o *Orchestrator) AddListener(l OrchestratorListener) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.listeners = append(o.listeners, l)
}

// RemoveListener unregisters a listener.
func (o *Orchestrator) RemoveListener(l OrchestratorListener) {
	if l == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for i, listener := range o.listeners {
		if listener == l {
			o.listeners = append(o.listeners[:i], o.listeners[i+1:]...)
			return
		}
	}
}

// notifyListeners asynchronously notifies all listeners with timeout protection.
// Each listener callback runs in its own goroutine with panic recovery.
// Note: If a listener blocks longer than listenerTimeout, the timeout context is canceled.
// Listeners must respect ctx.Done() to avoid lingering goroutines.
//
// Goroutines:
// - listenerRunner: wraps each listener, enforces a per-listener timeout, exits on done/timeout.
// - listenerCall: invokes listener.OnAgentsChanged with a timeout context and closes done when finished.
//
// Channel: done (chan struct{})
// - Created by: listenerRunner
// - Writers: listenerCall (close)
// - Readers: listenerRunner (select)
func (o *Orchestrator) notifyListeners(listeners []OrchestratorListener) {
	if len(listeners) == 0 {
		return
	}

	agents := o.GetAll()
	var wg sync.WaitGroup

	for _, l := range listeners {
		wg.Add(1)
		go func(listener OrchestratorListener) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), listenerTimeout)
			defer cancel()

			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() {
					if rec := recover(); rec != nil {
						log.Printf("orchestrator: listener panic recovered: %v", rec)
					}
				}()
				listener.OnAgentsChanged(ctx, agents)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				atomic.AddInt64(&abandonedListenerCount, 1)
				log.Printf("orchestrator: listener notification timeout after %v", listenerTimeout)
			}
		}(l)
	}

	wg.Wait()
}

// GetSharedFiles returns files that have been read by multiple agents.
// This helps identify shared knowledge across agents.
func (o *Orchestrator) GetSharedFiles() map[string][]string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Build file -> agents map
	fileAgents := make(map[string][]string)
	for _, agent := range o.agents {
		files := agent.FilesRead()
		for file := range files {
			fileAgents[file] = append(fileAgents[file], agent.ID())
		}
	}

	// Filter to only files read by multiple agents
	shared := make(map[string][]string)
	for file, agents := range fileAgents {
		if len(agents) > 1 {
			shared[file] = agents
		}
	}

	return shared
}

// GetAgentWithFile returns the first agent that has read a specific file.
// Returns nil if no agent has read the file.
func (o *Orchestrator) GetAgentWithFile(filePath string) *Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for _, agent := range o.agents {
		if agent.HasRead(filePath) {
			return agent
		}
	}
	return nil
}

// GetAllAgentsWithFile returns all agents that have read a specific file.
func (o *Orchestrator) GetAllAgentsWithFile(filePath string) []*Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var agents []*Agent
	for _, agent := range o.agents {
		if agent.HasRead(filePath) {
			agents = append(agents, agent)
		}
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name() < agents[j].Name()
	})

	return agents
}
