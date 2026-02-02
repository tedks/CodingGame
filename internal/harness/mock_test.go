package harness

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMockHarness_SendPrompt(t *testing.T) {
	mock := NewMockHarness()

	// Should fail when not running
	if err := mock.SendPrompt("test"); err == nil {
		t.Error("SendPrompt should fail when not running")
	}

	// Start the harness
	config := NewConfig("/tmp")
	if err := mock.Start(context.Background(), config); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Now prompts should be recorded
	if err := mock.SendPrompt("prompt 1"); err != nil {
		t.Errorf("SendPrompt failed: %v", err)
	}
	if err := mock.SendPrompt("prompt 2"); err != nil {
		t.Errorf("SendPrompt failed: %v", err)
	}

	// Verify prompts were recorded
	if mock.PromptCount() != 2 {
		t.Errorf("PromptCount = %d, want 2", mock.PromptCount())
	}
	if mock.LastPrompt() != "prompt 2" {
		t.Errorf("LastPrompt = %q, want 'prompt 2'", mock.LastPrompt())
	}

	prompts := mock.Prompts()
	if len(prompts) != 2 || prompts[0] != "prompt 1" || prompts[1] != "prompt 2" {
		t.Errorf("Prompts = %v, want [prompt 1, prompt 2]", prompts)
	}

	// Stop and verify prompts fail again
	if err := mock.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if err := mock.SendPrompt("should fail"); err == nil {
		t.Error("SendPrompt should fail after Stop")
	}

	if err := mock.Start(context.Background(), config); err == nil {
		t.Error("Start should fail after Stop")
	}
}

func TestMockHarness_SimulateEvents(t *testing.T) {
	mock := NewMockHarness()

	config := NewConfig("/tmp")
	if err := mock.Start(context.Background(), config); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Simulate a text event
	go mock.SimulateText("Hello from mock")

	event := <-mock.Events()
	if event.Type != EventText {
		t.Errorf("Type = %v, want EventText", event.Type)
	}
	if event.Text != "Hello from mock" {
		t.Errorf("Text = %q, want 'Hello from mock'", event.Text)
	}

	// Simulate turn complete
	go mock.SimulateTurnComplete()

	event = <-mock.Events()
	if event.Type != EventTurnComplete {
		t.Errorf("Type = %v, want EventTurnComplete", event.Type)
	}
}

func TestMockHarness_ImplementsHarness(t *testing.T) {
	// Verify MockHarness implements Harness interface
	var _ Harness = NewMockHarness()
}

// --- P2 MockHarness Fidelity Tests ---

// TestMockHarness_SimulateEventAfterStop tests that simulating events after stop is safe
func TestMockHarness_SimulateEventAfterStop(t *testing.T) {
	mock := NewMockHarness()

	config := NewConfig("/tmp")
	if err := mock.Start(context.Background(), config); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if err := mock.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Simulating event after stop should be safe (no panic)
	// The current implementation checks IsRunning() and EventsClosed()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SimulateEvent after Stop caused panic: %v", r)
		}
	}()

	mock.SimulateEvent(NewEvent(EventText).WithText("test").WithSource("mock").Build())
	mock.SimulateText("text after stop")
	mock.SimulateTurnComplete()

	// All should complete without panic
}

// TestMockHarness_FidelityChecklist tests that MockHarness has same edge case behavior as real harness
func TestMockHarness_FidelityChecklist(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "start twice returns error",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				config := NewConfig("/tmp")

				if err := mock.Start(context.Background(), config); err != nil {
					t.Fatalf("First Start failed: %v", err)
				}

				if err := mock.Start(context.Background(), config); err == nil {
					t.Error("Second Start should return error")
				}
			},
		},
		{
			name: "stop before start succeeds",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				// Stop before Start should not error
				if err := mock.Stop(); err != nil {
					t.Errorf("Stop before Start should succeed: %v", err)
				}
			},
		},
		{
			name: "send prompt before start fails",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				if err := mock.SendPrompt("test"); err == nil {
					t.Error("SendPrompt before Start should fail")
				}
			},
		},
		{
			name: "send prompt after stop fails",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				config := NewConfig("/tmp")
				mock.Start(context.Background(), config)
				mock.Stop()

				if err := mock.SendPrompt("test"); err == nil {
					t.Error("SendPrompt after Stop should fail")
				}
			},
		},
		{
			name: "events channel closed after stop",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				config := NewConfig("/tmp")
				mock.Start(context.Background(), config)
				mock.Stop()

				select {
				case _, ok := <-mock.Events():
					if ok {
						t.Error("Events channel should be closed after Stop")
					}
				default:
					t.Error("Events channel should be closed and readable")
				}
			},
		},
		{
			name: "multiple stop calls are safe",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				config := NewConfig("/tmp")
				mock.Start(context.Background(), config)

				for i := 0; i < 5; i++ {
					if err := mock.Stop(); err != nil {
						t.Errorf("Stop %d failed: %v", i, err)
					}
				}
			},
		},
		{
			name: "capabilities are non-empty",
			test: func(t *testing.T) {
				mock := NewMockHarness()
				caps := mock.Capabilities()

				if len(caps.SupportedModels) == 0 {
					t.Error("MockHarness should have at least one supported model")
				}
			},
		},
		{
			name: "prompts cleared on new instance",
			test: func(t *testing.T) {
				mock := NewMockHarness()

				if mock.PromptCount() != 0 {
					t.Error("New MockHarness should have 0 prompts")
				}
				if mock.LastPrompt() != "" {
					t.Error("New MockHarness LastPrompt should be empty")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.test)
	}
}

// TestMockHarness_ConcurrentSimulation tests that concurrent event simulation is safe
func TestMockHarness_ConcurrentSimulation(t *testing.T) {
	mock := NewMockHarness()
	config := NewConfig("/tmp")
	if err := mock.Start(context.Background(), config); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	const numGoroutines = 10
	const eventsPerGoroutine = 10

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				mock.SimulateText(fmt.Sprintf("goroutine-%d-event-%d", n, j))
			}
		}(i)
	}

	// Drain events while they're being sent
	drainDone := make(chan struct{})
	go func() {
		count := 0
		for range mock.Events() {
			count++
			if count >= numGoroutines*eventsPerGoroutine {
				break
			}
		}
		close(drainDone)
	}()

	wg.Wait()

	// Give drain a moment to finish
	select {
	case <-drainDone:
		// Success
	case <-time.After(2 * time.Second):
		t.Error("Timeout draining events")
	}

	if err := mock.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
