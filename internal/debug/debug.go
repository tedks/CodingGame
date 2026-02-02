// Package debug provides visual debugging infrastructure for CodingGame.
//
// It implements a language-agnostic debugging model that can track data flow
// through functions, supporting the belt visualization metaphor. Different
// debugger backends (Go/delve, Python/pdb, TypeScript/debugger) can plug into
// this interface.
//
// The core abstraction is a Session, which manages:
// - Stack frames and their variables
// - Breakpoints and stepping
// - Data flow events for belt visualization
//
// Data flow is modeled as "items on a belt" flowing through functions:
// - Inputs: Arguments entering a function
// - Operations: Variable transformations within the function
// - Outputs: Return values leaving the function
package debug

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// State represents the current state of a debug session.
type State int

const (
	// StateIdle indicates the debugger is not attached.
	StateIdle State = iota
	// StateRunning indicates the program is executing.
	StateRunning
	// StatePaused indicates execution is paused (breakpoint, step, etc.).
	StatePaused
	// StateStopped indicates the program has terminated.
	StateStopped
)

// String returns a human-readable state name.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateRunning:
		return "Running"
	case StatePaused:
		return "Paused"
	case StateStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// ValidTransitions defines the allowed state transitions.
var ValidTransitions = map[State][]State{
	StateIdle:    {StateRunning, StateStopped},
	StateRunning: {StatePaused, StateStopped},
	StatePaused:  {StateRunning, StateStopped},
	StateStopped: {},
}

// ErrInvalidTransition is returned when an invalid state transition is attempted.
var ErrInvalidTransition = errors.New("invalid state transition")

// ValidateTransition checks if a state transition is valid.
func ValidateTransition(from, to State) error {
	if from == to {
		return nil
	}
	for _, valid := range ValidTransitions[from] {
		if valid == to {
			return nil
		}
	}
	return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidTransition, from, to)
}

// HandlerID uniquely identifies an event handler.
type HandlerID int64

type handlerEntry struct {
	id      HandlerID
	handler EventHandler
}

var handlerIDCounter atomic.Int64

// Session represents an active debugging session.
//
// # Concurrency
//
// mu protects all fields on Session. Event handlers are invoked without holding mu;
// handlers must be thread-safe and should avoid long-running work.
//
// # State machine
//
// Idle -> Running -> Paused -> Running
// Running/Paused -> Stopped
// Stopped is terminal. SetState does not enforce transitions; callers must
// follow the expected sequence.
//
// Thread-safe: All methods are safe for concurrent access.
type Session struct {
	mu sync.RWMutex

	// Session metadata
	id        string
	language  string // "go", "python", "typescript"
	targetDir string // Project directory being debugged
	startTime time.Time

	// Session state
	state     State
	frames    []*Frame    // Call stack (top of stack first)
	dataFlows []*DataFlow // Data flow events for belt visualization

	// Breakpoint management
	breakpoints map[string]*Breakpoint   // Keyed by ID
	byLocation  map[string][]*Breakpoint // Indexed by file:line

	// Event handlers with IDs for removal
	handlers []handlerEntry
}

// SessionSnapshot is an immutable, point-in-time view of session state.
// Modifying the snapshot does not affect the underlying session.
type SessionSnapshot struct {
	ID          string
	Language    string
	TargetDir   string
	StartTime   time.Time
	State       State
	Frames      []*Frame
	DataFlows   []*DataFlow
	Breakpoints []*Breakpoint
}

// NewSession creates a new debug session.
//
// Parameters:
//   - id: Unique session identifier
//   - language: Programming language ("go", "python", "typescript")
//   - targetDir: Directory of the project being debugged
func NewSession(id, language, targetDir string) *Session {
	return &Session{
		id:          id,
		language:    language,
		targetDir:   targetDir,
		startTime:   time.Now(),
		state:       StateIdle,
		frames:      make([]*Frame, 0),
		dataFlows:   make([]*DataFlow, 0),
		breakpoints: make(map[string]*Breakpoint),
		byLocation:  make(map[string][]*Breakpoint),
		handlers:    make([]handlerEntry, 0),
	}
}

// ID returns the session identifier.
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// Language returns the programming language.
func (s *Session) Language() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.language
}

// State returns the current session state.
func (s *Session) State() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Snapshot returns an immutable, point-in-time view of session state.
func (s *Session) Snapshot() SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SessionSnapshot{
		ID:          s.id,
		Language:    s.language,
		TargetDir:   s.targetDir,
		StartTime:   s.startTime,
		State:       s.state,
		Frames:      cloneFrames(s.frames),
		DataFlows:   cloneDataFlows(s.dataFlows),
		Breakpoints: cloneBreakpoints(s.breakpoints),
	}
}

// SetState updates the session state and notifies handlers.
func (s *Session) SetState(state State) {
	s.mu.Lock()
	oldState := s.state
	s.state = state
	handlers := s.copyHandlers()
	s.mu.Unlock()

	if oldState != state {
		event := &Event{
			Type:      EventStateChanged,
			SessionID: s.id,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"old_state": oldState.String(),
				"new_state": state.String(),
			},
		}
		for _, h := range handlers {
			h(event)
		}
	}
}

// copyHandlers returns handlers as EventHandler slice. Must hold mu.
func (s *Session) copyHandlers() []EventHandler {
	handlers := make([]EventHandler, len(s.handlers))
	for i, e := range s.handlers {
		handlers[i] = e.handler
	}
	return handlers
}

// Frames returns the current call stack (top of stack first).
// Returns a copy; modifying it doesn't affect the session.
func (s *Session) Frames() []*Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneFrames(s.frames)
}

// CurrentFrame returns the topmost stack frame, or nil if none.
func (s *Session) CurrentFrame() *Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.frames) == 0 {
		return nil
	}
	return cloneFrame(s.frames[0])
}

// SetFrames updates the call stack.
func (s *Session) SetFrames(frames []*Frame) {
	s.mu.Lock()
	s.frames = cloneFrames(frames)
	handlers := s.copyHandlers()
	frameCount := len(s.frames)
	s.mu.Unlock()

	event := &Event{
		Type:      EventStackChanged,
		SessionID: s.id,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"frame_count": frameCount,
		},
	}
	for _, h := range handlers {
		h(event)
	}
}

// AddHandler registers an event handler. Returns ID for removal.
func (s *Session) AddHandler(handler EventHandler) HandlerID {
	id := HandlerID(handlerIDCounter.Add(1))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handlerEntry{id: id, handler: handler})
	return id
}

// RemoveHandler removes an event handler by ID.
func (s *Session) RemoveHandler(id HandlerID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.handlers {
		if e.id == id {
			s.handlers[i] = s.handlers[len(s.handlers)-1]
			s.handlers = s.handlers[:len(s.handlers)-1]
			return true
		}
	}
	return false
}

// HandlerCount returns the number of registered handlers.
func (s *Session) HandlerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handlers)
}

// AddBreakpoint adds a breakpoint to the session.
func (s *Session) AddBreakpoint(bp *Breakpoint) {
	s.mu.Lock()
	s.breakpoints[bp.ID] = bp
	locKey := bp.locationKey()
	s.byLocation[locKey] = append(s.byLocation[locKey], bp)
	handlers := s.copyHandlers()
	s.mu.Unlock()

	event := &Event{
		Type:      EventBreakpointSet,
		SessionID: s.id,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"breakpoint_id": bp.ID,
			"file":          bp.File,
			"line":          bp.Line,
		},
	}
	for _, h := range handlers {
		h(event)
	}
}

// RemoveBreakpoint removes a breakpoint by ID.
func (s *Session) RemoveBreakpoint(id string) bool {
	s.mu.Lock()
	bp, ok := s.breakpoints[id]
	if !ok {
		s.mu.Unlock()
		return false
	}

	delete(s.breakpoints, id)
	locKey := bp.locationKey()
	bps := s.byLocation[locKey]
	for i, b := range bps {
		if b.ID == id {
			s.byLocation[locKey] = append(bps[:i], bps[i+1:]...)
			break
		}
	}
	handlers := s.copyHandlers()
	s.mu.Unlock()

	event := &Event{
		Type:      EventBreakpointRemoved,
		SessionID: s.id,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"breakpoint_id": id,
		},
	}
	for _, h := range handlers {
		h(event)
	}
	return true
}

// GetBreakpoint returns a breakpoint by ID.
func (s *Session) GetBreakpoint(id string) *Breakpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.breakpoints[id]
}

// BreakpointsAt returns all breakpoints at a file:line location.
func (s *Session) BreakpointsAt(file string, line int) []*Breakpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bp := &Breakpoint{File: file, Line: line}
	locKey := bp.locationKey()
	result := make([]*Breakpoint, len(s.byLocation[locKey]))
	copy(result, s.byLocation[locKey])
	return result
}

// AllBreakpoints returns all breakpoints.
func (s *Session) AllBreakpoints() []*Breakpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Breakpoint, 0, len(s.breakpoints))
	for _, bp := range s.breakpoints {
		result = append(result, bp)
	}
	return result
}

// RecordDataFlow records a data flow event for belt visualization.
func (s *Session) RecordDataFlow(df *DataFlow) {
	s.mu.Lock()
	s.dataFlows = append(s.dataFlows, df)
	handlers := s.copyHandlers()
	s.mu.Unlock()

	event := &Event{
		Type:      EventDataFlow,
		SessionID: s.id,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"flow_type": df.Type.String(),
			"file":      df.Location.File,
			"line":      df.Location.Line,
			"function":  df.Function,
		},
	}
	for _, h := range handlers {
		h(event)
	}
}

// DataFlows returns all recorded data flow events.
func (s *Session) DataFlows() []*DataFlow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DataFlow, len(s.dataFlows))
	copy(result, s.dataFlows)
	return result
}

// ClearDataFlows clears all recorded data flow events.
func (s *Session) ClearDataFlows() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dataFlows = make([]*DataFlow, 0)
}
