package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewBindings_Vim(t *testing.T) {
	b := NewBindings(StyleVim)

	if b.Style() != StyleVim {
		t.Errorf("Style() = %v, want %v", b.Style(), StyleVim)
	}

	// Check that some vim-specific bindings exist
	// h should map to MoveLeft in Normal mode
	action, found := b.GetAction(ebiten.KeyH, Modifiers{}, ModeNormal)
	if !found {
		t.Error("expected 'h' key to be bound in vim style")
	}
	if action != ActionMoveLeft {
		t.Errorf("'h' action = %v, want %v", action, ActionMoveLeft)
	}
}

func TestNewBindings_Emacs(t *testing.T) {
	b := NewBindings(StyleEmacs)

	if b.Style() != StyleEmacs {
		t.Errorf("Style() = %v, want %v", b.Style(), StyleEmacs)
	}

	// Check that some emacs-specific bindings exist
	// Ctrl+P should map to MoveUp
	action, found := b.GetAction(ebiten.KeyP, Modifiers{Ctrl: true}, ModeNormal)
	if !found {
		t.Error("expected Ctrl+P to be bound in emacs style")
	}
	if action != ActionMoveUp {
		t.Errorf("Ctrl+P action = %v, want %v", action, ActionMoveUp)
	}
}

func TestBindings_SetStyle(t *testing.T) {
	b := NewBindings(StyleVim)

	// Switch to emacs
	b.SetStyle(StyleEmacs)

	if b.Style() != StyleEmacs {
		t.Errorf("after SetStyle(Emacs), Style() = %v, want %v", b.Style(), StyleEmacs)
	}

	// Vim 'h' binding should no longer exist (without modifiers)
	_, found := b.GetAction(ebiten.KeyH, Modifiers{}, ModeNormal)
	if found {
		t.Error("vim 'h' binding should not exist after switching to emacs style")
	}
}

func TestBindings_CommonBindings(t *testing.T) {
	// Both styles should have common bindings
	styles := []BindingStyle{StyleVim, StyleEmacs}

	for _, style := range styles {
		b := NewBindings(style)

		// Arrow keys should always work
		testCases := []struct {
			key    ebiten.Key
			action Action
		}{
			{ebiten.KeyArrowUp, ActionMoveUp},
			{ebiten.KeyArrowDown, ActionMoveDown},
			{ebiten.KeyArrowLeft, ActionMoveLeft},
			{ebiten.KeyArrowRight, ActionMoveRight},
		}

		for _, tc := range testCases {
			action, found := b.GetAction(tc.key, Modifiers{}, ModeNormal)
			if !found {
				t.Errorf("style %v: expected %v to be bound", style, tc.key)
				continue
			}
			if action != tc.action {
				t.Errorf("style %v: %v action = %v, want %v", style, tc.key, action, tc.action)
			}
		}

		// Tab should cycle focus
		action, found := b.GetAction(ebiten.KeyTab, Modifiers{}, ModeNormal)
		if !found {
			t.Errorf("style %v: expected Tab to be bound", style)
		} else if action != ActionFocusNext {
			t.Errorf("style %v: Tab action = %v, want %v", style, action, ActionFocusNext)
		}

		// Number keys should switch views
		viewKeys := []struct {
			key    ebiten.Key
			action Action
		}{
			{ebiten.Key1, ActionView1},
			{ebiten.Key2, ActionView2},
			{ebiten.Key3, ActionView3},
			{ebiten.Key4, ActionView4},
			{ebiten.Key5, ActionView5},
		}

		for _, tc := range viewKeys {
			action, found := b.GetAction(tc.key, Modifiers{}, ModeNormal)
			if !found {
				t.Errorf("style %v: expected %v to be bound", style, tc.key)
				continue
			}
			if action != tc.action {
				t.Errorf("style %v: %v action = %v, want %v", style, tc.key, action, tc.action)
			}
		}
	}
}

func TestBindings_ModeSpecific(t *testing.T) {
	b := NewBindings(StyleVim)

	// 'i' should enter insert mode only in Normal mode
	action, found := b.GetAction(ebiten.KeyI, Modifiers{}, ModeNormal)
	if !found {
		t.Error("expected 'i' to be bound in Normal mode")
	}
	if action != ActionEnterInsert {
		t.Errorf("'i' action in Normal = %v, want %v", action, ActionEnterInsert)
	}

	// 'i' should NOT be bound in Insert mode (it's for typing)
	_, found = b.GetAction(ebiten.KeyI, Modifiers{}, ModeInsert)
	if found {
		t.Error("'i' should not be bound as action in Insert mode")
	}
}

func TestBindings_GetAction_NotFound(t *testing.T) {
	b := NewBindings(StyleVim)

	// Some key that's not bound
	_, found := b.GetAction(ebiten.KeyF12, Modifiers{}, ModeNormal)
	if found {
		t.Error("expected F12 to not be bound")
	}
}

func TestBindings_AddBinding(t *testing.T) {
	b := NewBindings(StyleVim)

	// Add a custom binding for F12
	b.AddBinding(Binding{
		Key:    ebiten.KeyF12,
		Action: ActionSearch,
	})

	action, found := b.GetAction(ebiten.KeyF12, Modifiers{}, ModeNormal)
	if !found {
		t.Error("expected F12 to be bound after AddBinding")
	}
	if action != ActionSearch {
		t.Errorf("F12 action = %v, want %v", action, ActionSearch)
	}
}

func TestBindings_RemoveBinding(t *testing.T) {
	b := NewBindings(StyleVim)

	// Remove the search action ('/')
	b.RemoveBinding(ActionSearch)

	_, found := b.GetAction(ebiten.KeySlash, Modifiers{}, ModeNormal)
	if found {
		t.Error("expected '/' binding to be removed")
	}
}

func TestModifiers_Equality(t *testing.T) {
	m1 := Modifiers{Shift: true, Ctrl: false, Alt: false}
	m2 := Modifiers{Shift: true, Ctrl: false, Alt: false}
	m3 := Modifiers{Shift: false, Ctrl: true, Alt: false}

	if m1 != m2 {
		t.Error("expected equal modifiers to be equal")
	}
	if m1 == m3 {
		t.Error("expected different modifiers to not be equal")
	}
}

func TestBindings_ModifierMatching(t *testing.T) {
	b := NewBindings(StyleVim)

	// Shift+Tab should be focus prev
	action, found := b.GetAction(ebiten.KeyTab, Modifiers{Shift: true}, ModeNormal)
	if !found {
		t.Error("expected Shift+Tab to be bound")
	}
	if action != ActionFocusPrev {
		t.Errorf("Shift+Tab action = %v, want %v", action, ActionFocusPrev)
	}

	// Tab without shift should be focus next
	action, found = b.GetAction(ebiten.KeyTab, Modifiers{}, ModeNormal)
	if !found {
		t.Error("expected Tab to be bound")
	}
	if action != ActionFocusNext {
		t.Errorf("Tab action = %v, want %v", action, ActionFocusNext)
	}
}
