package ui

import (
	"testing"
)

func TestNewMenuItem(t *testing.T) {
	item := NewMenuItem("Test Label")

	if item.Label != "Test Label" {
		t.Errorf("expected label 'Test Label', got %q", item.Label)
	}
	if item.Value != "Test Label" {
		t.Errorf("expected value 'Test Label', got %q", item.Value)
	}
	if !item.Enabled {
		t.Error("expected item to be enabled by default")
	}
}

func TestNewMenuItemWithValue(t *testing.T) {
	item := NewMenuItemWithValue("Display Label", "internal-value")

	if item.Label != "Display Label" {
		t.Errorf("expected label 'Display Label', got %q", item.Label)
	}
	if item.Value != "internal-value" {
		t.Errorf("expected value 'internal-value', got %q", item.Value)
	}
}

func TestNewMenu(t *testing.T) {
	items := []*MenuItem{
		NewMenuItem("Option 1"),
		NewMenuItem("Option 2"),
	}
	menu := NewMenu("Test Menu", items)

	if menu.Title != "Test Menu" {
		t.Errorf("expected title 'Test Menu', got %q", menu.Title)
	}
	if len(menu.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(menu.Items))
	}
	if menu.SelectedIndex != 0 {
		t.Errorf("expected initial selection 0, got %d", menu.SelectedIndex)
	}
	if !menu.CancelAllowed {
		t.Error("expected CancelAllowed to be true by default")
	}
}

func TestMenu_SelectedItem(t *testing.T) {
	items := []*MenuItem{
		NewMenuItem("Option 1"),
		NewMenuItem("Option 2"),
	}
	menu := NewMenu("Test", items)

	// Initial selection
	selected := menu.SelectedItem()
	if selected == nil || selected.Label != "Option 1" {
		t.Error("expected initial selected item to be 'Option 1'")
	}

	// After changing selection
	menu.SelectedIndex = 1
	selected = menu.SelectedItem()
	if selected == nil || selected.Label != "Option 2" {
		t.Error("expected selected item to be 'Option 2'")
	}
}

func TestMenu_SelectedItem_InvalidIndex(t *testing.T) {
	menu := NewMenu("Test", []*MenuItem{})

	selected := menu.SelectedItem()
	if selected != nil {
		t.Error("expected nil for empty menu")
	}

	menu.SelectedIndex = -1
	selected = menu.SelectedItem()
	if selected != nil {
		t.Error("expected nil for negative index")
	}

	menu.SelectedIndex = 100
	selected = menu.SelectedItem()
	if selected != nil {
		t.Error("expected nil for out-of-bounds index")
	}
}

func TestMenu_SetSelectedByValue(t *testing.T) {
	items := []*MenuItem{
		NewMenuItemWithValue("Label A", "value-a"),
		NewMenuItemWithValue("Label B", "value-b"),
		NewMenuItemWithValue("Label C", "value-c"),
	}
	menu := NewMenu("Test", items)

	menu.SetSelectedByValue("value-b")
	if menu.SelectedIndex != 1 {
		t.Errorf("expected selected index 1, got %d", menu.SelectedIndex)
	}

	menu.SetSelectedByValue("value-c")
	if menu.SelectedIndex != 2 {
		t.Errorf("expected selected index 2, got %d", menu.SelectedIndex)
	}

	// Non-existent value should not change selection
	menu.SetSelectedByValue("nonexistent")
	if menu.SelectedIndex != 2 {
		t.Errorf("expected selected index to remain 2, got %d", menu.SelectedIndex)
	}
}

func TestMenu_Center(t *testing.T) {
	menu := NewMenu("Test", []*MenuItem{
		NewMenuItem("Item 1"),
		NewMenuItem("Item 2"),
	})
	menu.Width = 200

	menu.Center(800, 600)

	// Menu should be roughly centered
	if menu.X < 250 || menu.X > 350 {
		t.Errorf("expected X around 300, got %d", menu.X)
	}
	if menu.Y < 200 || menu.Y > 350 {
		t.Errorf("expected Y roughly centered, got %d", menu.Y)
	}
}

func TestMenu_SetPosition(t *testing.T) {
	menu := NewMenu("Test", nil)

	menu.SetPosition(100, 200)

	if menu.X != 100 {
		t.Errorf("expected X 100, got %d", menu.X)
	}
	if menu.Y != 200 {
		t.Errorf("expected Y 200, got %d", menu.Y)
	}
}

func TestMenu_MoveSelection_SkipsDisabled(t *testing.T) {
	items := []*MenuItem{
		NewMenuItem("Item 1"),
		NewMenuItem("Item 2"),
		NewMenuItem("Item 3"),
	}
	items[1].Enabled = false // Middle item disabled

	menu := NewMenu("Test", items)
	menu.SelectedIndex = 0

	// Moving down should skip disabled item
	menu.moveSelection(1)
	if menu.SelectedIndex != 2 {
		t.Errorf("expected to skip to index 2, got %d", menu.SelectedIndex)
	}

	// Moving up should wrap and skip disabled
	menu.moveSelection(-1)
	if menu.SelectedIndex != 0 {
		t.Errorf("expected to wrap to index 0, got %d", menu.SelectedIndex)
	}
}

func TestMenu_MoveSelection_Wrap(t *testing.T) {
	items := []*MenuItem{
		NewMenuItem("Item 1"),
		NewMenuItem("Item 2"),
		NewMenuItem("Item 3"),
	}
	menu := NewMenu("Test", items)
	menu.SelectedIndex = 0

	// Move up should wrap to end
	menu.moveSelection(-1)
	if menu.SelectedIndex != 2 {
		t.Errorf("expected wrap to index 2, got %d", menu.SelectedIndex)
	}

	// Move down should wrap to start
	menu.moveSelection(1)
	if menu.SelectedIndex != 0 {
		t.Errorf("expected wrap to index 0, got %d", menu.SelectedIndex)
	}
}

func TestMenu_MoveSelection_EmptyMenu(t *testing.T) {
	menu := NewMenu("Test", []*MenuItem{})

	// Should not panic
	menu.moveSelection(1)
	menu.moveSelection(-1)
}

func TestMenu_MoveSelection_AllDisabled(t *testing.T) {
	items := []*MenuItem{
		NewMenuItem("Item 1"),
		NewMenuItem("Item 2"),
	}
	items[0].Enabled = false
	items[1].Enabled = false

	menu := NewMenu("Test", items)
	initialIndex := menu.SelectedIndex

	// Should not change selection when all items disabled
	menu.moveSelection(1)
	if menu.SelectedIndex != initialIndex {
		t.Errorf("expected selection to remain at %d, got %d", initialIndex, menu.SelectedIndex)
	}
}
