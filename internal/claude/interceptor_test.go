package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

	handlerCalled := make(chan struct{}, 1)
	handler := func(e *Event) {
		select {
		case handlerCalled <- struct{}{}:
		default:
		}
	}

	interceptor.AddHandler(handler)

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	interceptor.SimulateFileRead("/path/to/file.go")

	select {
	case <-handlerCalled:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("expected handler to be called within timeout")
	}
}

func TestStartStop(t *testing.T) {
	interceptor := New()

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}

	if !interceptor.running {
		t.Error("expected interceptor to be running after Start()")
	}

	if err := interceptor.Start(); err == nil {
		t.Error("expected error when starting already running interceptor")
	}

	if err := interceptor.Stop(); err != nil {
		t.Fatalf("failed to stop interceptor: %v", err)
	}

	if interceptor.running {
		t.Error("expected interceptor to not be running after Stop()")
	}
}

func TestEventTypes(t *testing.T) {
	interceptor := New()

	eventsCh := make(chan *Event, 3)
	handler := func(e *Event) {
		eventsCh <- e
	}

	interceptor.AddHandler(handler)

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	interceptor.SimulateFileRead("/file1.go")
	interceptor.SimulateFileWrite("/file2.go")
	interceptor.SimulateFileEdit("/file3.go")

	var receivedEvents []*Event
	for i := 0; i < 3; i++ {
		select {
		case e := <-eventsCh:
			receivedEvents = append(receivedEvents, e)
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for event %d, got %d events", i+1, len(receivedEvents))
		}
	}

	if receivedEvents[0].Type != EventFileRead {
		t.Errorf("expected first event to be FileRead, got %v", receivedEvents[0].Type)
	}
	if receivedEvents[1].Type != EventFileWrite {
		t.Errorf("expected second event to be FileWrite, got %v", receivedEvents[1].Type)
	}
	if receivedEvents[2].Type != EventFileEdit {
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
		{"file read", map[string]interface{}{"tool": "Read", "file_path": "/file.go"}, EventFileRead},
		{"file write", map[string]interface{}{"tool": "Write", "file_path": "/file.go"}, EventFileWrite},
		{"file edit", map[string]interface{}{"tool": "Edit", "file_path": "/file.go"}, EventFileEdit},
		{"build command", map[string]interface{}{"tool": "Bash", "command": "go build ./..."}, EventBuildRun},
		{"test command", map[string]interface{}{"tool": "Bash", "command": "go test ./..."}, EventTestRun},
		{"subagent", map[string]interface{}{"tool": "Task"}, EventSubagentRun},
		{"generic tool", map[string]interface{}{"tool": "SomeTool"}, EventToolUse},
		{"text content", map[string]interface{}{"text": "some text"}, EventText},
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

	handler1Called := make(chan struct{}, 1)
	handler2Called := make(chan struct{}, 1)

	interceptor.AddHandler(func(e *Event) {
		select {
		case handler1Called <- struct{}{}:
		default:
		}
	})
	interceptor.AddHandler(func(e *Event) {
		select {
		case handler2Called <- struct{}{}:
		default:
		}
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	interceptor.SimulateFileRead("/file.go")

	for i, ch := range []chan struct{}{handler1Called, handler2Called} {
		select {
		case <-ch:
		case <-time.After(1 * time.Second):
			t.Errorf("timeout waiting for handler%d to be called", i+1)
		}
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

// =============================================================================
// CRITICAL BUG FIX TESTS
// =============================================================================

// TestHandlerPanicRecovery verifies that a panicking handler doesn't kill
// the dispatch goroutine, allowing subsequent events to be delivered.
func TestHandlerPanicRecovery(t *testing.T) {
	interceptor := New()

	var callOrder []string
	var mu sync.Mutex
	done := make(chan struct{})

	interceptor.AddHandler(func(e *Event) {
		mu.Lock()
		callOrder = append(callOrder, "panic-handler")
		mu.Unlock()
		panic("intentional panic for testing")
	})

	interceptor.AddHandler(func(e *Event) {
		mu.Lock()
		callOrder = append(callOrder, "second-handler")
		mu.Unlock()
	})

	interceptor.AddHandler(func(e *Event) {
		mu.Lock()
		callOrder = append(callOrder, "third-handler")
		mu.Unlock()
		close(done)
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	interceptor.SimulateFileRead("/test.go")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for handlers - dispatch goroutine may have died")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(callOrder) < 3 {
		t.Errorf("expected 3 handlers called, got %d: %v", len(callOrder), callOrder)
	}
}

// TestEventsAfterPanic verifies events continue after a handler panics.
func TestEventsAfterPanic(t *testing.T) {
	interceptor := New()

	var eventCount int32
	panicOnFirst := true
	done := make(chan struct{})

	interceptor.AddHandler(func(e *Event) {
		count := atomic.AddInt32(&eventCount, 1)
		if panicOnFirst && count == 1 {
			panicOnFirst = false
			panic("first event panic")
		}
		if count == 3 {
			close(done)
		}
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	interceptor.SimulateFileRead("/file1.go")
	interceptor.SimulateFileRead("/file2.go")
	interceptor.SimulateFileRead("/file3.go")

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout: only %d events received", atomic.LoadInt32(&eventCount))
	}
}

// TestLongJSONLine verifies that long JSON lines (>64KB) are parsed correctly.
func TestLongJSONLine(t *testing.T) {
	interceptor := New()

	largeContent := strings.Repeat("x", 100*1024)
	jsonData := map[string]interface{}{
		"tool":      "Read",
		"file_path": "/large-file.txt",
		"content":   largeContent,
	}
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	if len(jsonBytes) <= 64*1024 {
		t.Fatalf("test JSON should be >64KB, got %d bytes", len(jsonBytes))
	}

	received := make(chan *Event, 1)
	interceptor.AddHandler(func(e *Event) {
		received <- e
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	reader := bytes.NewReader(append(jsonBytes, '\n'))
	go interceptor.parseOutput(reader)

	select {
	case event := <-received:
		if event.Type != EventFileRead {
			t.Errorf("expected EventFileRead, got %v", event.Type)
		}
		content, ok := event.Data["content"].(string)
		if !ok {
			t.Error("content field missing or not a string")
		} else if len(content) != len(largeContent) {
			t.Errorf("content length mismatch: got %d, want %d", len(content), len(largeContent))
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for large JSON event - buffer may be too small")
	}
}

// =============================================================================
// PROPERTY-BASED TESTS
// =============================================================================

func TestPropertyRoundtrip(t *testing.T) {
	testCases := []struct {
		eventType EventType
		data      map[string]interface{}
	}{
		{EventFileRead, map[string]interface{}{"tool": "Read", "file_path": "/a.go"}},
		{EventFileWrite, map[string]interface{}{"tool": "Write", "file_path": "/b.go"}},
		{EventFileEdit, map[string]interface{}{"tool": "Edit", "file_path": "/c.go"}},
	}

	for _, tc := range testCases {
		t.Run(string(tc.eventType), func(t *testing.T) {
			interceptor := New()

			received := make(chan *Event, 1)
			interceptor.AddHandler(func(e *Event) {
				received <- e
			})

			if err := interceptor.Start(); err != nil {
				t.Fatalf("failed to start: %v", err)
			}
			defer interceptor.Stop()

			interceptor.emitEvent(&Event{
				Type:      tc.eventType,
				Timestamp: time.Now(),
				Data:      tc.data,
			})

			select {
			case event := <-received:
				if event.Type != tc.eventType {
					t.Errorf("Type mismatch: got %v, want %v", event.Type, tc.eventType)
				}
				for k, v := range tc.data {
					if event.Data[k] != v {
						t.Errorf("Data[%q] mismatch: got %v, want %v", k, event.Data[k], v)
					}
				}
			case <-time.After(time.Second):
				t.Error("timeout waiting for event")
			}
		})
	}
}

func TestPropertyOrdering(t *testing.T) {
	interceptor := New()

	const numEvents = 100
	received := make(chan int, numEvents)

	interceptor.AddHandler(func(e *Event) {
		if seq, ok := e.Data["seq"].(int); ok {
			received <- seq
		}
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	for i := 0; i < numEvents; i++ {
		interceptor.emitEvent(&Event{
			Type:      EventToolUse,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"seq": i},
		})
	}

	for i := 0; i < numEvents; i++ {
		select {
		case seq := <-received:
			if seq != i {
				t.Errorf("event %d received out of order: got seq %d", i, seq)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for event %d", i)
		}
	}
}

func TestPropertyIdempotentStop(t *testing.T) {
	interceptor := New()

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	for i := 0; i < 10; i++ {
		err := interceptor.Stop()
		if err != nil {
			t.Errorf("Stop() returned error on call %d: %v", i, err)
		}
	}
}

// =============================================================================
// CONCURRENCY STRESS TESTS
// =============================================================================

func TestConcurrentAddHandler(t *testing.T) {
	interceptor := New()

	var handleCount int32
	baseHandler := func(e *Event) {
		atomic.AddInt32(&handleCount, 1)
	}
	interceptor.AddHandler(baseHandler)

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			interceptor.SimulateFileRead(fmt.Sprintf("/file%d.go", i))
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			interceptor.AddHandler(baseHandler)
			time.Sleep(10 * time.Microsecond)
		}
	}()

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	count := atomic.LoadInt32(&handleCount)
	if count == 0 {
		t.Error("no events were handled")
	}
}

func TestRapidStartStop(t *testing.T) {
	for i := 0; i < 100; i++ {
		interceptor := New()

		if err := interceptor.Start(); err != nil {
			t.Fatalf("iteration %d: failed to start: %v", i, err)
		}

		interceptor.SimulateFileRead("/test.go")

		if err := interceptor.Stop(); err != nil {
			t.Fatalf("iteration %d: failed to stop: %v", i, err)
		}
	}
}

func TestChannelBackpressure(t *testing.T) {
	interceptor := New()

	var processed int32
	interceptor.AddHandler(func(e *Event) {
		time.Sleep(time.Millisecond)
		atomic.AddInt32(&processed, 1)
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	const numEvents = 200
	for i := 0; i < numEvents; i++ {
		interceptor.SimulateFileRead(fmt.Sprintf("/file%d.go", i))
	}

	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt32(&processed) < numEvents && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	count := atomic.LoadInt32(&processed)
	if count < numEvents {
		t.Errorf("expected %d events processed, got %d", numEvents, count)
	}
}

func TestNoGoroutineLeaks(t *testing.T) {
	initialGoroutines := runtime.NumGoroutine()

	for i := 0; i < 10; i++ {
		interceptor := New()
		interceptor.AddHandler(func(e *Event) {})
		if err := interceptor.Start(); err != nil {
			t.Fatalf("failed to start: %v", err)
		}
		interceptor.SimulateFileRead("/test.go")
		if err := interceptor.Stop(); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	if finalGoroutines > initialGoroutines+5 {
		t.Errorf("goroutine leak detected: started with %d, ended with %d",
			initialGoroutines, finalGoroutines)
	}
}

// =============================================================================
// METAMORPHIC TESTS
// =============================================================================

func TestToolNameVariants(t *testing.T) {
	interceptor := New()

	readVariants := []string{"Read", "read", "read_file"}
	for _, variant := range readVariants {
		data := map[string]interface{}{"tool": variant}
		eventType := interceptor.inferEventType(data)
		if eventType != EventFileRead {
			t.Errorf("tool %q should map to EventFileRead, got %v", variant, eventType)
		}
	}

	writeVariants := []string{"Write", "write", "write_file"}
	for _, variant := range writeVariants {
		data := map[string]interface{}{"tool": variant}
		eventType := interceptor.inferEventType(data)
		if eventType != EventFileWrite {
			t.Errorf("tool %q should map to EventFileWrite, got %v", variant, eventType)
		}
	}

	editVariants := []string{"Edit", "edit", "edit_file"}
	for _, variant := range editVariants {
		data := map[string]interface{}{"tool": variant}
		eventType := interceptor.inferEventType(data)
		if eventType != EventFileEdit {
			t.Errorf("tool %q should map to EventFileEdit, got %v", variant, eventType)
		}
	}
}

func TestBuildTestClassification(t *testing.T) {
	interceptor := New()

	tests := []struct {
		command  string
		expected EventType
	}{
		{"bazel build //...", EventBuildRun},
		{"cargo build --release", EventBuildRun},
		{"make all", EventBuildRun},
		{"go build ./cmd/...", EventBuildRun},
		{"npm run build", EventBuildRun},
		{"bazel test //...", EventTestRun},
		{"pytest -v", EventTestRun},
		{"go test ./...", EventTestRun},
		{"npm test", EventTestRun},
		{"cargo test", EventTestRun},
		{"ls -la", EventToolUse},
		{"git status", EventToolUse},
	}

	for _, tc := range tests {
		data := map[string]interface{}{"tool": "Bash", "command": tc.command}
		eventType := interceptor.inferEventType(data)
		if eventType != tc.expected {
			t.Errorf("command %q: expected %v, got %v", tc.command, tc.expected, eventType)
		}
	}
}

// =============================================================================
// CHAOS TESTS
// =============================================================================

func TestParseOutputReaderClose(t *testing.T) {
	interceptor := New()

	received := make(chan *Event, 10)
	interceptor.AddHandler(func(e *Event) {
		received <- e
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	jsonLine := `{"tool": "Read", "file_path": "/test.go"}` + "\n"
	reader := strings.NewReader(jsonLine)

	done := make(chan struct{})
	go func() {
		interceptor.parseOutput(reader)
		close(done)
	}()

	select {
	case event := <-received:
		if event.Type != EventFileRead {
			t.Errorf("expected EventFileRead, got %v", event.Type)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("parseOutput didn't exit after reader closed")
	}
}

func TestMalformedJSONRecovery(t *testing.T) {
	interceptor := New()

	received := make(chan *Event, 10)
	interceptor.AddHandler(func(e *Event) {
		received <- e
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	input := `{"tool": "Read", "file_path": "/first.go"}
{this is not valid json}
{"tool": "Write", "file_path": "/second.go"}
truncated json {
{"tool": "Edit", "file_path": "/third.go"}
`

	go interceptor.parseOutput(strings.NewReader(input))

	var events []*Event
	timeout := time.After(2 * time.Second)
	for len(events) < 3 {
		select {
		case e := <-received:
			events = append(events, e)
		case <-timeout:
			t.Fatalf("timeout: only received %d events, expected 3", len(events))
		}
	}

	expectedTypes := []EventType{EventFileRead, EventFileWrite, EventFileEdit}
	for i, event := range events {
		if event.Type != expectedTypes[i] {
			t.Errorf("event %d: expected %v, got %v", i, expectedTypes[i], event.Type)
		}
	}
}

func TestTypeConfusionInJSON(t *testing.T) {
	interceptor := New()

	tests := []struct {
		name string
		json string
	}{
		{"tool as number", `{"tool": 123}`},
		{"tool as array", `{"tool": ["Read"]}`},
		{"tool as object", `{"tool": {"name": "Read"}}`},
		{"tool as null", `{"tool": null}`},
		{"command as number", `{"tool": "Bash", "command": 42}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(tc.json), &data); err != nil {
				t.Fatalf("test setup error: %v", err)
			}
			// Should not panic
			_ = interceptor.inferEventType(data)
		})
	}
}
