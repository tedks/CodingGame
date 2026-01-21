package multiagent

import (
	"fmt"
	"sort"
	"sync"
)

// OrchestratorListener receives notifications about agent changes.
type OrchestratorListener interface {
	// OnAgentsChanged is called when agents are added, removed, or updated.
	// The provided slice is read-only and must not be modified.
	// Listeners are called asynchronously in goroutines.
	OnAgentsChanged(agents []*Agent)
}

// Orchestrator manages multiple concurrent agents and their task assignments.
// It provides a central point for:
// - Adding/removing agents
// - Assigning tasks to agents
// - Monitoring agent status
// - Coordinating agent handoffs
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
// Returns an error if the agent is not found or already working.
func (o *Orchestrator) AssignTask(agentID string, taskDescription string) error {
	o.mu.RLock()
	agent, exists := o.agents[agentID]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	status := agent.Status()
	if status == StatusWorking {
		return fmt.Errorf("agent %s is already working", agentID)
	}

	agent.StartTask(taskDescription)

	// Notify listeners
	o.mu.RLock()
	listeners := make([]OrchestratorListener, len(o.listeners))
	copy(listeners, o.listeners)
	o.mu.RUnlock()

	o.notifyListeners(listeners)
	return nil
}

// HandoffTask transfers a task from one agent to another.
// The source agent is paused and the target agent starts the task.
func (o *Orchestrator) HandoffTask(fromAgentID, toAgentID string) error {
	o.mu.RLock()
	fromAgent, fromExists := o.agents[fromAgentID]
	toAgent, toExists := o.agents[toAgentID]
	o.mu.RUnlock()

	if !fromExists {
		return fmt.Errorf("source agent not found: %s", fromAgentID)
	}
	if !toExists {
		return fmt.Errorf("target agent not found: %s", toAgentID)
	}

	// Get the task from the source agent
	task := fromAgent.CurrentTask()
	if task == "" {
		return fmt.Errorf("source agent %s has no active task", fromAgentID)
	}

	// Pause source and start target
	fromAgent.PauseTask()
	toAgent.StartTask(task)

	// Notify listeners
	o.mu.RLock()
	listeners := make([]OrchestratorListener, len(o.listeners))
	copy(listeners, o.listeners)
	o.mu.RUnlock()

	o.notifyListeners(listeners)
	return nil
}

// AddListener registers a listener for orchestrator events.
func (o *Orchestrator) AddListener(l OrchestratorListener) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.listeners = append(o.listeners, l)
}

// RemoveListener unregisters a listener.
func (o *Orchestrator) RemoveListener(l OrchestratorListener) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for i, listener := range o.listeners {
		if listener == l {
			o.listeners = append(o.listeners[:i], o.listeners[i+1:]...)
			return
		}
	}
}

// notifyListeners asynchronously notifies all listeners.
func (o *Orchestrator) notifyListeners(listeners []OrchestratorListener) {
	agents := o.GetAll()
	for _, l := range listeners {
		go func(listener OrchestratorListener) {
			defer func() {
				if rec := recover(); rec != nil {
					// Silently ignore panics from listeners
				}
			}()
			listener.OnAgentsChanged(agents)
		}(l)
	}
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
