package claude

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	interceptor := New()

	if interceptor == nil {
		t.Fatal("expected non-nil interceptor")
	}
	if interceptor.running {
		t.Error("expected interceptor to not be running initially")
	}
}

func TestAddHandler(t *testing.T) {
	interceptor := New()

	handlerCalled := false
	handler := func(e *Event) {
		handlerCalled = true
	}

	interceptor.AddHandler(handler)

	// Start interceptor and simulate an event
	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	// Simulate a file read event
	interceptor.SimulateFileRead("/path/to/file.go")

	// Wait for event to be processed
	time.Sleep(50 * time.Millisecond)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

func TestStartStop(t *testing.T) {
	interceptor := New()

	// Start interceptor
	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}

	if !interceptor.running {
		t.Error("expected interceptor to be running after Start()")
	}

	// Try starting again (should fail)
	if err := interceptor.Start(); err == nil {
		t.Error("expected error when starting already running interceptor")
	}

	// Stop interceptor
	if err := interceptor.Stop(); err != nil {
		t.Fatalf("failed to stop interceptor: %v", err)
	}

	if interceptor.running {
		t.Error("expected interceptor to not be running after Stop()")
	}
}

func TestEventTypes(t *testing.T) {
	interceptor := New()

	var receivedEvents []*Event
	var mu sync.Mutex

	handler := func(e *Event) {
		mu.Lock()
		defer mu.Unlock()
		receivedEvents = append(receivedEvents, e)
	}

	interceptor.AddHandler(handler)

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	// Simulate different event types
	interceptor.SimulateFileRead("/file1.go")
	interceptor.SimulateFileWrite("/file2.go")
	interceptor.SimulateFileEdit("/file3.go")

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedEvents) != 3 {
		t.Errorf("expected 3 events, got %d", len(receivedEvents))
	}

	// Verify event types
	if len(receivedEvents) >= 1 && receivedEvents[0].Type != EventFileRead {
		t.Errorf("expected first event to be FileRead, got %v", receivedEvents[0].Type)
	}
	if len(receivedEvents) >= 2 && receivedEvents[1].Type != EventFileWrite {
		t.Errorf("expected second event to be FileWrite, got %v", receivedEvents[1].Type)
	}
	if len(receivedEvents) >= 3 && receivedEvents[2].Type != EventFileEdit {
		t.Errorf("expected third event to be FileEdit, got %v", receivedEvents[2].Type)
	}
}

func TestInferEventType(t *testing.T) {
	interceptor := New()

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected EventType
	}{
		{
			name:     "file read",
			data:     map[string]interface{}{"tool": "Read", "file_path": "/file.go"},
			expected: EventFileRead,
		},
		{
			name:     "file write",
			data:     map[string]interface{}{"tool": "Write", "file_path": "/file.go"},
			expected: EventFileWrite,
		},
		{
			name:     "file edit",
			data:     map[string]interface{}{"tool": "Edit", "file_path": "/file.go"},
			expected: EventFileEdit,
		},
		{
			name:     "build command",
			data:     map[string]interface{}{"tool": "Bash", "command": "go build ./..."},
			expected: EventBuildRun,
		},
		{
			name:     "test command",
			data:     map[string]interface{}{"tool": "Bash", "command": "go test ./..."},
			expected: EventTestRun,
		},
		{
			name:     "subagent",
			data:     map[string]interface{}{"tool": "Task"},
			expected: EventSubagentRun,
		},
		{
			name:     "generic tool",
			data:     map[string]interface{}{"tool": "SomeTool"},
			expected: EventToolUse,
		},
		{
			name:     "text content",
			data:     map[string]interface{}{"text": "some text"},
			expected: EventText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventType := interceptor.inferEventType(tt.data)
			if eventType != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, eventType)
			}
		})
	}
}

func TestMultipleHandlers(t *testing.T) {
	interceptor := New()

	var handler1Called, handler2Called bool
	var mu sync.Mutex

	handler1 := func(e *Event) {
		mu.Lock()
		defer mu.Unlock()
		handler1Called = true
	}

	handler2 := func(e *Event) {
		mu.Lock()
		defer mu.Unlock()
		handler2Called = true
	}

	interceptor.AddHandler(handler1)
	interceptor.AddHandler(handler2)

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	// Simulate event
	interceptor.SimulateFileRead("/file.go")

	// Wait for event to be processed
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !handler1Called {
		t.Error("expected handler1 to be called")
	}
	if !handler2Called {
		t.Error("expected handler2 to be called")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		s        string
		substrs  []string
		expected bool
	}{
		{"go build ./...", []string{"build", "test"}, true},
		{"go test ./...", []string{"build", "test"}, true},
		{"npm install", []string{"build", "test"}, false},
		{"cargo build --release", []string{"cargo", "build"}, true},
		{"", []string{"test"}, false},
	}

	for _, tt := range tests {
		result := containsAny(tt.s, tt.substrs)
		if result != tt.expected {
			t.Errorf("containsAny(%q, %v) = %v, expected %v",
				tt.s, tt.substrs, result, tt.expected)
		}
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Hello", "hello"},
		{"hello", "hello"},
		{"GoLang123", "golang123"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toLower(tt.input)
		if result != tt.expected {
			t.Errorf("toLower(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
