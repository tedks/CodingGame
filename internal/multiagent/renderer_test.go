package multiagent

import (
	"testing"
)

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	if r == nil {
		t.Fatal("NewRenderer returned nil")
	}
}

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		status   AgentStatus
		expected [4]uint8 // R, G, B, A
	}{
		{StatusIdle, [4]uint8{0x75, 0x75, 0x75, 0xFF}},      // Gray
		{StatusWorking, [4]uint8{0x2E, 0x7D, 0x32, 0xFF}},   // Green
		{StatusPaused, [4]uint8{0xF5, 0x7F, 0x17, 0xFF}},    // Orange
		{StatusCompleted, [4]uint8{0x19, 0x76, 0xD2, 0xFF}}, // Blue
		{StatusError, [4]uint8{0xC6, 0x28, 0x28, 0xFF}},     // Red
	}

	for _, tc := range tests {
		c := GetStatusColor(tc.status)
		if c.R != tc.expected[0] || c.G != tc.expected[1] || c.B != tc.expected[2] || c.A != tc.expected[3] {
			t.Errorf("GetStatusColor(%v) = (%d,%d,%d,%d), expected (%d,%d,%d,%d)",
				tc.status, c.R, c.G, c.B, c.A,
				tc.expected[0], tc.expected[1], tc.expected[2], tc.expected[3])
		}
	}
}

func TestLayoutConstants(t *testing.T) {
	// Verify layout constants are reasonable
	if agentCardWidth <= 0 {
		t.Error("agentCardWidth should be positive")
	}
	if agentCardHeight <= 0 {
		t.Error("agentCardHeight should be positive")
	}
	if agentCardsPerRow <= 0 {
		t.Error("agentCardsPerRow should be positive")
	}
	if agentHeaderHeight <= 0 {
		t.Error("agentHeaderHeight should be positive")
	}
}

func TestStatusColorsMapComplete(t *testing.T) {
	// Verify all statuses have colors defined
	statuses := []AgentStatus{
		StatusIdle,
		StatusWorking,
		StatusPaused,
		StatusCompleted,
		StatusError,
	}
	for _, status := range statuses {
		if _, ok := statusColors[status]; !ok {
			t.Errorf("statusColors missing entry for %v", status)
		}
	}
}

func TestGetUsageColor(t *testing.T) {
	tests := []struct {
		usage    float64
		expected string // Color name for readability
	}{
		{0.0, "green"},
		{0.4, "green"},
		{0.5, "orange"},
		{0.7, "orange"},
		{0.8, "red"},
		{1.0, "red"},
	}

	for _, tc := range tests {
		c := getUsageColor(tc.usage)
		// Just verify it returns a valid color (non-zero)
		if c.A == 0 {
			t.Errorf("getUsageColor(%f) returned transparent color", tc.usage)
		}
	}
}
