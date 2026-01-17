// Package delve provides a delve-based debug adapter for Go programs.
//
// It implements the debug.Adapter interface using the delve debugger's JSON-RPC API.
// Delve is started in headless mode and controlled via RPC calls.
//
// Usage:
//
//	adapter := delve.NewAdapter()
//	session, err := adapter.Launch("./myprogram", []string{"arg1"}, ".")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer adapter.Terminate(session)
//
//	// Set breakpoints, continue, step, etc.
package delve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tedks/CodingGame/internal/debug"
)

// Adapter implements debug.Adapter for Go using delve.
type Adapter struct {
	mu sync.Mutex

	// Active sessions
	sessions map[string]*sessionState

	// ID generator
	nextID atomic.Int64
}

// sessionState tracks the state of a debug session.
type sessionState struct {
	session *debug.Session
	cmd     *exec.Cmd    // dlv process
	conn    net.Conn     // RPC connection
	rpcID   atomic.Int64 // RPC request ID counter
	port    int          // dlv API port
}

// NewAdapter creates a new delve adapter.
func NewAdapter() *Adapter {
	return &Adapter{
		sessions: make(map[string]*sessionState),
	}
}

// Name returns the adapter name.
func (a *Adapter) Name() string {
	return "delve"
}

// Language returns the supported language.
func (a *Adapter) Language() string {
	return "go"
}

// Launch starts a new debug session by launching a Go program.
func (a *Adapter) Launch(program string, args []string, cwd string) (*debug.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Build dlv command
	// dlv exec --headless --api-version=2 --listen=127.0.0.1:port -- program args...
	dlvArgs := []string{
		"exec",
		program,
		"--headless",
		"--api-version=2",
		fmt.Sprintf("--listen=127.0.0.1:%d", port),
		"--accept-multiclient",
	}
	if len(args) > 0 {
		dlvArgs = append(dlvArgs, "--")
		dlvArgs = append(dlvArgs, args...)
	}

	cmd := exec.Command("dlv", dlvArgs...)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dlv: %w", err)
	}

	// Give dlv time to start listening
	time.Sleep(500 * time.Millisecond)

	// Connect to dlv
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to dlv: %w", err)
	}

	// Create session
	sessionID := fmt.Sprintf("dlv-%d", a.nextID.Add(1))
	session := debug.NewSession(sessionID, "go", cwd)
	session.SetState(debug.StatePaused) // dlv starts paused at entry

	state := &sessionState{
		session: session,
		cmd:     cmd,
		conn:    conn,
		port:    port,
	}

	a.sessions[sessionID] = state

	return session, nil
}

// Attach attaches to an existing Go process.
func (a *Adapter) Attach(pid int) (*debug.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find an available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Build dlv command
	dlvArgs := []string{
		"attach",
		strconv.Itoa(pid),
		"--headless",
		"--api-version=2",
		fmt.Sprintf("--listen=127.0.0.1:%d", port),
		"--accept-multiclient",
	}

	cmd := exec.Command("dlv", dlvArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dlv: %w", err)
	}

	// Give dlv time to attach
	time.Sleep(500 * time.Millisecond)

	// Connect to dlv
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to connect to dlv: %w", err)
	}

	// Create session
	sessionID := fmt.Sprintf("dlv-%d", a.nextID.Add(1))
	session := debug.NewSession(sessionID, "go", "")
	session.SetState(debug.StatePaused)

	state := &sessionState{
		session: session,
		cmd:     cmd,
		conn:    conn,
		port:    port,
	}

	a.sessions[sessionID] = state

	return session, nil
}

// Connect connects to a remote delve server.
func (a *Adapter) Connect(address string) (*debug.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to dlv at %s: %w", address, err)
	}

	// Create session
	sessionID := fmt.Sprintf("dlv-%d", a.nextID.Add(1))
	session := debug.NewSession(sessionID, "go", "")
	session.SetState(debug.StatePaused)

	state := &sessionState{
		session: session,
		conn:    conn,
	}

	a.sessions[sessionID] = state

	return session, nil
}

// Disconnect ends the session without terminating the target.
func (a *Adapter) Disconnect(session *debug.Session) error {
	a.mu.Lock()
	state, ok := a.sessions[session.ID()]
	if !ok {
		a.mu.Unlock()
		return errors.New("session not found")
	}
	delete(a.sessions, session.ID())
	a.mu.Unlock()

	// Close connection
	if state.conn != nil {
		state.conn.Close()
	}

	// Detach from process (let it continue)
	if state.cmd != nil {
		// Send detach command via SIGHUP or just close
		state.cmd.Process.Signal(os.Interrupt)
	}

	session.SetState(debug.StateStopped)
	return nil
}

// Terminate ends the session and terminates the target.
func (a *Adapter) Terminate(session *debug.Session) error {
	a.mu.Lock()
	state, ok := a.sessions[session.ID()]
	if !ok {
		a.mu.Unlock()
		return errors.New("session not found")
	}
	delete(a.sessions, session.ID())
	a.mu.Unlock()

	// Send halt command to dlv
	if state.conn != nil {
		a.callRPC(state, "Command", map[string]interface{}{
			"name": "halt",
		})
		state.conn.Close()
	}

	// Kill dlv process
	if state.cmd != nil {
		state.cmd.Process.Kill()
		state.cmd.Wait()
	}

	session.SetState(debug.StateStopped)
	return nil
}

// Continue resumes execution.
func (a *Adapter) Continue(session *debug.Session) error {
	state := a.getState(session)
	if state == nil {
		return errors.New("session not found")
	}

	_, err := a.callRPC(state, "Command", map[string]interface{}{
		"name": "continue",
	})
	if err != nil {
		return err
	}

	session.SetState(debug.StateRunning)
	return nil
}

// Pause pauses execution.
func (a *Adapter) Pause(session *debug.Session) error {
	state := a.getState(session)
	if state == nil {
		return errors.New("session not found")
	}

	_, err := a.callRPC(state, "Command", map[string]interface{}{
		"name": "halt",
	})
	if err != nil {
		return err
	}

	session.SetState(debug.StatePaused)
	return nil
}

// StepOver executes the next statement.
func (a *Adapter) StepOver(session *debug.Session) error {
	state := a.getState(session)
	if state == nil {
		return errors.New("session not found")
	}

	_, err := a.callRPC(state, "Command", map[string]interface{}{
		"name": "next",
	})
	if err != nil {
		return err
	}

	// Update stack frames after step
	a.updateStackFrames(session, state)
	return nil
}

// StepInto steps into a function call.
func (a *Adapter) StepInto(session *debug.Session) error {
	state := a.getState(session)
	if state == nil {
		return errors.New("session not found")
	}

	_, err := a.callRPC(state, "Command", map[string]interface{}{
		"name": "step",
	})
	if err != nil {
		return err
	}

	a.updateStackFrames(session, state)
	return nil
}

// StepOut continues until the current function returns.
func (a *Adapter) StepOut(session *debug.Session) error {
	state := a.getState(session)
	if state == nil {
		return errors.New("session not found")
	}

	_, err := a.callRPC(state, "Command", map[string]interface{}{
		"name": "stepout",
	})
	if err != nil {
		return err
	}

	a.updateStackFrames(session, state)
	return nil
}

// SetBreakpoint sets a breakpoint.
func (a *Adapter) SetBreakpoint(session *debug.Session, bp *debug.Breakpoint) (*debug.Breakpoint, error) {
	state := a.getState(session)
	if state == nil {
		return nil, errors.New("session not found")
	}

	args := map[string]interface{}{
		"Breakpoint": map[string]interface{}{
			"file": bp.File,
			"line": bp.Line,
		},
	}

	if bp.Condition != "" {
		args["Breakpoint"].(map[string]interface{})["cond"] = bp.Condition
	}

	result, err := a.callRPC(state, "CreateBreakpoint", args)
	if err != nil {
		return nil, err
	}

	// Extract ID from result
	if bpResult, ok := result["Breakpoint"].(map[string]interface{}); ok {
		if id, ok := bpResult["id"].(float64); ok {
			bp.ID = fmt.Sprintf("bp-%d", int(id))
		}
		bp.Verified = true
	}

	session.AddBreakpoint(bp)
	return bp, nil
}

// ClearBreakpoint removes a breakpoint.
func (a *Adapter) ClearBreakpoint(session *debug.Session, bp *debug.Breakpoint) error {
	state := a.getState(session)
	if state == nil {
		return errors.New("session not found")
	}

	// Extract numeric ID from bp.ID
	idStr := strings.TrimPrefix(bp.ID, "bp-")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("invalid breakpoint ID: %s", bp.ID)
	}

	_, err = a.callRPC(state, "ClearBreakpoint", map[string]interface{}{
		"Id": id,
	})
	if err != nil {
		return err
	}

	session.RemoveBreakpoint(bp.ID)
	return nil
}

// GetStackFrames retrieves the current call stack.
func (a *Adapter) GetStackFrames(session *debug.Session) ([]*debug.Frame, error) {
	state := a.getState(session)
	if state == nil {
		return nil, errors.New("session not found")
	}

	result, err := a.callRPC(state, "Stacktrace", map[string]interface{}{
		"Id":    -1, // Current goroutine
		"Depth": 50, // Max frames
		"Full":  true,
	})
	if err != nil {
		return nil, err
	}

	frames := make([]*debug.Frame, 0)
	if locs, ok := result["Locations"].([]interface{}); ok {
		for i, loc := range locs {
			if locMap, ok := loc.(map[string]interface{}); ok {
				frame := a.convertLocation(i, locMap)
				frames = append(frames, frame)
			}
		}
	}

	session.SetFrames(frames)
	return frames, nil
}

// GetVariables retrieves variables for a scope.
func (a *Adapter) GetVariables(session *debug.Session, frameID int, scope string) ([]*debug.Variable, error) {
	state := a.getState(session)
	if state == nil {
		return nil, errors.New("session not found")
	}

	result, err := a.callRPC(state, "ListLocalVars", map[string]interface{}{
		"Scope": map[string]interface{}{
			"GoroutineID": -1,
			"Frame":       frameID,
		},
		"Cfg": map[string]interface{}{
			"MaxStringLen":       256,
			"MaxArrayValues":     64,
			"MaxStructFields":    -1,
			"MaxVariableRecurse": 1,
		},
	})
	if err != nil {
		return nil, err
	}

	variables := make([]*debug.Variable, 0)
	if vars, ok := result["Variables"].([]interface{}); ok {
		for _, v := range vars {
			if varMap, ok := v.(map[string]interface{}); ok {
				variable := a.convertVariable(varMap)
				variables = append(variables, variable)
			}
		}
	}

	return variables, nil
}

// Evaluate evaluates an expression.
func (a *Adapter) Evaluate(session *debug.Session, frameID int, expression string) (*debug.Variable, error) {
	state := a.getState(session)
	if state == nil {
		return nil, errors.New("session not found")
	}

	result, err := a.callRPC(state, "Eval", map[string]interface{}{
		"Scope": map[string]interface{}{
			"GoroutineID": -1,
			"Frame":       frameID,
		},
		"Expr": expression,
		"Cfg": map[string]interface{}{
			"MaxStringLen":       256,
			"MaxArrayValues":     64,
			"MaxStructFields":    -1,
			"MaxVariableRecurse": 2,
		},
	})
	if err != nil {
		return nil, err
	}

	if varMap, ok := result["Variable"].(map[string]interface{}); ok {
		return a.convertVariable(varMap), nil
	}

	return nil, errors.New("failed to evaluate expression")
}

// SetVariable sets a variable's value.
func (a *Adapter) SetVariable(session *debug.Session, frameID int, name, value string) (*debug.Variable, error) {
	state := a.getState(session)
	if state == nil {
		return nil, errors.New("session not found")
	}

	_, err := a.callRPC(state, "Set", map[string]interface{}{
		"Scope": map[string]interface{}{
			"GoroutineID": -1,
			"Frame":       frameID,
		},
		"Symbol": name,
		"Value":  value,
	})
	if err != nil {
		return nil, err
	}

	// Re-evaluate to get new value
	return a.Evaluate(session, frameID, name)
}

// SupportsDataFlow returns true (we can implement data flow tracking).
func (a *Adapter) SupportsDataFlow() bool {
	return true
}

// EnableDataFlow enables data flow recording.
func (a *Adapter) EnableDataFlow(session *debug.Session, enabled bool) error {
	// Data flow recording is always available via stepping and variable inspection
	return nil
}

// getState retrieves session state.
func (a *Adapter) getState(session *debug.Session) *sessionState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions[session.ID()]
}

// updateStackFrames updates the session's stack frames.
func (a *Adapter) updateStackFrames(session *debug.Session, state *sessionState) {
	frames, err := a.GetStackFrames(session)
	if err == nil && len(frames) > 0 {
		session.SetFrames(frames)
	}
}

// callRPC makes a JSON-RPC call to delve.
func (a *Adapter) callRPC(state *sessionState, method string, args interface{}) (map[string]interface{}, error) {
	id := state.rpcID.Add(1)

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "RPCServer." + method,
		"params":  []interface{}{args},
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	// Write request
	_, err = state.conn.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to send RPC request: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(state.conn)
	var response struct {
		JSONRPC string                 `json:"jsonrpc"`
		ID      int64                  `json:"id"`
		Result  map[string]interface{} `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := decoder.Decode(&response); err != nil {
		if err == io.EOF {
			return nil, errors.New("connection closed")
		}
		return nil, fmt.Errorf("failed to decode RPC response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", response.Error.Code, response.Error.Message)
	}

	return response.Result, nil
}

// convertLocation converts a dlv location to a Frame.
func (a *Adapter) convertLocation(id int, loc map[string]interface{}) *debug.Frame {
	frame := debug.NewFrame(id, "", "", 0)

	if fn, ok := loc["function"].(map[string]interface{}); ok {
		if name, ok := fn["name"].(string); ok {
			frame.Function = name
		}
	}

	if file, ok := loc["file"].(string); ok {
		frame.File = file
	}
	if line, ok := loc["line"].(float64); ok {
		frame.Line = int(line)
	}

	// Extract local variables if present
	if locals, ok := loc["Locals"].([]interface{}); ok {
		for _, v := range locals {
			if varMap, ok := v.(map[string]interface{}); ok {
				variable := a.convertVariable(varMap)
				frame.AddLocal(variable)
			}
		}
	}

	// Extract arguments if present
	if args, ok := loc["Arguments"].([]interface{}); ok {
		for _, v := range args {
			if varMap, ok := v.(map[string]interface{}); ok {
				variable := a.convertVariable(varMap)
				frame.AddArgument(variable)
			}
		}
	}

	return frame
}

// convertVariable converts a dlv variable to a Variable.
func (a *Adapter) convertVariable(v map[string]interface{}) *debug.Variable {
	name := ""
	if n, ok := v["name"].(string); ok {
		name = n
	}

	typ := ""
	if t, ok := v["type"].(string); ok {
		typ = t
	}

	value := ""
	if val, ok := v["value"].(string); ok {
		value = val
	}

	variable := debug.NewVariable(name, typ, value)

	// Check if value was truncated
	if len, ok := v["len"].(float64); ok {
		if cap, ok := v["cap"].(float64); ok {
			if int(len) > 0 || int(cap) > 0 {
				variable.ElementCount = int(len)
			}
		}
	}

	// Extract children
	if children, ok := v["children"].([]interface{}); ok {
		for _, child := range children {
			if childMap, ok := child.(map[string]interface{}); ok {
				childVar := a.convertVariable(childMap)
				variable.AddChild(childVar)
			}
		}
		if hasMore, ok := v["hasMore"].(bool); ok {
			variable.HasMore = hasMore
		}
	}

	// Check if pointer
	if addr, ok := v["addr"].(float64); ok {
		variable.Address = uint64(addr)
	}
	if kind, ok := v["kind"].(float64); ok {
		// Go reflect.Kind: 22 = Ptr
		variable.IsPointer = int(kind) == 22
	}

	return variable
}

// findAvailablePort finds an available TCP port.
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// Register the adapter with the default registry.
func init() {
	debug.DefaultRegistry.Register(NewAdapter())
}
