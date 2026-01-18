package debug

import (
	"sync"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	s := NewSession("test-1", "go", "/project")

	if s.ID() != "test-1" {
		t.Errorf("expected ID 'test-1', got '%s'", s.ID())
	}
	if s.Language() != "go" {
		t.Errorf("expected language 'go', got '%s'", s.Language())
	}
	if s.State() != StateIdle {
		t.Errorf("expected state Idle, got '%s'", s.State())
	}
}

func TestSessionStateTransition(t *testing.T) {
	s := NewSession("test-1", "go", "/project")

	events := make([]*Event, 0)
	s.AddHandler(func(e *Event) {
		events = append(events, e)
	})

	s.SetState(StateRunning)
	if s.State() != StateRunning {
		t.Errorf("expected state Running, got '%s'", s.State())
	}

	s.SetState(StatePaused)
	if s.State() != StatePaused {
		t.Errorf("expected state Paused, got '%s'", s.State())
	}

	// Check events were emitted
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != EventStateChanged {
		t.Errorf("expected EventStateChanged, got %s", events[0].Type)
	}
}

func TestSessionFrames(t *testing.T) {
	s := NewSession("test-1", "go", "/project")

	frame1 := NewFrame(0, "main.doWork", "main.go", 42)
	frame2 := NewFrame(1, "main.main", "main.go", 10)

	s.SetFrames([]*Frame{frame1, frame2})

	frames := s.Frames()
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}

	current := s.CurrentFrame()
	if current != frame1 {
		t.Error("expected current frame to be frame1")
	}
	if current.Function != "main.doWork" {
		t.Errorf("expected function 'main.doWork', got '%s'", current.Function)
	}
}

func TestSessionBreakpoints(t *testing.T) {
	s := NewSession("test-1", "go", "/project")

	bp1 := NewLineBreakpoint("bp-1", "main.go", 42)
	bp2 := NewLineBreakpoint("bp-2", "main.go", 50)
	bp3 := NewLineBreakpoint("bp-3", "util.go", 10)

	s.AddBreakpoint(bp1)
	s.AddBreakpoint(bp2)
	s.AddBreakpoint(bp3)

	// Test GetBreakpoint
	got := s.GetBreakpoint("bp-1")
	if got != bp1 {
		t.Error("expected to get bp1")
	}

	// Test BreakpointsAt
	atMain42 := s.BreakpointsAt("main.go", 42)
	if len(atMain42) != 1 || atMain42[0] != bp1 {
		t.Error("expected to find bp1 at main.go:42")
	}

	// Test AllBreakpoints
	all := s.AllBreakpoints()
	if len(all) != 3 {
		t.Errorf("expected 3 breakpoints, got %d", len(all))
	}

	// Test RemoveBreakpoint
	if !s.RemoveBreakpoint("bp-1") {
		t.Error("expected RemoveBreakpoint to return true")
	}
	if s.GetBreakpoint("bp-1") != nil {
		t.Error("expected bp1 to be removed")
	}
	all = s.AllBreakpoints()
	if len(all) != 2 {
		t.Errorf("expected 2 breakpoints after removal, got %d", len(all))
	}
}

func TestSessionDataFlow(t *testing.T) {
	s := NewSession("test-1", "go", "/project")

	df1 := NewDataFlow(FlowInput, "main.doWork", Location{File: "main.go", Line: 42})
	df1.AddVariable(NewVariable("x", "int", "42"))

	df2 := NewDataFlow(FlowTransform, "main.doWork", Location{File: "main.go", Line: 43})
	df2.WithExpression("y := x * 2")
	df2.AddVariable(NewVariable("y", "int", "84"))

	s.RecordDataFlow(df1)
	s.RecordDataFlow(df2)

	flows := s.DataFlows()
	if len(flows) != 2 {
		t.Fatalf("expected 2 data flows, got %d", len(flows))
	}

	if flows[0].Type != FlowInput {
		t.Errorf("expected FlowInput, got %s", flows[0].Type)
	}
	if flows[1].Expression != "y := x * 2" {
		t.Errorf("expected expression 'y := x * 2', got '%s'", flows[1].Expression)
	}

	s.ClearDataFlows()
	flows = s.DataFlows()
	if len(flows) != 0 {
		t.Errorf("expected 0 data flows after clear, got %d", len(flows))
	}
}

func TestFrame(t *testing.T) {
	frame := NewFrame(0, "pkg.Func", "pkg/file.go", 42)
	frame.Column = 5

	if frame.Location() != "pkg/file.go:42:5" {
		t.Errorf("expected 'pkg/file.go:42:5', got '%s'", frame.Location())
	}

	arg := NewVariable("x", "int", "42")
	local := NewVariable("y", "string", "hello")
	ret := NewVariable("result", "bool", "true")

	frame.AddArgument(arg)
	frame.AddLocal(local)
	frame.AddReturn(ret)

	if frame.GetVariable("x") != arg {
		t.Error("expected to find argument x")
	}
	if frame.GetVariable("y") != local {
		t.Error("expected to find local y")
	}
	if frame.GetVariable("result") != ret {
		t.Error("expected to find return result")
	}
	if frame.GetVariable("nonexistent") != nil {
		t.Error("expected nil for nonexistent variable")
	}

	all := frame.AllVariables()
	if len(all) != 3 {
		t.Errorf("expected 3 variables, got %d", len(all))
	}
}

func TestVariable(t *testing.T) {
	v := NewVariable("data", "[]byte", `[72, 101, 108, 108, 111]`)
	v.ElementCount = 5
	v.IsTruncated = false

	if !v.IsCollection() {
		t.Error("expected []byte to be a collection")
	}
	if v.IsStruct() {
		t.Error("expected []byte not to be a struct")
	}

	// Test struct detection
	sv := NewVariable("person", "Person", "{Name: John, Age: 30}")
	sv.AddChild(NewVariable("Name", "string", "John"))
	sv.AddChild(NewVariable("Age", "int", "30"))

	if sv.IsCollection() {
		t.Error("expected Person not to be a collection")
	}
	if !sv.IsStruct() {
		t.Error("expected Person to be a struct")
	}

	// Test short value
	long := NewVariable("long", "string", "This is a very long string value that should be truncated")
	if short := long.ShortValue(20); short != "This is a very lo..." {
		t.Errorf("expected truncated string, got '%s'", short)
	}
}

func TestBreakpoint(t *testing.T) {
	// Line breakpoint
	bp := NewLineBreakpoint("bp-1", "main.go", 42)
	if bp.Type != BreakpointLine {
		t.Errorf("expected BreakpointLine, got %s", bp.Type)
	}
	if bp.Location() != "main.go:42" {
		t.Errorf("expected 'main.go:42', got '%s'", bp.Location())
	}
	if !bp.Enabled {
		t.Error("expected breakpoint to be enabled by default")
	}

	// Function breakpoint
	fbp := NewFunctionBreakpoint("bp-2", "main.doWork")
	if fbp.Type != BreakpointFunction {
		t.Errorf("expected BreakpointFunction, got %s", fbp.Type)
	}
	if fbp.Location() != "main.doWork" {
		t.Errorf("expected 'main.doWork', got '%s'", fbp.Location())
	}

	// Conditional breakpoint
	cbp := NewConditionalBreakpoint("bp-3", "main.go", 50, "x > 10")
	if cbp.Type != BreakpointConditional {
		t.Errorf("expected BreakpointConditional, got %s", cbp.Type)
	}
	if cbp.Condition != "x > 10" {
		t.Errorf("expected condition 'x > 10', got '%s'", cbp.Condition)
	}

	// Logpoint
	lp := NewLogpoint("bp-4", "main.go", 60, "Value is {x}")
	if lp.Type != BreakpointLogpoint {
		t.Errorf("expected BreakpointLogpoint, got %s", lp.Type)
	}
	if lp.LogMessage != "Value is {x}" {
		t.Errorf("expected message 'Value is {x}', got '%s'", lp.LogMessage)
	}

	// Toggle
	bp.Disable()
	if bp.Enabled {
		t.Error("expected breakpoint to be disabled")
	}
	bp.Enable()
	if !bp.Enabled {
		t.Error("expected breakpoint to be enabled")
	}
	bp.Toggle()
	if bp.Enabled {
		t.Error("expected breakpoint to be disabled after toggle")
	}

	// Hit count
	bp.IncrementHitCount()
	bp.IncrementHitCount()
	if bp.HitCount != 2 {
		t.Errorf("expected hit count 2, got %d", bp.HitCount)
	}
}

func TestDataFlow(t *testing.T) {
	loc := Location{File: "main.go", Line: 42}

	// Input flow
	input := NewDataFlow(FlowInput, "main.doWork", loc)
	input.AddVariable(NewVariable("x", "int", "42"))
	input.AddVariable(NewVariable("y", "int", "10"))

	if !input.IsInbound() {
		t.Error("expected FlowInput to be inbound")
	}
	if input.IsOutbound() {
		t.Error("expected FlowInput not to be outbound")
	}

	summary := input.Summary()
	if summary != "Arguments: x=42, y=10" {
		t.Errorf("unexpected summary: %s", summary)
	}

	// Transform flow
	transform := NewDataFlow(FlowTransform, "main.doWork", loc)
	transform.WithExpression("result := x + y")
	transform.AddVariable(NewVariable("result", "int", "52"))

	if !transform.IsInternal() {
		t.Error("expected FlowTransform to be internal")
	}

	// Call flow
	call := NewDataFlow(FlowCall, "main.doWork", loc)
	call.WithCalledFunction("fmt.Println")
	call.AddVariable(NewVariable("a", "string", "hello"))

	if !call.IsOutbound() {
		t.Error("expected FlowCall to be outbound")
	}

	callSummary := call.Summary()
	if callSummary != "Call: fmt.Println(a=hello)" {
		t.Errorf("unexpected call summary: %s", callSummary)
	}
}

func TestDataFlowRecorder(t *testing.T) {
	s := NewSession("test-1", "go", "/project")
	recorder := NewDataFlowRecorder(s)

	loc := Location{File: "main.go", Line: 42}

	// Record input
	recorder.RecordInput("main.doWork", loc, []*Variable{
		NewVariable("x", "int", "42"),
	})

	// Record assignment
	recorder.RecordAssignment("main.doWork", loc, NewVariable("y", "int", "84"))

	// Record transform
	recorder.RecordTransform("main.doWork", loc, "z := x * y", NewVariable("z", "int", "3528"))

	// Record call
	recorder.RecordCall("main.doWork", loc, "fmt.Println", []*Variable{
		NewVariable("msg", "string", "Result: 3528"),
	})

	// Record return
	recorder.RecordReturn("main.doWork", loc, "helper.Calc", []*Variable{
		NewVariable("result", "int", "100"),
	})

	// Record output
	recorder.RecordOutput("main.doWork", loc, []*Variable{
		NewVariable("result", "int", "3528"),
	})

	flows := s.DataFlows()
	if len(flows) != 6 {
		t.Fatalf("expected 6 flows, got %d", len(flows))
	}

	// Verify sequence
	for i, f := range flows {
		if f.Sequence != i {
			t.Errorf("expected sequence %d, got %d", i, f.Sequence)
		}
	}

	// Verify types
	expectedTypes := []FlowType{FlowInput, FlowAssignment, FlowTransform, FlowCall, FlowReturn, FlowOutput}
	for i, f := range flows {
		if f.Type != expectedTypes[i] {
			t.Errorf("flow %d: expected type %s, got %s", i, expectedTypes[i], f.Type)
		}
	}
}

func TestEvent(t *testing.T) {
	e := NewEvent(EventBreakpointHit, "session-1")
	e.WithData("file", "main.go")
	e.WithData("line", 42)
	e.WithData("enabled", true)

	if e.GetString("file") != "main.go" {
		t.Errorf("expected 'main.go', got '%s'", e.GetString("file"))
	}
	if e.GetInt("line") != 42 {
		t.Errorf("expected 42, got %d", e.GetInt("line"))
	}
	if !e.GetBool("enabled") {
		t.Error("expected true")
	}

	// Missing keys
	if e.GetString("missing") != "" {
		t.Error("expected empty string for missing key")
	}
	if e.GetInt("missing") != 0 {
		t.Error("expected 0 for missing key")
	}
	if e.GetBool("missing") != false {
		t.Error("expected false for missing key")
	}
}

func TestEventBus(t *testing.T) {
	bus := NewEventBus()

	received := make([]*Event, 0)
	bus.Subscribe(func(e *Event) {
		received = append(received, e)
	})

	e1 := NewEvent(EventStateChanged, "s1")
	e2 := NewEvent(EventBreakpointHit, "s1")

	bus.Publish(e1)
	bus.Publish(e2)

	if len(received) != 2 {
		t.Errorf("expected 2 events, got %d", len(received))
	}
}

func TestAdapterRegistry(t *testing.T) {
	reg := NewAdapterRegistry()

	// No adapter registered yet
	if reg.Get("go") != nil {
		t.Error("expected nil for unregistered language")
	}

	// Verify Languages returns empty
	langs := reg.Languages()
	if len(langs) != 0 {
		t.Errorf("expected 0 languages, got %d", len(langs))
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateIdle, "Idle"},
		{StateRunning, "Running"},
		{StatePaused, "Paused"},
		{StateStopped, "Stopped"},
		{State(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %s, want %s", tt.state, got, tt.want)
		}
	}
}

func TestSessionConcurrency(t *testing.T) {
	s := NewSession("test-1", "go", "/project")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// Add breakpoint
			bp := NewLineBreakpoint("bp-"+string(rune(n)), "main.go", n)
			s.AddBreakpoint(bp)

			// Record data flow
			df := NewDataFlow(FlowInput, "test", Location{File: "main.go", Line: n})
			s.RecordDataFlow(df)

			// Read state
			_ = s.State()
			_ = s.Frames()
			_ = s.DataFlows()

			// Toggle state
			if n%2 == 0 {
				s.SetState(StateRunning)
			} else {
				s.SetState(StatePaused)
			}

			time.Sleep(time.Microsecond)
		}(i)
	}

	wg.Wait()

	// Verify no panic occurred (test passes if we get here)
	flows := s.DataFlows()
	if len(flows) != 100 {
		t.Errorf("expected 100 data flows, got %d", len(flows))
	}
}
