package multiagent

import (
	"image/color"
	"testing"

	"github.com/tedks/CodingGame/internal/theme"
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
		expected color.RGBA
	}{
		{StatusIdle, theme.StatusNeutral},
		{StatusWorking, theme.StatusSuccess},
		{StatusPaused, theme.StatusWarning},
		{StatusCompleted, theme.StatusInfo},
		{StatusError, theme.StatusError},
	}

	for _, tc := range tests {
		c := GetStatusColor(tc.status)
		if c != tc.expected {
			t.Errorf("GetStatusColor(%v) = %v, expected %v", tc.status, c, tc.expected)
		}
	}
}

func TestLayoutConstants(t *testing.T) {
	// Verify layout constants are reasonable
	if theme.MultiAgentCardWidth <= 0 {
		t.Error("MultiAgentCardWidth should be positive")
	}
	if theme.MultiAgentCardHeight <= 0 {
		t.Error("MultiAgentCardHeight should be positive")
	}
	if theme.MultiAgentCardsPerRow <= 0 {
		t.Error("MultiAgentCardsPerRow should be positive")
	}
	if theme.PanelHeaderHeight <= 0 {
		t.Error("PanelHeaderHeight should be positive")
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

func TestClampUnitInterval(t *testing.T) {
	tests := []struct {
		value    float64
		expected float64
	}{
		{-0.5, 0},
		{0, 0},
		{0.35, 0.35},
		{1, 1},
		{1.25, 1},
	}

	for _, tc := range tests {
		if got := clampUnitInterval(tc.value); got != tc.expected {
			t.Errorf("clampUnitInterval(%v) = %v, want %v", tc.value, got, tc.expected)
		}
	}
}

func TestTokenSummaryXGuard(t *testing.T) {
	x := 5

	if _, ok := tokenSummaryX(x, 100); ok {
		t.Error("expected guard to skip token summary for narrow width")
	}

	width := 300
	want := x + width - theme.MultiAgentTokenSummaryWidth
	got, ok := tokenSummaryX(x, width)
	if !ok {
		t.Fatal("expected token summary to render for wide width")
	}
	if got != want {
		t.Errorf("tokenSummaryX = %d, want %d", got, want)
	}
}
