package belt

import (
	"image/color"
	"testing"

	"github.com/tedks/CodingGame/internal/connection"
)

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	if r == nil {
		t.Fatal("NewRenderer returned nil")
	}

	// Verify default values
	if r.minWidth != 1.0 {
		t.Errorf("expected minWidth 1.0, got %f", r.minWidth)
	}
	if r.maxWidth != 6.0 {
		t.Errorf("expected maxWidth 6.0, got %f", r.maxWidth)
	}
	if r.maxStrength != 10 {
		t.Errorf("expected maxStrength 10, got %d", r.maxStrength)
	}
}

func TestTilePositionCenter(t *testing.T) {
	tests := []struct {
		pos     TilePosition
		expectX float32
		expectY float32
	}{
		{TilePosition{0, 0, 100, 100}, 50, 50},
		{TilePosition{10, 20, 100, 200}, 60, 120},
		{TilePosition{-50, -50, 100, 100}, 0, 0},
	}

	for _, tt := range tests {
		x, y := tt.pos.Center()
		if x != tt.expectX || y != tt.expectY {
			t.Errorf("TilePosition{%v}.Center() = (%f, %f), want (%f, %f)",
				tt.pos, x, y, tt.expectX, tt.expectY)
		}
	}
}

func TestRendererGetColor(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		name     string
		connType connection.Type
		circular bool
		want     color.RGBA
	}{
		{"import", connection.TypeImport, false, r.importColor},
		{"inheritance", connection.TypeInheritance, false, r.inheritanceColor},
		{"composition", connection.TypeComposition, false, r.compositionColor},
		{"call", connection.TypeCall, false, r.callColor},
		{"unknown", connection.TypeUnknown, false, r.importColor}, // Falls back to import
		{"circular import", connection.TypeImport, true, r.circularColor},
		{"circular call", connection.TypeCall, true, r.circularColor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := connection.NewConnection("a.go", "b.go", tt.connType)
			if tt.circular {
				conn.SetCircular(true)
			}
			got := r.getColor(conn)
			if got != tt.want {
				t.Errorf("getColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRendererGetWidth(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		strength int
		wantMin  float32 // Minimum expected value
		wantMax  float32 // Maximum expected value
	}{
		{0, r.minWidth, r.minWidth},
		{-5, r.minWidth, r.minWidth},
		{1, r.minWidth, r.maxWidth},
		{5, r.minWidth, r.maxWidth},   // Midpoint
		{10, r.maxWidth, r.maxWidth},  // At max
		{100, r.maxWidth, r.maxWidth}, // Above max
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			conn := connection.NewConnection("a.go", "b.go", connection.TypeImport)
			conn.SetStrength(tt.strength)
			got := r.getWidth(conn)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("getWidth(strength=%d) = %f, want between %f and %f",
					tt.strength, got, tt.wantMin, tt.wantMax)
			}
		})
	}

	// Test linear interpolation more specifically
	conn := connection.NewConnection("a.go", "b.go", connection.TypeImport)
	conn.SetStrength(5)
	width5 := r.getWidth(conn)
	expectedMid := r.minWidth + 0.5*(r.maxWidth-r.minWidth)
	if width5 != expectedMid {
		t.Errorf("getWidth(strength=5) = %f, want %f (midpoint)", width5, expectedMid)
	}
}

func TestRendererSetColors(t *testing.T) {
	r := NewRenderer()

	newImport := color.RGBA{1, 2, 3, 4}
	newInheritance := color.RGBA{5, 6, 7, 8}
	newComposition := color.RGBA{9, 10, 11, 12}
	newCall := color.RGBA{13, 14, 15, 16}
	newCircular := color.RGBA{17, 18, 19, 20}

	r.SetColors(newImport, newInheritance, newComposition, newCall, newCircular)

	if r.importColor != newImport {
		t.Errorf("importColor = %v, want %v", r.importColor, newImport)
	}
	if r.inheritanceColor != newInheritance {
		t.Errorf("inheritanceColor = %v, want %v", r.inheritanceColor, newInheritance)
	}
	if r.compositionColor != newComposition {
		t.Errorf("compositionColor = %v, want %v", r.compositionColor, newComposition)
	}
	if r.callColor != newCall {
		t.Errorf("callColor = %v, want %v", r.callColor, newCall)
	}
	if r.circularColor != newCircular {
		t.Errorf("circularColor = %v, want %v", r.circularColor, newCircular)
	}
}

func TestRendererSetWidthRange(t *testing.T) {
	r := NewRenderer()

	r.SetWidthRange(2.0, 10.0, 20)

	if r.minWidth != 2.0 {
		t.Errorf("minWidth = %f, want 2.0", r.minWidth)
	}
	if r.maxWidth != 10.0 {
		t.Errorf("maxWidth = %f, want 10.0", r.maxWidth)
	}
	if r.maxStrength != 20 {
		t.Errorf("maxStrength = %d, want 20", r.maxStrength)
	}
}

func TestRendererSetAnimationSpeed(t *testing.T) {
	r := NewRenderer()

	r.SetAnimationSpeed(60.0)

	if r.animSpeed != 60.0 {
		t.Errorf("animSpeed = %f, want 60.0", r.animSpeed)
	}
}

func TestRendererSetDashPattern(t *testing.T) {
	r := NewRenderer()

	r.SetDashPattern(16.0, 8.0)

	if r.dashLength != 16.0 {
		t.Errorf("dashLength = %f, want 16.0", r.dashLength)
	}
	if r.dashGap != 8.0 {
		t.Errorf("dashGap = %f, want 8.0", r.dashGap)
	}
}

func TestRendererDrawWithNilGraph(t *testing.T) {
	r := NewRenderer()

	// This should not panic
	r.Draw(nil, nil, nil, 0, 0)
}

func TestRendererDrawSkipsSelfReferences(t *testing.T) {
	// Self-references should be skipped
	// This is a logic test - we verify the behavior by checking that
	// a graph with only self-references doesn't cause issues

	r := NewRenderer()
	g := connection.NewGraph()
	g.AddNew("a.go", "a.go", connection.TypeImport) // Self-reference

	positions := map[string]TilePosition{
		"a.go": {X: 0, Y: 0, Width: 100, Height: 100},
	}

	// This should not panic and should effectively do nothing
	r.Draw(nil, g, positions, 0, 0)
}
