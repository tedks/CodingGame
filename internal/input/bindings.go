package input

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// BindingStyle represents the keybinding style preference.
type BindingStyle int

const (
	// StyleVim uses vim-style keybindings (hjkl, Escape to normal mode).
	StyleVim BindingStyle = iota
	// StyleEmacs uses emacs-style keybindings (Ctrl combinations).
	StyleEmacs
)

// Action represents an input action that can be triggered by keybindings.
type Action int

const (
	// Navigation actions
	ActionMoveUp Action = iota
	ActionMoveDown
	ActionMoveLeft
	ActionMoveRight

	// Zoom actions
	ActionZoomIn
	ActionZoomOut

	// Mode actions
	ActionEnterInsert
	ActionEnterVisual
	ActionExitMode // Exit to Normal mode

	// Focus actions
	ActionFocusNext   // Tab to next panel
	ActionFocusPrev   // Shift+Tab to previous panel
	ActionFocusPrompt // Quick focus to prompt
	ActionFocusMap    // Quick focus to map

	// View switching (1-5)
	ActionView1
	ActionView2
	ActionView3
	ActionView4
	ActionView5

	// Prompt actions
	ActionSubmitPrompt // Enter in prompt
	ActionCancelPrompt // Escape in prompt

	// Selection actions
	ActionSelect      // Enter/Space on selected item
	ActionSelectMulti // Add to selection in Visual mode

	// Search/Go-to
	ActionSearch // '/' to search
	ActionGoTo   // 'g' to go to file

	// Map view actions
	ActionToggleMapView // Toggle between Directory and Dataflow views
)

// Binding maps a key combination to an action.
type Binding struct {
	Key       ebiten.Key
	Modifiers Modifiers
	Action    Action
	Modes     []Mode // Modes where this binding is active (empty = all modes)
}

// Modifiers represents keyboard modifier keys.
type Modifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
}

// Bindings holds all keybindings organized by style.
type Bindings struct {
	style    BindingStyle
	bindings []Binding
}

// NewBindings creates a new bindings set with the given style.
func NewBindings(style BindingStyle) *Bindings {
	b := &Bindings{
		style: style,
	}
	b.loadDefaultBindings()
	return b
}

// Style returns the current binding style.
func (b *Bindings) Style() BindingStyle {
	return b.style
}

// SetStyle changes the binding style and reloads defaults.
func (b *Bindings) SetStyle(style BindingStyle) {
	b.style = style
	b.loadDefaultBindings()
}

// loadDefaultBindings loads the default bindings for the current style.
func (b *Bindings) loadDefaultBindings() {
	b.bindings = nil

	switch b.style {
	case StyleVim:
		b.loadVimBindings()
	case StyleEmacs:
		b.loadEmacsBindings()
	}

	// Common bindings (same for all styles)
	b.loadCommonBindings()
}

// loadVimBindings loads vim-style keybindings.
func (b *Bindings) loadVimBindings() {
	normalMode := []Mode{ModeNormal}
	insertMode := []Mode{ModeInsert}
	normalVisualModes := []Mode{ModeNormal, ModeVisual}

	// Navigation (Normal and Visual modes)
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyH, Action: ActionMoveLeft, Modes: normalVisualModes},
		Binding{Key: ebiten.KeyJ, Action: ActionMoveDown, Modes: normalVisualModes},
		Binding{Key: ebiten.KeyK, Action: ActionMoveUp, Modes: normalVisualModes},
		Binding{Key: ebiten.KeyL, Action: ActionMoveRight, Modes: normalVisualModes},
	)

	// Mode switching
	insertVisualModes := []Mode{ModeInsert, ModeVisual}
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyI, Action: ActionEnterInsert, Modes: normalMode},
		Binding{Key: ebiten.KeyV, Action: ActionEnterVisual, Modes: normalMode},
		Binding{Key: ebiten.KeyEscape, Action: ActionExitMode, Modes: insertVisualModes},
	)

	// Focus to prompt with colon or Enter in Normal mode
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeySemicolon, Modifiers: Modifiers{Shift: true}, Action: ActionFocusPrompt, Modes: normalMode}, // ':'
		Binding{Key: ebiten.KeyEnter, Action: ActionFocusPrompt, Modes: normalMode},
	)

	// Search and go-to
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeySlash, Action: ActionSearch, Modes: normalMode},
		Binding{Key: ebiten.KeyG, Action: ActionGoTo, Modes: normalMode},
	)

	// Map view toggle
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyT, Action: ActionToggleMapView, Modes: normalMode},
	)

	// Prompt actions in Insert mode
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyEnter, Action: ActionSubmitPrompt, Modes: insertMode},
		Binding{Key: ebiten.KeyEscape, Action: ActionCancelPrompt, Modes: insertMode},
	)
}

// loadEmacsBindings loads emacs-style keybindings.
func (b *Bindings) loadEmacsBindings() {
	// Navigation with Ctrl combinations
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyP, Modifiers: Modifiers{Ctrl: true}, Action: ActionMoveUp},
		Binding{Key: ebiten.KeyN, Modifiers: Modifiers{Ctrl: true}, Action: ActionMoveDown},
		Binding{Key: ebiten.KeyB, Modifiers: Modifiers{Ctrl: true}, Action: ActionMoveLeft},
		Binding{Key: ebiten.KeyF, Modifiers: Modifiers{Ctrl: true}, Action: ActionMoveRight},
	)

	// Mode switching
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyG, Modifiers: Modifiers{Ctrl: true}, Action: ActionExitMode},
	)

	// Search
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyS, Modifiers: Modifiers{Ctrl: true}, Action: ActionSearch},
	)

	// Submit with Ctrl+Enter or just Enter
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyEnter, Modifiers: Modifiers{Ctrl: true}, Action: ActionSubmitPrompt},
		Binding{Key: ebiten.KeyEnter, Action: ActionSubmitPrompt, Modes: []Mode{ModeInsert}},
	)
}

// loadCommonBindings loads bindings that are the same for all styles.
func (b *Bindings) loadCommonBindings() {
	// Arrow keys for navigation (always work)
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyArrowUp, Action: ActionMoveUp},
		Binding{Key: ebiten.KeyArrowDown, Action: ActionMoveDown},
		Binding{Key: ebiten.KeyArrowLeft, Action: ActionMoveLeft},
		Binding{Key: ebiten.KeyArrowRight, Action: ActionMoveRight},
	)

	// Zoom with +/- keys
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyEqual, Action: ActionZoomIn},
		Binding{Key: ebiten.KeyMinus, Action: ActionZoomOut},
		Binding{Key: ebiten.KeyKPAdd, Action: ActionZoomIn},
		Binding{Key: ebiten.KeyKPSubtract, Action: ActionZoomOut},
	)

	// Tab cycling
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeyTab, Action: ActionFocusNext},
		Binding{Key: ebiten.KeyTab, Modifiers: Modifiers{Shift: true}, Action: ActionFocusPrev},
	)

	// View switching with number keys 1-5
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.Key1, Action: ActionView1},
		Binding{Key: ebiten.Key2, Action: ActionView2},
		Binding{Key: ebiten.Key3, Action: ActionView3},
		Binding{Key: ebiten.Key4, Action: ActionView4},
		Binding{Key: ebiten.Key5, Action: ActionView5},
	)

	// Selection
	b.bindings = append(b.bindings,
		Binding{Key: ebiten.KeySpace, Action: ActionSelect},
	)
}

// GetAction returns the action for the given key press in the given mode.
// Returns the action and true if a binding exists, or (0, false) if not.
func (b *Bindings) GetAction(key ebiten.Key, modifiers Modifiers, mode Mode) (Action, bool) {
	for _, binding := range b.bindings {
		if binding.Key != key {
			continue
		}
		if binding.Modifiers != modifiers {
			continue
		}
		// Check if binding applies to current mode
		if len(binding.Modes) > 0 {
			found := false
			for _, m := range binding.Modes {
				if m == mode {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		return binding.Action, true
	}
	return 0, false
}

// AddBinding adds a custom binding.
func (b *Bindings) AddBinding(binding Binding) {
	b.bindings = append(b.bindings, binding)
}

// RemoveBinding removes a binding by action.
func (b *Bindings) RemoveBinding(action Action) {
	newBindings := make([]Binding, 0, len(b.bindings))
	for _, binding := range b.bindings {
		if binding.Action != action {
			newBindings = append(newBindings, binding)
		}
	}
	b.bindings = newBindings
}
