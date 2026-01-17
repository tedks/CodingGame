package debug

import "fmt"

// BreakpointType indicates the type of breakpoint.
type BreakpointType int

const (
	// BreakpointLine is a standard line breakpoint.
	BreakpointLine BreakpointType = iota
	// BreakpointFunction breaks on function entry.
	BreakpointFunction
	// BreakpointConditional breaks when a condition is true.
	BreakpointConditional
	// BreakpointWatch breaks when a variable changes.
	BreakpointWatch
	// BreakpointLogpoint logs a message without stopping.
	BreakpointLogpoint
)

// String returns a human-readable breakpoint type name.
func (bt BreakpointType) String() string {
	switch bt {
	case BreakpointLine:
		return "Line"
	case BreakpointFunction:
		return "Function"
	case BreakpointConditional:
		return "Conditional"
	case BreakpointWatch:
		return "Watch"
	case BreakpointLogpoint:
		return "Logpoint"
	default:
		return "Unknown"
	}
}

// Breakpoint represents a breakpoint in the debug session.
type Breakpoint struct {
	// Identity
	ID   string         // Unique identifier
	Type BreakpointType // Type of breakpoint

	// Location
	File     string // Source file path
	Line     int    // Line number (0 for function breakpoints)
	Column   int    // Column (optional, 0 if not specified)
	Function string // Function name (for function breakpoints)

	// Condition (for conditional breakpoints)
	Condition string // Expression that must be true to trigger

	// Logpoint message (for logpoints)
	LogMessage string // Message to log (can include expressions like {var})

	// State
	Enabled  bool // Whether the breakpoint is active
	Verified bool // Whether the debugger has verified this breakpoint
	HitCount int  // Number of times this breakpoint was hit
}

// NewLineBreakpoint creates a line breakpoint.
func NewLineBreakpoint(id, file string, line int) *Breakpoint {
	return &Breakpoint{
		ID:      id,
		Type:    BreakpointLine,
		File:    file,
		Line:    line,
		Enabled: true,
	}
}

// NewFunctionBreakpoint creates a function entry breakpoint.
func NewFunctionBreakpoint(id, function string) *Breakpoint {
	return &Breakpoint{
		ID:       id,
		Type:     BreakpointFunction,
		Function: function,
		Enabled:  true,
	}
}

// NewConditionalBreakpoint creates a conditional breakpoint.
func NewConditionalBreakpoint(id, file string, line int, condition string) *Breakpoint {
	return &Breakpoint{
		ID:        id,
		Type:      BreakpointConditional,
		File:      file,
		Line:      line,
		Condition: condition,
		Enabled:   true,
	}
}

// NewWatchpoint creates a watch breakpoint that triggers on variable change.
func NewWatchpoint(id string, expression string) *Breakpoint {
	return &Breakpoint{
		ID:        id,
		Type:      BreakpointWatch,
		Condition: expression, // Expression to watch
		Enabled:   true,
	}
}

// NewLogpoint creates a logpoint that logs without stopping.
func NewLogpoint(id, file string, line int, message string) *Breakpoint {
	return &Breakpoint{
		ID:         id,
		Type:       BreakpointLogpoint,
		File:       file,
		Line:       line,
		LogMessage: message,
		Enabled:    true,
	}
}

// locationKey returns a string key for file:line lookup.
func (bp *Breakpoint) locationKey() string {
	return fmt.Sprintf("%s:%d", bp.File, bp.Line)
}

// Location returns a formatted location string.
func (bp *Breakpoint) Location() string {
	if bp.Type == BreakpointFunction {
		return bp.Function
	}
	if bp.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", bp.File, bp.Line, bp.Column)
	}
	return fmt.Sprintf("%s:%d", bp.File, bp.Line)
}

// Enable enables the breakpoint.
func (bp *Breakpoint) Enable() {
	bp.Enabled = true
}

// Disable disables the breakpoint.
func (bp *Breakpoint) Disable() {
	bp.Enabled = false
}

// Toggle toggles the breakpoint enabled state.
func (bp *Breakpoint) Toggle() {
	bp.Enabled = !bp.Enabled
}

// IncrementHitCount increments the hit counter.
func (bp *Breakpoint) IncrementHitCount() {
	bp.HitCount++
}
