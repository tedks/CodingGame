package debug

import "time"

// FlowType indicates the type of data flow event.
type FlowType int

const (
	// FlowInput indicates data flowing into a function (arguments).
	FlowInput FlowType = iota
	// FlowOutput indicates data flowing out of a function (return values).
	FlowOutput
	// FlowAssignment indicates a variable assignment within a function.
	FlowAssignment
	// FlowTransform indicates a value transformation (computation).
	FlowTransform
	// FlowCall indicates data flowing into a called function.
	FlowCall
	// FlowReturn indicates data flowing back from a called function.
	FlowReturn
)

// String returns a human-readable flow type name.
func (ft FlowType) String() string {
	switch ft {
	case FlowInput:
		return "Input"
	case FlowOutput:
		return "Output"
	case FlowAssignment:
		return "Assignment"
	case FlowTransform:
		return "Transform"
	case FlowCall:
		return "Call"
	case FlowReturn:
		return "Return"
	default:
		return "Unknown"
	}
}

// Location represents a source code location.
type Location struct {
	File   string // Source file path
	Line   int    // Line number
	Column int    // Column number (0 if unknown)
}

// DataFlow represents a data flow event for belt visualization.
//
// Data flows model "items on a belt" flowing through functions:
// - Inputs: Arguments entering a function
// - Operations: Variable transformations within the function
// - Outputs: Return values leaving the function
type DataFlow struct {
	// Flow metadata
	Type      FlowType  // Type of data flow
	Timestamp time.Time // When this flow occurred
	Sequence  int       // Ordering within the session

	// Source and target
	Function string   // Function where this flow occurred
	Location Location // Source location of the flow

	// Data involved
	Variables []*Variable // Variables involved in this flow

	// Expression (for transforms)
	Expression string // The expression being evaluated (if applicable)

	// Call information (for FlowCall and FlowReturn)
	CalledFunction string // Name of the function being called/returned from

	// Belt visualization hints
	ConnectionID string // ID of the connection in the dependency graph (if applicable)
}

// NewDataFlow creates a new data flow event.
func NewDataFlow(flowType FlowType, function string, loc Location) *DataFlow {
	return &DataFlow{
		Type:      flowType,
		Timestamp: time.Now(),
		Function:  function,
		Location:  loc,
		Variables: make([]*Variable, 0),
	}
}

// AddVariable adds a variable to this data flow.
func (df *DataFlow) AddVariable(v *Variable) {
	df.Variables = append(df.Variables, v)
}

// WithExpression sets the expression for transform flows.
func (df *DataFlow) WithExpression(expr string) *DataFlow {
	df.Expression = expr
	return df
}

// WithCalledFunction sets the called function for call/return flows.
func (df *DataFlow) WithCalledFunction(fn string) *DataFlow {
	df.CalledFunction = fn
	return df
}

// WithConnection links this flow to a dependency connection for belt rendering.
func (df *DataFlow) WithConnection(connID string) *DataFlow {
	df.ConnectionID = connID
	return df
}

// IsInbound returns true if data is flowing into a function.
func (df *DataFlow) IsInbound() bool {
	return df.Type == FlowInput || df.Type == FlowReturn
}

// IsOutbound returns true if data is flowing out of a function.
func (df *DataFlow) IsOutbound() bool {
	return df.Type == FlowOutput || df.Type == FlowCall
}

// IsInternal returns true if data is moving within a function.
func (df *DataFlow) IsInternal() bool {
	return df.Type == FlowAssignment || df.Type == FlowTransform
}

// Summary returns a brief description of this data flow.
func (df *DataFlow) Summary() string {
	switch df.Type {
	case FlowInput:
		return "Arguments: " + df.variableList()
	case FlowOutput:
		return "Returns: " + df.variableList()
	case FlowAssignment:
		if len(df.Variables) > 0 {
			return df.Variables[0].Name + " = " + df.Variables[0].Value
		}
		return "Assignment"
	case FlowTransform:
		if df.Expression != "" {
			return df.Expression
		}
		return "Transform: " + df.variableList()
	case FlowCall:
		return "Call: " + df.CalledFunction + "(" + df.variableList() + ")"
	case FlowReturn:
		return "Return from: " + df.CalledFunction + " = " + df.variableList()
	default:
		return df.Type.String()
	}
}

// variableList returns a comma-separated list of variable names and values.
func (df *DataFlow) variableList() string {
	if len(df.Variables) == 0 {
		return "(none)"
	}
	result := ""
	for i, v := range df.Variables {
		if i > 0 {
			result += ", "
		}
		result += v.Name + "=" + v.ShortValue(20)
	}
	return result
}

// DataFlowRecorder is a helper for building data flow sequences.
type DataFlowRecorder struct {
	session  *Session
	sequence int
}

// NewDataFlowRecorder creates a recorder that adds flows to a session.
func NewDataFlowRecorder(session *Session) *DataFlowRecorder {
	return &DataFlowRecorder{
		session:  session,
		sequence: 0,
	}
}

// RecordInput records function inputs (arguments).
func (r *DataFlowRecorder) RecordInput(function string, loc Location, args []*Variable) {
	df := NewDataFlow(FlowInput, function, loc)
	df.Sequence = r.sequence
	r.sequence++
	for _, arg := range args {
		df.AddVariable(arg)
	}
	r.session.RecordDataFlow(df)
}

// RecordOutput records function outputs (return values).
func (r *DataFlowRecorder) RecordOutput(function string, loc Location, returns []*Variable) {
	df := NewDataFlow(FlowOutput, function, loc)
	df.Sequence = r.sequence
	r.sequence++
	for _, ret := range returns {
		df.AddVariable(ret)
	}
	r.session.RecordDataFlow(df)
}

// RecordAssignment records a variable assignment.
func (r *DataFlowRecorder) RecordAssignment(function string, loc Location, variable *Variable) {
	df := NewDataFlow(FlowAssignment, function, loc)
	df.Sequence = r.sequence
	r.sequence++
	df.AddVariable(variable)
	r.session.RecordDataFlow(df)
}

// RecordTransform records a transformation with an expression.
func (r *DataFlowRecorder) RecordTransform(function string, loc Location, expression string, result *Variable) {
	df := NewDataFlow(FlowTransform, function, loc)
	df.Sequence = r.sequence
	r.sequence++
	df.Expression = expression
	if result != nil {
		df.AddVariable(result)
	}
	r.session.RecordDataFlow(df)
}

// RecordCall records a function call with arguments.
func (r *DataFlowRecorder) RecordCall(function string, loc Location, calledFn string, args []*Variable) {
	df := NewDataFlow(FlowCall, function, loc)
	df.Sequence = r.sequence
	r.sequence++
	df.CalledFunction = calledFn
	for _, arg := range args {
		df.AddVariable(arg)
	}
	r.session.RecordDataFlow(df)
}

// RecordReturn records a return from a called function.
func (r *DataFlowRecorder) RecordReturn(function string, loc Location, calledFn string, returns []*Variable) {
	df := NewDataFlow(FlowReturn, function, loc)
	df.Sequence = r.sequence
	r.sequence++
	df.CalledFunction = calledFn
	for _, ret := range returns {
		df.AddVariable(ret)
	}
	r.session.RecordDataFlow(df)
}
