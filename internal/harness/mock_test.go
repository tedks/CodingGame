package harness

import (
	"context"
	"testing"
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
