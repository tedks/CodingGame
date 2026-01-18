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
	"sync"
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

// Session represents an active debugging session.
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

	// Event handlers
	handlers []EventHandler
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
		handlers:    make([]EventHandler, 0),
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

// SetState updates the session state and notifies handlers.
func (s *Session) SetState(state State) {
	s.mu.Lock()
	oldState := s.state
	s.state = state
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
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

// Frames returns the current call stack (top of stack first).
// Returns a copy; modifying it doesn't affect the session.
func (s *Session) Frames() []*Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Frame, len(s.frames))
	copy(result, s.frames)
	return result
}

// CurrentFrame returns the topmost stack frame, or nil if none.
func (s *Session) CurrentFrame() *Frame {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.frames) == 0 {
		return nil
	}
	return s.frames[0]
}

// SetFrames updates the call stack.
func (s *Session) SetFrames(frames []*Frame) {
	s.mu.Lock()
	s.frames = frames
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.Unlock()

	event := &Event{
		Type:      EventStackChanged,
		SessionID: s.id,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"frame_count": len(frames),
		},
	}
	for _, h := range handlers {
		h(event)
	}
}

// AddHandler registers an event handler for debug events.
func (s *Session) AddHandler(handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// AddBreakpoint adds a breakpoint to the session.
func (s *Session) AddBreakpoint(bp *Breakpoint) {
	s.mu.Lock()
	s.breakpoints[bp.ID] = bp
	locKey := bp.locationKey()
	s.byLocation[locKey] = append(s.byLocation[locKey], bp)
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
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
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
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
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
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
