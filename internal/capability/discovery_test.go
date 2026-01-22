package capability

import (
	"testing"
)

func TestBuiltinToolDiscoverer(t *testing.T) {
	d := NewBuiltinToolDiscoverer()

	if d.Name() != "builtin-tools" {
		t.Errorf("expected name 'builtin-tools', got %q", d.Name())
	}

	caps, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	if len(caps) == 0 {
		t.Error("expected at least one built-in tool")
	}

	// Check for known built-in tools
	toolNames := make(map[string]bool)
	for _, cap := range caps {
		toolNames[cap.ID] = true
	}

	expectedTools := []string{"read", "write", "edit", "bash", "glob", "grep", "task"}
	for _, tool := range expectedTools {
		if !toolNames[tool] {
			t.Errorf("expected built-in tool %q not found", tool)
		}
	}
}

func TestBuiltinToolDiscovererWatchPaths(t *testing.T) {
	d := NewBuiltinToolDiscoverer()

	paths := d.WatchPaths()
	if paths != nil {
		t.Error("expected WatchPaths() to return nil for built-in tools")
	}
}

func TestBuiltinToolsHaveCorrectMetadata(t *testing.T) {
	d := NewBuiltinToolDiscoverer()
	caps, _ := d.Discover()

	for _, cap := range caps {
		if cap.Source != "builtin" {
			t.Errorf("tool %q has source %q, expected 'builtin'", cap.ID, cap.Source)
		}
		if cap.Type != TypeTool {
			t.Errorf("tool %q has type %v, expected TypeTool", cap.ID, cap.Type)
		}
		if !cap.Enabled {
			t.Errorf("tool %q should be enabled by default", cap.ID)
		}
	}
}
