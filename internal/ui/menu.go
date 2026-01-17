package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// MenuItem represents a single menu option.
type MenuItem struct {
	Label   string
	Value   string // Optional value for the item (e.g., path, ID)
	Enabled bool
}

// NewMenuItem creates an enabled menu item.
func NewMenuItem(label string) *MenuItem {
	return &MenuItem{
		Label:   label,
		Value:   label,
		Enabled: true,
	}
}

// NewMenuItemWithValue creates an enabled menu item with a specific value.
func NewMenuItemWithValue(label, value string) *MenuItem {
	return &MenuItem{
		Label:   label,
		Value:   value,
		Enabled: true,
	}
}

// Menu represents a navigable list of options.
type Menu struct {
	Title         string
	Items         []*MenuItem
	SelectedIndex int

	// Position and size
	X, Y          int
	Width, Height int

	// Styling
	TitleColor      color.RGBA
	ItemColor       color.RGBA
	SelectedColor   color.RGBA
	DisabledColor   color.RGBA
	BackgroundColor color.RGBA
	BorderColor     color.RGBA

	// Behavior
	CancelAllowed bool // Whether Escape key goes back/cancels
}

// NewMenu creates a new menu with default styling.
func NewMenu(title string, items []*MenuItem) *Menu {
	return &Menu{
		Title:           title,
		Items:           items,
		SelectedIndex:   0,
		Width:           300,
		Height:          0, // Auto-calculated
		TitleColor:      color.RGBA{255, 255, 255, 255},
		ItemColor:       color.RGBA{180, 180, 200, 255},
		SelectedColor:   color.RGBA{100, 200, 255, 255},
		DisabledColor:   color.RGBA{100, 100, 100, 255},
		BackgroundColor: color.RGBA{30, 30, 45, 230},
		BorderColor:     color.RGBA{80, 80, 120, 255},
		CancelAllowed:   true,
	}
}

// Update handles menu navigation input.
// Returns: (selected item value, cancelled, error)
func (m *Menu) Update() (selected string, cancelled bool, err error) {
	// Handle up/down navigation
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyK) {
		m.moveSelection(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyJ) {
		m.moveSelection(1)
	}

	// Handle selection with Enter or Space
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if m.SelectedIndex >= 0 && m.SelectedIndex < len(m.Items) {
			item := m.Items[m.SelectedIndex]
			if item.Enabled {
				return item.Value, false, nil
			}
		}
	}

	// Handle cancel with Escape
	if m.CancelAllowed && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return "", true, nil
	}

	return "", false, nil
}

// moveSelection moves the selection by delta, skipping disabled items.
func (m *Menu) moveSelection(delta int) {
	if len(m.Items) == 0 {
		return
	}

	newIndex := m.SelectedIndex
	for i := 0; i < len(m.Items); i++ {
		newIndex += delta
		if newIndex < 0 {
			newIndex = len(m.Items) - 1
		} else if newIndex >= len(m.Items) {
			newIndex = 0
		}

		if m.Items[newIndex].Enabled {
			m.SelectedIndex = newIndex
			return
		}
	}
}

// Draw renders the menu.
func (m *Menu) Draw(screen *ebiten.Image) {
	const (
		padding     = 20
		titleHeight = 30
		itemHeight  = 24
		lineSpacing = 4
	)

	// Calculate height based on items
	menuHeight := padding*2 + titleHeight + len(m.Items)*(itemHeight+lineSpacing)

	// Draw background
	vector.DrawFilledRect(
		screen,
		float32(m.X),
		float32(m.Y),
		float32(m.Width),
		float32(menuHeight),
		m.BackgroundColor,
		false,
	)

	// Draw border
	vector.StrokeRect(
		screen,
		float32(m.X),
		float32(m.Y),
		float32(m.Width),
		float32(menuHeight),
		2,
		m.BorderColor,
		false,
	)

	// Draw title
	titleX := m.X + padding
	titleY := m.Y + padding
	ebitenutil.DebugPrintAt(screen, m.Title, titleX, titleY)

	// Draw items
	itemY := m.Y + padding + titleHeight
	for i, item := range m.Items {
		// Determine item color
		itemColor := m.ItemColor
		if !item.Enabled {
			itemColor = m.DisabledColor
		} else if i == m.SelectedIndex {
			itemColor = m.SelectedColor

			// Draw selection highlight
			vector.DrawFilledRect(
				screen,
				float32(m.X+padding/2),
				float32(itemY-2),
				float32(m.Width-padding),
				float32(itemHeight),
				color.RGBA{60, 60, 80, 200},
				false,
			)
		}

		// Draw item text with prefix
		prefix := "  "
		if i == m.SelectedIndex && item.Enabled {
			prefix = "> "
		}
		text := prefix + item.Label

		// Note: ebitenutil.DebugPrint doesn't support color,
		// so we use the same color for all text for now.
		// In a production version, we'd use a proper text renderer.
		_ = itemColor // TODO: Use colored text rendering
		ebitenutil.DebugPrintAt(screen, text, m.X+padding, itemY)

		itemY += itemHeight + lineSpacing
	}
}

// SetPosition sets the menu's top-left position.
func (m *Menu) SetPosition(x, y int) {
	m.X = x
	m.Y = y
}

// Center centers the menu on the screen.
func (m *Menu) Center(screenWidth, screenHeight int) {
	const (
		padding     = 20
		titleHeight = 30
		itemHeight  = 24
		lineSpacing = 4
	)

	menuHeight := padding*2 + titleHeight + len(m.Items)*(itemHeight+lineSpacing)

	m.X = (screenWidth - m.Width) / 2
	m.Y = (screenHeight - menuHeight) / 2
}

// SelectedItem returns the currently selected item, or nil if none.
func (m *Menu) SelectedItem() *MenuItem {
	if m.SelectedIndex >= 0 && m.SelectedIndex < len(m.Items) {
		return m.Items[m.SelectedIndex]
	}
	return nil
}

// SetSelectedByValue selects the item with the given value.
func (m *Menu) SetSelectedByValue(value string) {
	for i, item := range m.Items {
		if item.Value == value {
			m.SelectedIndex = i
			return
		}
	}
}
