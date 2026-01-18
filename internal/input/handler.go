package input

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// Handler manages keyboard input state including modes, focus, and keybindings.
type Handler struct {
	mu sync.RWMutex

	// Current state
	mode     Mode
	focus    FocusArea
	view     ViewNumber
	bindings *Bindings

	// Input source (for testing injection)
	inputSource InputSource

	// Event callbacks
	onAction      func(Action)
	onModeChange  func(Mode)
	onFocusChange func(FocusArea)
	onViewChange  func(ViewNumber)

	// Text input buffer (for Insert mode)
	textBuffer   string
	onTextChange func(string)
}

// NewHandler creates a new input handler with vim-style bindings.
func NewHandler() *Handler {
	return &Handler{
		mode:        ModeNormal,
		focus:       FocusMap,
		view:        ViewMap,
		bindings:    NewBindings(StyleVim),
		inputSource: DefaultSource,
	}
}

// SetInputSource sets the input source for testing.
// Pass nil to reset to the default Ebitengine source.
func (h *Handler) SetInputSource(source InputSource) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if source == nil {
		h.inputSource = DefaultSource
	} else {
		h.inputSource = source
	}
}

// InputSource returns the current input source.
func (h *Handler) InputSource() InputSource {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.inputSource
}

// Mode returns the current input mode.
func (h *Handler) Mode() Mode {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.mode
}

// SetMode sets the input mode.
func (h *Handler) SetMode(mode Mode) {
	h.mu.Lock()
	oldMode := h.mode
	h.mode = mode
	callback := h.onModeChange
	h.mu.Unlock()

	if callback != nil && oldMode != mode {
		callback(mode)
	}
}

// Focus returns the current focus area.
func (h *Handler) Focus() FocusArea {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.focus
}

// SetFocus sets the focus area.
func (h *Handler) SetFocus(focus FocusArea) {
	h.mu.Lock()
	oldFocus := h.focus
	h.focus = focus
	callback := h.onFocusChange
	h.mu.Unlock()

	if callback != nil && oldFocus != focus {
		callback(focus)
	}
}

// View returns the current view number.
func (h *Handler) View() ViewNumber {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.view
}

// SetView sets the current view.
func (h *Handler) SetView(view ViewNumber) {
	h.mu.Lock()
	oldView := h.view
	h.view = view
	callback := h.onViewChange
	h.mu.Unlock()

	if callback != nil && oldView != view {
		callback(view)
	}
}

// Bindings returns the keybindings configuration.
func (h *Handler) Bindings() *Bindings {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.bindings
}

// SetBindingStyle changes the keybinding style.
func (h *Handler) SetBindingStyle(style BindingStyle) {
	h.mu.Lock()
	h.bindings.SetStyle(style)
	h.mu.Unlock()
}

// TextBuffer returns the current text input buffer.
func (h *Handler) TextBuffer() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.textBuffer
}

// SetTextBuffer sets the text input buffer.
func (h *Handler) SetTextBuffer(text string) {
	h.mu.Lock()
	h.textBuffer = text
	callback := h.onTextChange
	h.mu.Unlock()

	if callback != nil {
		callback(text)
	}
}

// ClearTextBuffer clears the text input buffer.
func (h *Handler) ClearTextBuffer() {
	h.SetTextBuffer("")
}

// OnAction sets the callback for action events.
func (h *Handler) OnAction(callback func(Action)) {
	h.mu.Lock()
	h.onAction = callback
	h.mu.Unlock()
}

// OnModeChange sets the callback for mode change events.
func (h *Handler) OnModeChange(callback func(Mode)) {
	h.mu.Lock()
	h.onModeChange = callback
	h.mu.Unlock()
}

// OnFocusChange sets the callback for focus change events.
func (h *Handler) OnFocusChange(callback func(FocusArea)) {
	h.mu.Lock()
	h.onFocusChange = callback
	h.mu.Unlock()
}

// OnViewChange sets the callback for view change events.
func (h *Handler) OnViewChange(callback func(ViewNumber)) {
	h.mu.Lock()
	h.onViewChange = callback
	h.mu.Unlock()
}

// OnTextChange sets the callback for text buffer change events.
func (h *Handler) OnTextChange(callback func(string)) {
	h.mu.Lock()
	h.onTextChange = callback
	h.mu.Unlock()
}

// Update processes input for the current frame.
// Should be called once per frame in the game's Update method.
func (h *Handler) Update() {
	h.mu.RLock()
	mode := h.mode
	focus := h.focus
	bindings := h.bindings
	actionCallback := h.onAction
	inputSource := h.inputSource
	h.mu.RUnlock()

	// In Insert mode, handle text input
	if mode == ModeInsert {
		h.handleTextInput()
	}

	// Check for modifier keys
	modifiers := Modifiers{
		Shift: inputSource.IsKeyPressed(ebiten.KeyShift),
		Ctrl:  inputSource.IsKeyPressed(ebiten.KeyControl),
		Alt:   inputSource.IsKeyPressed(ebiten.KeyAlt),
	}

	// Process all just-pressed keys
	pressedKeys := inputSource.JustPressedKeys()
	for _, key := range pressedKeys {
		// Skip modifier keys themselves
		if key == ebiten.KeyShift || key == ebiten.KeyControl || key == ebiten.KeyAlt {
			continue
		}

		// Look up action in bindings
		if action, found := bindings.GetAction(key, modifiers, mode); found {
			// Handle built-in actions
			h.handleBuiltInAction(action)

			// Call user callback
			if actionCallback != nil {
				actionCallback(action)
			}
		}
	}

	// Handle focus-specific input
	h.handleFocusInput(focus, mode)
}

// handleTextInput handles text entry in Insert mode.
func (h *Handler) handleTextInput() {
	h.mu.RLock()
	inputSource := h.inputSource
	h.mu.RUnlock()

	// Get typed characters
	chars := inputSource.AppendInputChars(nil)
	if len(chars) > 0 {
		h.mu.Lock()
		h.textBuffer += string(chars)
		text := h.textBuffer
		callback := h.onTextChange
		h.mu.Unlock()

		if callback != nil {
			callback(text)
		}
	}

	// Handle backspace
	if inputSource.IsKeyJustPressed(ebiten.KeyBackspace) {
		h.mu.Lock()
		if len(h.textBuffer) > 0 {
			h.textBuffer = h.textBuffer[:len(h.textBuffer)-1]
		}
		text := h.textBuffer
		callback := h.onTextChange
		h.mu.Unlock()

		if callback != nil {
			callback(text)
		}
	}
}

// handleBuiltInAction handles actions that affect the handler's internal state.
func (h *Handler) handleBuiltInAction(action Action) {
	switch action {
	case ActionEnterInsert:
		h.SetMode(ModeInsert)
	case ActionEnterVisual:
		h.SetMode(ModeVisual)
	case ActionExitMode, ActionCancelPrompt:
		h.SetMode(ModeNormal)
	case ActionFocusNext:
		h.SetFocus(NextFocus(h.Focus()))
	case ActionFocusPrev:
		h.SetFocus(PrevFocus(h.Focus()))
	case ActionFocusPrompt:
		h.SetFocus(FocusPrompt)
		h.SetMode(ModeInsert)
	case ActionFocusMap:
		h.SetFocus(FocusMap)
		h.SetMode(ModeNormal)
	case ActionView1:
		h.SetView(ViewMap)
	case ActionView2:
		h.SetView(ViewBuilding)
	case ActionView3:
		h.SetView(ViewUnit)
	case ActionView4:
		h.SetView(ViewTech)
	case ActionView5:
		h.SetView(ViewMission)
	}
}

// handleFocusInput handles input specific to the current focus area.
func (h *Handler) handleFocusInput(focus FocusArea, mode Mode) {
	// Future: Add focus-specific input handling
	// For now, most input is handled through the general bindings system
	_ = focus
	_ = mode
}

// IsAction checks if a specific action was just triggered this frame.
func (h *Handler) IsAction(action Action) bool {
	h.mu.RLock()
	mode := h.mode
	bindings := h.bindings
	inputSource := h.inputSource
	h.mu.RUnlock()

	modifiers := Modifiers{
		Shift: inputSource.IsKeyPressed(ebiten.KeyShift),
		Ctrl:  inputSource.IsKeyPressed(ebiten.KeyControl),
		Alt:   inputSource.IsKeyPressed(ebiten.KeyAlt),
	}

	pressedKeys := inputSource.JustPressedKeys()
	for _, key := range pressedKeys {
		if a, found := bindings.GetAction(key, modifiers, mode); found && a == action {
			return true
		}
	}
	return false
}

// IsActionHeld checks if a key for a specific action is currently held.
func (h *Handler) IsActionHeld(action Action) bool {
	h.mu.RLock()
	mode := h.mode
	bindings := h.bindings
	inputSource := h.inputSource
	h.mu.RUnlock()

	modifiers := Modifiers{
		Shift: inputSource.IsKeyPressed(ebiten.KeyShift),
		Ctrl:  inputSource.IsKeyPressed(ebiten.KeyControl),
		Alt:   inputSource.IsKeyPressed(ebiten.KeyAlt),
	}

	// Check all bindings for this action
	for _, binding := range bindings.bindings {
		if binding.Action != action {
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
		// Check if key is held
		if inputSource.IsKeyPressed(binding.Key) {
			return true
		}
	}
	return false
}
