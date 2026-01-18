package debug

import "fmt"

// Frame represents a stack frame in the call stack.
type Frame struct {
	// Frame identity
	ID       int    // Frame number (0 = top of stack)
	Function string // Function name (e.g., "main.doWork")
	File     string // Source file path
	Line     int    // Current line number
	Column   int    // Current column (0 if unknown)

	// Variables in this frame
	Arguments []*Variable // Function arguments
	Locals    []*Variable // Local variables
	Returns   []*Variable // Return values (if available)
}

// NewFrame creates a new stack frame.
func NewFrame(id int, function, file string, line int) *Frame {
	return &Frame{
		ID:        id,
		Function:  function,
		File:      file,
		Line:      line,
		Arguments: make([]*Variable, 0),
		Locals:    make([]*Variable, 0),
		Returns:   make([]*Variable, 0),
	}
}

// Location returns a formatted file:line:column string.
func (f *Frame) Location() string {
	if f.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", f.File, f.Line, f.Column)
	}
	return fmt.Sprintf("%s:%d", f.File, f.Line)
}

// AddArgument adds an argument variable to the frame.
func (f *Frame) AddArgument(v *Variable) {
	f.Arguments = append(f.Arguments, v)
}

// AddLocal adds a local variable to the frame.
func (f *Frame) AddLocal(v *Variable) {
	f.Locals = append(f.Locals, v)
}

// AddReturn adds a return value to the frame.
func (f *Frame) AddReturn(v *Variable) {
	f.Returns = append(f.Returns, v)
}

// GetVariable searches all variable categories for a variable by name.
func (f *Frame) GetVariable(name string) *Variable {
	for _, v := range f.Arguments {
		if v.Name == name {
			return v
		}
	}
	for _, v := range f.Locals {
		if v.Name == name {
			return v
		}
	}
	for _, v := range f.Returns {
		if v.Name == name {
			return v
		}
	}
	return nil
}

// AllVariables returns all variables (arguments, locals, returns).
func (f *Frame) AllVariables() []*Variable {
	result := make([]*Variable, 0, len(f.Arguments)+len(f.Locals)+len(f.Returns))
	result = append(result, f.Arguments...)
	result = append(result, f.Locals...)
	result = append(result, f.Returns...)
	return result
}

// Variable represents a variable value in a debug session.
type Variable struct {
	// Identity
	Name string // Variable name
	Type string // Type name (e.g., "string", "int", "[]byte")

	// Value representation
	Value       string      // String representation of value
	FullValue   interface{} // Raw value (if available)
	IsTruncated bool        // True if Value was truncated

	// Structure (for complex types)
	Children     []*Variable // Child members (for structs, maps, slices)
	ElementCount int         // Number of elements (for collections)
	HasMore      bool        // True if there are more children not loaded

	// Source information
	Address    uint64 // Memory address (if available)
	IsPointer  bool   // True if this is a pointer
	PointsToID string // ID of referenced variable (for pointers)
}

// NewVariable creates a new variable.
func NewVariable(name, typ, value string) *Variable {
	return &Variable{
		Name:     name,
		Type:     typ,
		Value:    value,
		Children: make([]*Variable, 0),
	}
}

// AddChild adds a child variable (for structs, maps, etc.).
func (v *Variable) AddChild(child *Variable) {
	v.Children = append(v.Children, child)
}

// IsCollection returns true if this is a slice, array, map, or channel.
func (v *Variable) IsCollection() bool {
	// Simple heuristic based on type prefix
	if len(v.Type) >= 2 {
		prefix := v.Type[:2]
		if prefix == "[]" || prefix == "[" {
			return true
		}
	}
	if len(v.Type) >= 3 {
		prefix := v.Type[:3]
		if prefix == "map" {
			return true
		}
	}
	if len(v.Type) >= 4 {
		prefix := v.Type[:4]
		if prefix == "chan" {
			return true
		}
	}
	return false
}

// IsStruct returns true if this appears to be a struct type.
func (v *Variable) IsStruct() bool {
	return len(v.Children) > 0 && !v.IsCollection()
}

// ShortValue returns a truncated value string for display.
func (v *Variable) ShortValue(maxLen int) string {
	if len(v.Value) <= maxLen {
		return v.Value
	}
	return v.Value[:maxLen-3] + "..."
}
