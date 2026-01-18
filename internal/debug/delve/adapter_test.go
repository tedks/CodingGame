package delve

import (
	"testing"

	"github.com/tedks/CodingGame/internal/debug"
)

func TestNewAdapter(t *testing.T) {
	adapter := NewAdapter()
	if adapter == nil {
		t.Fatal("NewAdapter returned nil")
	}
}

func TestAdapterName(t *testing.T) {
	adapter := NewAdapter()
	if adapter.Name() != "delve" {
		t.Errorf("expected name 'delve', got '%s'", adapter.Name())
	}
}

func TestAdapterLanguage(t *testing.T) {
	adapter := NewAdapter()
	if adapter.Language() != "go" {
		t.Errorf("expected language 'go', got '%s'", adapter.Language())
	}
}

func TestAdapterSupportsDataFlow(t *testing.T) {
	adapter := NewAdapter()
	if !adapter.SupportsDataFlow() {
		t.Error("expected SupportsDataFlow to return true")
	}
}

func TestAdapterRegistered(t *testing.T) {
	// The adapter should be auto-registered via init()
	adapter := debug.DefaultRegistry.Get("go")
	if adapter == nil {
		t.Fatal("expected Go adapter to be registered")
	}
	if adapter.Name() != "delve" {
		t.Errorf("expected delve adapter, got '%s'", adapter.Name())
	}
}

func TestConvertVariable(t *testing.T) {
	adapter := NewAdapter()

	// Test simple variable conversion
	varMap := map[string]interface{}{
		"name":  "x",
		"type":  "int",
		"value": "42",
	}

	v := adapter.convertVariable(varMap)
	if v.Name != "x" {
		t.Errorf("expected name 'x', got '%s'", v.Name)
	}
	if v.Type != "int" {
		t.Errorf("expected type 'int', got '%s'", v.Type)
	}
	if v.Value != "42" {
		t.Errorf("expected value '42', got '%s'", v.Value)
	}
}

func TestConvertVariableWithChildren(t *testing.T) {
	adapter := NewAdapter()

	varMap := map[string]interface{}{
		"name":  "person",
		"type":  "Person",
		"value": "{...}",
		"children": []interface{}{
			map[string]interface{}{
				"name":  "Name",
				"type":  "string",
				"value": "John",
			},
			map[string]interface{}{
				"name":  "Age",
				"type":  "int",
				"value": "30",
			},
		},
		"hasMore": false,
	}

	v := adapter.convertVariable(varMap)
	if len(v.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(v.Children))
	}
	if v.Children[0].Name != "Name" {
		t.Errorf("expected first child 'Name', got '%s'", v.Children[0].Name)
	}
	if v.Children[1].Name != "Age" {
		t.Errorf("expected second child 'Age', got '%s'", v.Children[1].Name)
	}
}

func TestConvertVariableWithCollection(t *testing.T) {
	adapter := NewAdapter()

	varMap := map[string]interface{}{
		"name":  "items",
		"type":  "[]string",
		"value": "[a, b, c]",
		"len":   float64(3),
		"cap":   float64(4),
	}

	v := adapter.convertVariable(varMap)
	if v.ElementCount != 3 {
		t.Errorf("expected element count 3, got %d", v.ElementCount)
	}
}

func TestConvertVariablePointer(t *testing.T) {
	adapter := NewAdapter()

	varMap := map[string]interface{}{
		"name":  "ptr",
		"type":  "*int",
		"value": "0xc0000100a0",
		"addr":  float64(0xc0000100a0),
		"kind":  float64(22), // reflect.Ptr
	}

	v := adapter.convertVariable(varMap)
	if !v.IsPointer {
		t.Error("expected IsPointer to be true")
	}
	if v.Address != 0xc0000100a0 {
		t.Errorf("expected address 0xc0000100a0, got 0x%x", v.Address)
	}
}

func TestConvertLocation(t *testing.T) {
	adapter := NewAdapter()

	loc := map[string]interface{}{
		"function": map[string]interface{}{
			"name": "main.doWork",
		},
		"file": "/project/main.go",
		"line": float64(42),
	}

	frame := adapter.convertLocation(0, loc)
	if frame.Function != "main.doWork" {
		t.Errorf("expected function 'main.doWork', got '%s'", frame.Function)
	}
	if frame.File != "/project/main.go" {
		t.Errorf("expected file '/project/main.go', got '%s'", frame.File)
	}
	if frame.Line != 42 {
		t.Errorf("expected line 42, got %d", frame.Line)
	}
}

func TestConvertLocationWithVariables(t *testing.T) {
	adapter := NewAdapter()

	loc := map[string]interface{}{
		"function": map[string]interface{}{
			"name": "main.doWork",
		},
		"file": "/project/main.go",
		"line": float64(42),
		"Arguments": []interface{}{
			map[string]interface{}{
				"name":  "x",
				"type":  "int",
				"value": "10",
			},
		},
		"Locals": []interface{}{
			map[string]interface{}{
				"name":  "y",
				"type":  "int",
				"value": "20",
			},
		},
	}

	frame := adapter.convertLocation(0, loc)
	if len(frame.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(frame.Arguments))
	}
	if frame.Arguments[0].Name != "x" {
		t.Errorf("expected argument 'x', got '%s'", frame.Arguments[0].Name)
	}
	if len(frame.Locals) != 1 {
		t.Fatalf("expected 1 local, got %d", len(frame.Locals))
	}
	if frame.Locals[0].Name != "y" {
		t.Errorf("expected local 'y', got '%s'", frame.Locals[0].Name)
	}
}

func TestFindAvailablePort(t *testing.T) {
	port, err := findAvailablePort()
	if err != nil {
		t.Fatalf("findAvailablePort failed: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("expected valid port, got %d", port)
	}
}

func TestSessionNotFoundErrors(t *testing.T) {
	adapter := NewAdapter()
	fakeSession := debug.NewSession("fake", "go", "/")

	// All operations should return errors for unknown sessions
	if err := adapter.Terminate(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if err := adapter.Disconnect(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if err := adapter.Continue(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if err := adapter.Pause(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if err := adapter.StepOver(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if err := adapter.StepInto(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if err := adapter.StepOut(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if _, err := adapter.GetStackFrames(fakeSession); err == nil {
		t.Error("expected error for unknown session")
	}
	if _, err := adapter.GetVariables(fakeSession, 0, "all"); err == nil {
		t.Error("expected error for unknown session")
	}
	if _, err := adapter.Evaluate(fakeSession, 0, "x"); err == nil {
		t.Error("expected error for unknown session")
	}
	if _, err := adapter.SetVariable(fakeSession, 0, "x", "1"); err == nil {
		t.Error("expected error for unknown session")
	}
}

// Integration tests that require dlv to be installed
// These are skipped if dlv is not available

func TestLaunchRequiresDlv(t *testing.T) {
	// This test verifies the launch behavior when dlv is not available
	adapter := NewAdapter()

	// Try to launch a non-existent program
	_, err := adapter.Launch("/nonexistent/program", nil, "/tmp")
	if err == nil {
		t.Error("expected error for non-existent program or missing dlv")
	}
}
