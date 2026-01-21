// Package input provides keyboard input handling with vim-style modes and
// configurable keybindings. It supports Normal, Insert, and Visual modes
// with focus management for navigating between UI panels.
package input

// Mode represents the current input mode (vim-style).
type Mode int

const (
	// ModeNormal is for navigation and commands (default mode).
	ModeNormal Mode = iota
	// ModeInsert is for text entry (typing into prompt/input fields).
	ModeInsert
	// ModeVisual is for selecting multiple elements on the map.
	ModeVisual
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	default:
		return "UNKNOWN"
	}
}

// FocusArea represents which UI panel has keyboard focus.
type FocusArea int

const (
	// FocusMap is the main map view (directory/dataflow).
	FocusMap FocusArea = iota
	// FocusPrompt is the prompt/command input area.
	FocusPrompt
	// FocusAdvisors is the advisor panel.
	FocusAdvisors
	// FocusMissions is the mission/objective panel.
	FocusMissions
	// FocusResponse is the response/output panel.
	FocusResponse
)

// String returns the string representation of the focus area.
func (f FocusArea) String() string {
	switch f {
	case FocusMap:
		return "Map"
	case FocusPrompt:
		return "Prompt"
	case FocusAdvisors:
		return "Advisors"
	case FocusMissions:
		return "Missions"
	case FocusResponse:
		return "Response"
	default:
		return "Unknown"
	}
}

// focusOrder defines the order for Tab cycling between panels.
var focusOrder = []FocusArea{
	FocusMap,
	FocusPrompt,
	FocusAdvisors,
	FocusMissions,
	FocusResponse,
}

// NextFocus returns the next focus area in the cycle.
func NextFocus(current FocusArea) FocusArea {
	for i, f := range focusOrder {
		if f == current {
			return focusOrder[(i+1)%len(focusOrder)]
		}
	}
	return FocusMap
}

// PrevFocus returns the previous focus area in the cycle.
func PrevFocus(current FocusArea) FocusArea {
	for i, f := range focusOrder {
		if f == current {
			idx := i - 1
			if idx < 0 {
				idx = len(focusOrder) - 1
			}
			return focusOrder[idx]
		}
	}
	return FocusMap
}

// ViewNumber represents the numbered view (1-7) for quick switching.
type ViewNumber int

const (
	ViewMap        ViewNumber = 1 // Directory/Dataflow map
	ViewBuilding   ViewNumber = 2 // Buildings (build targets)
	ViewUnit       ViewNumber = 3 // Units (tests)
	ViewTech       ViewNumber = 4 // Tech tree (capabilities)
	ViewMission    ViewNumber = 5 // Missions/objectives
	ViewProduction ViewNumber = 6 // Production realm (deployed services)
	ViewMultiAgent ViewNumber = 7 // Multi-agent orchestration
)
