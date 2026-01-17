package input

import (
	"os"
	"testing"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// If no display is available, skip all tests in this package
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		println("Skipping input tests: no display available (DISPLAY and WAYLAND_DISPLAY not set)")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode     Mode
		expected string
	}{
		{ModeNormal, "NORMAL"},
		{ModeInsert, "INSERT"},
		{ModeVisual, "VISUAL"},
		{Mode(999), "UNKNOWN"},
	}

	for _, tc := range tests {
		result := tc.mode.String()
		if result != tc.expected {
			t.Errorf("Mode(%d).String() = %q, want %q", tc.mode, result, tc.expected)
		}
	}
}

func TestFocusArea_String(t *testing.T) {
	tests := []struct {
		focus    FocusArea
		expected string
	}{
		{FocusMap, "Map"},
		{FocusPrompt, "Prompt"},
		{FocusAdvisors, "Advisors"},
		{FocusMissions, "Missions"},
		{FocusResponse, "Response"},
		{FocusArea(999), "Unknown"},
	}

	for _, tc := range tests {
		result := tc.focus.String()
		if result != tc.expected {
			t.Errorf("FocusArea(%d).String() = %q, want %q", tc.focus, result, tc.expected)
		}
	}
}

func TestNextFocus(t *testing.T) {
	// Test cycling through focus areas
	tests := []struct {
		current  FocusArea
		expected FocusArea
	}{
		{FocusMap, FocusPrompt},
		{FocusPrompt, FocusAdvisors},
		{FocusAdvisors, FocusMissions},
		{FocusMissions, FocusResponse},
		{FocusResponse, FocusMap}, // Wrap around
	}

	for _, tc := range tests {
		result := NextFocus(tc.current)
		if result != tc.expected {
			t.Errorf("NextFocus(%v) = %v, want %v", tc.current, result, tc.expected)
		}
	}
}

func TestPrevFocus(t *testing.T) {
	// Test cycling backwards through focus areas
	tests := []struct {
		current  FocusArea
		expected FocusArea
	}{
		{FocusMap, FocusResponse},     // Wrap around backwards
		{FocusPrompt, FocusMap},
		{FocusAdvisors, FocusPrompt},
		{FocusMissions, FocusAdvisors},
		{FocusResponse, FocusMissions},
	}

	for _, tc := range tests {
		result := PrevFocus(tc.current)
		if result != tc.expected {
			t.Errorf("PrevFocus(%v) = %v, want %v", tc.current, result, tc.expected)
		}
	}
}

func TestNextFocus_InvalidArea(t *testing.T) {
	// Invalid focus area should default to FocusMap
	result := NextFocus(FocusArea(999))
	if result != FocusMap {
		t.Errorf("NextFocus(invalid) = %v, want %v", result, FocusMap)
	}
}

func TestPrevFocus_InvalidArea(t *testing.T) {
	// Invalid focus area should default to FocusMap
	result := PrevFocus(FocusArea(999))
	if result != FocusMap {
		t.Errorf("PrevFocus(invalid) = %v, want %v", result, FocusMap)
	}
}

func TestViewNumber_Constants(t *testing.T) {
	// Verify view numbers are 1-5 as expected
	if ViewMap != 1 {
		t.Errorf("ViewMap = %d, want 1", ViewMap)
	}
	if ViewBuilding != 2 {
		t.Errorf("ViewBuilding = %d, want 2", ViewBuilding)
	}
	if ViewUnit != 3 {
		t.Errorf("ViewUnit = %d, want 3", ViewUnit)
	}
	if ViewTech != 4 {
		t.Errorf("ViewTech = %d, want 4", ViewTech)
	}
	if ViewMission != 5 {
		t.Errorf("ViewMission = %d, want 5", ViewMission)
	}
}

func TestMode_Constants(t *testing.T) {
	// Verify mode constants are distinct
	modes := []Mode{ModeNormal, ModeInsert, ModeVisual}
	seen := make(map[Mode]bool)
	for _, m := range modes {
		if seen[m] {
			t.Errorf("duplicate mode value: %d", m)
		}
		seen[m] = true
	}
}

func TestFocusArea_Constants(t *testing.T) {
	// Verify focus area constants are distinct
	areas := []FocusArea{FocusMap, FocusPrompt, FocusAdvisors, FocusMissions, FocusResponse}
	seen := make(map[FocusArea]bool)
	for _, a := range areas {
		if seen[a] {
			t.Errorf("duplicate focus area value: %d", a)
		}
		seen[a] = true
	}
}
