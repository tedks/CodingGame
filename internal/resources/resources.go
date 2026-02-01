// Package resources provides real-time tracking and visualization of development
// metrics in an RTS-style resource bar. It tracks actual values like context tokens,
// API costs, test coverage, and build status rather than synthetic game metrics.
//
// The resource tracker is thread-safe and supports custom resources for extensibility.
// All metrics are displayed with visual progress bars and formatted text.
package resources

import (
	"fmt"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Resource represents a trackable metric displayed in the resource bar
type Resource struct {
	Name    string
	Current int64
	Max     int64
	Unit    string
	Color   color.RGBA
}

// Tracker manages the resource bar showing real metrics
type Tracker struct {
	mu        sync.RWMutex
	resources []*Resource

	// Display settings
	bgColor   color.RGBA
	textColor color.RGBA
}

// New creates a new resource tracker with default resources
func New() *Tracker {
	return &Tracker{
		resources: []*Resource{
			{
				Name:    "Context",
				Current: 0,
				Max:     200000, // 200k tokens
				Unit:    "tokens",
				Color:   color.RGBA{100, 150, 255, 255}, // Blue
			},
			{
				Name:    "API Cost",
				Current: 0,
				Max:     10000, // $100.00 (in cents)
				Unit:    "cents",
				Color:   color.RGBA{255, 200, 100, 255}, // Gold
			},
			{
				Name:    "Coverage",
				Current: 0,
				Max:     100, // Percentage
				Unit:    "%",
				Color:   color.RGBA{100, 255, 150, 255}, // Green
			},
		},
		bgColor:   color.RGBA{30, 30, 40, 255},
		textColor: color.RGBA{200, 200, 220, 255},
	}
}

// Update updates resource values (called each frame)
func (t *Tracker) Update() {
	// Future: Poll real metrics from various sources
	// - Context tokens from Claude process
	// - API cost from usage tracking
	// - Build status from build system
	// - Test coverage from coverage reports
	// - CI status from CI/CD system
}

// Draw renders the resource bar
func (t *Tracker) Draw(screen *ebiten.Image, x, y, width, height int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.resources) == 0 {
		return
	}

	// Draw background
	vector.DrawFilledRect(
		screen,
		float32(x),
		float32(y),
		float32(width),
		float32(height),
		t.bgColor,
		false,
	)

	// Draw resources horizontally
	resourceWidth := width / len(t.resources)
	for i, r := range t.resources {
		offsetX := x + i*resourceWidth
		t.drawResource(screen, r, offsetX, y, resourceWidth, height)
	}
}

// drawResource renders a single resource meter
func (t *Tracker) drawResource(screen *ebiten.Image, r *Resource, x, y, width, height int) {
	padding := 10

	// Calculate fill percentage
	fillPct := float32(0)
	if r.Max > 0 {
		fillPct = float32(r.Current) / float32(r.Max)
		if fillPct > 1.0 {
			fillPct = 1.0
		}
	}

	// Draw resource bar background
	barY := float32(y + height - 10)
	barHeight := float32(5)
	barWidth := float32(width - 2*padding)

	vector.DrawFilledRect(
		screen,
		float32(x+padding),
		barY,
		barWidth,
		barHeight,
		color.RGBA{50, 50, 60, 255},
		false,
	)

	// Draw resource bar fill
	if fillPct > 0 {
		vector.DrawFilledRect(
			screen,
			float32(x+padding),
			barY,
			barWidth*fillPct,
			barHeight,
			r.Color,
			false,
		)
	}

	// Draw resource text
	text := t.formatResourceText(r)
	ebitenutil.DebugPrintAt(screen, text, x+padding, y+5)
}

// formatResourceText formats the resource display text
func (t *Tracker) formatResourceText(r *Resource) string {
	switch r.Unit {
	case "cents":
		// Format as dollars
		dollars := float64(r.Current) / 100.0
		maxDollars := float64(r.Max) / 100.0
		return fmt.Sprintf("%s: $%.2f / $%.2f", r.Name, dollars, maxDollars)
	case "%":
		// Just show percentage
		return fmt.Sprintf("%s: %d%%", r.Name, r.Current)
	default:
		// Show current/max with unit
		return fmt.Sprintf("%s: %d/%d %s", r.Name, r.Current, r.Max, r.Unit)
	}
}

// UpdateResource updates a specific resource value by name
func (t *Tracker) UpdateResource(name string, current int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, r := range t.resources {
		if r.Name == name {
			r.Current = current
			return
		}
	}
}

// AddResource adds a new custom resource to track
func (t *Tracker) AddResource(r *Resource) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resources = append(t.resources, r)
}

// GetResource retrieves a resource by name
func (t *Tracker) GetResource(name string) *Resource {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, r := range t.resources {
		if r.Name == name {
			return r
		}
	}
	return nil
}
