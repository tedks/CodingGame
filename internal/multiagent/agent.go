// Package multiagent provides multi-agent orchestration for CodingGame.
// It enables multiple Claude agents to work in parallel, each with their own
// context boundaries (fog of war) and task assignments.
//
// This follows the project philosophy: everything shown is real. Agents represent
// actual Claude instances, context is actual token usage, and tasks are real work.
package multiagent

import (
	"fmt"
	"sync"
	"time"
)

// DefaultTokenLimit is the default context window size for Claude models.
// Different models have different limits (e.g., claude-3-opus: 200k, claude-3-sonnet: 200k).
// This default should be updated when model context sizes change.
const DefaultTokenLimit int64 = 200000

// AgentStatus represents the current status of an agent.
type AgentStatus string

const (
	// StatusIdle means the agent is waiting for a task.
	StatusIdle AgentStatus = "idle"
	// StatusWorking means the agent is actively executing a task.
	StatusWorking AgentStatus = "working"
	// StatusPaused means the agent is paused (context preserved).
	StatusPaused AgentStatus = "paused"
	// StatusCompleted means the agent completed its task.
	StatusCompleted AgentStatus = "completed"
	// StatusError means the agent encountered an error.
	StatusError AgentStatus = "error"
)

// Agent represents an active Claude agent with its own context boundary.
// Each agent maintains its own "fog of war" - the set of files it has read.
//
// # Concurrency
//
// mu protects all fields on Agent. The id, name, and icon are immutable after
// construction; all other fields are mutable and guarded by mu.
//
// # State machine
//
// Idle -> Working -> Paused -> Working
// Working/Paused -> Completed
// Any state -> Error (SetError)
// Reset returns the agent to Idle and clears context.
//
// StartTask only transitions to Working from non-Working states, PauseTask only
// transitions from Working, and ResumeTask only transitions from Paused.
type Agent struct {
	mu sync.RWMutex

	// Identity
	id   string
	name string
	icon string

	// State
	status       AgentStatus
	currentTask  string
	lastActivity time.Time

	// Context boundary (fog of war)
	// Files the agent has read are "visible" in its context
	filesRead map[string]time.Time

	// Token tracking (real API usage)
	tokensUsed   int64
	tokenLimit   int64
	contextUsage float64 // 0.0 to 1.0, percentage of context filled

	// Position on map (for multi-agent visualization)
	x float64
	y float64

	// Error tracking
	lastError error
}

// AgentSnapshot is an immutable, point-in-time view of agent state.
// Modifying the snapshot does not affect the underlying agent.
type AgentSnapshot struct {
	ID           string
	Name         string
	Icon         string
	Status       AgentStatus
	CurrentTask  string
	LastActivity time.Time
	TokensUsed   int64
	TokenLimit   int64
	ContextUsage float64
	FileCount    int
	PositionX    float64
	PositionY    float64
}

// NewAgent creates a new agent with the given parameters.
func NewAgent(id, name, icon string) *Agent {
	return &Agent{
		id:           id,
		name:         name,
		icon:         icon,
		status:       StatusIdle,
		filesRead:    make(map[string]time.Time),
		lastActivity: time.Now(),
		tokenLimit:   DefaultTokenLimit,
	}
}

// ID returns the agent's unique identifier.
func (a *Agent) ID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.id
}

// Name returns the agent's display name.
func (a *Agent) Name() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.name
}

// Icon returns the agent's icon identifier.
func (a *Agent) Icon() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.icon
}

// Snapshot returns an immutable, point-in-time view of agent state.
func (a *Agent) Snapshot() AgentSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return AgentSnapshot{
		ID:           a.id,
		Name:         a.name,
		Icon:         a.icon,
		Status:       a.status,
		CurrentTask:  a.currentTask,
		LastActivity: a.lastActivity,
		TokensUsed:   a.tokensUsed,
		TokenLimit:   a.tokenLimit,
		ContextUsage: a.contextUsage,
		FileCount:    len(a.filesRead),
		PositionX:    a.x,
		PositionY:    a.y,
	}
}

// Status returns the agent's current status.
func (a *Agent) Status() AgentStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// CurrentTask returns the agent's current task description.
func (a *Agent) CurrentTask() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentTask
}

// Position returns the agent's map coordinates.
func (a *Agent) Position() (x, y float64) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.x, a.y
}

// SetPosition updates the agent's map coordinates.
func (a *Agent) SetPosition(x, y float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.x = x
	a.y = y
}

// FilesRead returns a copy of files this agent has read.
func (a *Agent) FilesRead() map[string]time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()

	files := make(map[string]time.Time, len(a.filesRead))
	for k, v := range a.filesRead {
		files[k] = v
	}
	return files
}

// HasRead returns whether the agent has read a specific file.
func (a *Agent) HasRead(filePath string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, exists := a.filesRead[filePath]
	return exists
}

// MarkFileRead records that the agent has read a file.
func (a *Agent) MarkFileRead(filePath string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.filesRead[filePath] = time.Now()
	a.lastActivity = time.Now()
}

// ContextUsage returns the percentage of context window used (0.0 to 1.0).
func (a *Agent) ContextUsage() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.contextUsage
}

// TokensUsed returns the number of tokens used by this agent.
func (a *Agent) TokensUsed() int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tokensUsed
}

// TokenLimit returns the agent's token limit.
func (a *Agent) TokenLimit() int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.tokenLimit
}

// SetTokenLimit sets the agent's token limit.
func (a *Agent) SetTokenLimit(limit int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tokenLimit = limit
}

// UpdateTokenUsage updates the agent's token usage metrics.
func (a *Agent) UpdateTokenUsage(tokensUsed int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tokensUsed = tokensUsed
	if a.tokenLimit > 0 {
		a.contextUsage = float64(tokensUsed) / float64(a.tokenLimit)
	} else {
		// If no limit is set, treat as 0% usage (no artificial constraint)
		a.contextUsage = 0.0
	}
	a.lastActivity = time.Now()
}

// LastActivity returns when the agent was last active.
func (a *Agent) LastActivity() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastActivity
}

// LastError returns the agent's last error.
func (a *Agent) LastError() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastError
}

// StartTask marks the agent as working on a task.
// Returns an error if the agent is already working (for atomic check-and-start).
func (a *Agent) StartTask(taskDescription string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.status == StatusWorking {
		return fmt.Errorf("agent %s is already working on: %s", a.id, a.currentTask)
	}
	a.status = StatusWorking
	a.currentTask = taskDescription
	a.lastActivity = time.Now()
	a.lastError = nil
	return nil
}

// CompleteTask marks the agent's current task as completed.
func (a *Agent) CompleteTask() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = StatusCompleted
	a.currentTask = ""
	a.lastActivity = time.Now()
}

// PauseTask pauses the agent while preserving context.
func (a *Agent) PauseTask() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.status == StatusWorking {
		a.status = StatusPaused
		a.lastActivity = time.Now()
	}
}

// ResumeTask resumes a paused agent.
func (a *Agent) ResumeTask() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.status == StatusPaused {
		a.status = StatusWorking
		a.lastActivity = time.Now()
	}
}

// SetError marks the agent as having encountered an error.
func (a *Agent) SetError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = StatusError
	a.lastError = err
	a.lastActivity = time.Now()
}

// Reset resets the agent to idle state, clearing context.
func (a *Agent) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = StatusIdle
	a.currentTask = ""
	a.filesRead = make(map[string]time.Time)
	a.tokensUsed = 0
	a.contextUsage = 0
	a.lastError = nil
	a.lastActivity = time.Now()
}

// FileCount returns the number of files in the agent's context.
func (a *Agent) FileCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.filesRead)
}
