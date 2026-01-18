package debug

// Adapter defines the interface that debugger backends must implement.
//
// Different debuggers (delve for Go, pdb for Python, Chrome DevTools for TypeScript)
// implement this interface to provide consistent debugging capabilities to CodingGame.
type Adapter interface {
	// Name returns the adapter name (e.g., "delve", "pdb", "chrome-devtools").
	Name() string

	// Language returns the language this adapter supports.
	Language() string

	// Launch starts a new debug session by launching a program.
	//
	// Parameters:
	//   - program: Path to the executable or script
	//   - args: Command-line arguments
	//   - cwd: Working directory
	//
	// Returns a new Session or error.
	Launch(program string, args []string, cwd string) (*Session, error)

	// Attach attaches to an existing process.
	//
	// Parameters:
	//   - pid: Process ID to attach to
	//
	// Returns a new Session or error.
	Attach(pid int) (*Session, error)

	// Connect connects to a debug server (e.g., remote debugging).
	//
	// Parameters:
	//   - address: Server address (e.g., "localhost:8080")
	//
	// Returns a new Session or error.
	Connect(address string) (*Session, error)

	// Disconnect ends the debug session without terminating the target.
	Disconnect(session *Session) error

	// Terminate ends the debug session and terminates the target.
	Terminate(session *Session) error

	// Continue resumes execution until the next breakpoint or termination.
	Continue(session *Session) error

	// Pause pauses execution.
	Pause(session *Session) error

	// StepOver executes the next statement, stepping over function calls.
	StepOver(session *Session) error

	// StepInto executes the next statement, stepping into function calls.
	StepInto(session *Session) error

	// StepOut continues execution until the current function returns.
	StepOut(session *Session) error

	// SetBreakpoint sets a breakpoint and returns the verified breakpoint.
	SetBreakpoint(session *Session, bp *Breakpoint) (*Breakpoint, error)

	// ClearBreakpoint removes a breakpoint.
	ClearBreakpoint(session *Session, bp *Breakpoint) error

	// GetStackFrames retrieves the current call stack.
	GetStackFrames(session *Session) ([]*Frame, error)

	// GetVariables retrieves variables for a scope.
	//
	// Parameters:
	//   - session: The debug session
	//   - frameID: Frame to get variables for (0 = current)
	//   - scope: "arguments", "locals", or "all"
	//
	// Returns the variables in the specified scope.
	GetVariables(session *Session, frameID int, scope string) ([]*Variable, error)

	// Evaluate evaluates an expression in the current context.
	//
	// Parameters:
	//   - session: The debug session
	//   - frameID: Frame context for evaluation
	//   - expression: Expression to evaluate
	//
	// Returns a Variable containing the result.
	Evaluate(session *Session, frameID int, expression string) (*Variable, error)

	// SetVariable sets a variable's value.
	//
	// Parameters:
	//   - session: The debug session
	//   - frameID: Frame containing the variable
	//   - name: Variable name
	//   - value: New value (as expression)
	//
	// Returns the updated Variable.
	SetVariable(session *Session, frameID int, name, value string) (*Variable, error)

	// SupportsDataFlow returns true if this adapter supports data flow recording.
	SupportsDataFlow() bool

	// EnableDataFlow enables data flow recording for belt visualization.
	// Not all adapters support this feature.
	EnableDataFlow(session *Session, enabled bool) error
}

// AdapterCapabilities describes what features an adapter supports.
type AdapterCapabilities struct {
	// Basic capabilities
	SupportsLaunch     bool
	SupportsAttach     bool
	SupportsConnect    bool
	SupportsTerminate  bool
	SupportsDisconnect bool

	// Stepping capabilities
	SupportsStepOver bool
	SupportsStepInto bool
	SupportsStepOut  bool

	// Breakpoint capabilities
	SupportsConditionalBreakpoints bool
	SupportsLogpoints              bool
	SupportsWatchpoints            bool
	SupportsFunctionBreakpoints    bool

	// Variable capabilities
	SupportsSetVariable bool
	SupportsEvaluate    bool

	// Data flow capabilities
	SupportsDataFlow bool
}

// AdapterRegistry manages available debugger adapters.
type AdapterRegistry struct {
	adapters map[string]Adapter
}

// NewAdapterRegistry creates a new adapter registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]Adapter),
	}
}

// Register registers an adapter.
func (r *AdapterRegistry) Register(adapter Adapter) {
	r.adapters[adapter.Language()] = adapter
}

// Get returns an adapter for the given language.
func (r *AdapterRegistry) Get(language string) Adapter {
	return r.adapters[language]
}

// Languages returns all registered languages.
func (r *AdapterRegistry) Languages() []string {
	result := make([]string, 0, len(r.adapters))
	for lang := range r.adapters {
		result = append(result, lang)
	}
	return result
}

// DefaultRegistry is the global adapter registry.
var DefaultRegistry = NewAdapterRegistry()
