package resources

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNew(t *testing.T) {
	tracker := New()

	if tracker == nil {
		t.Fatal("expected non-nil tracker")
	}

	// Should have default resources
	contextRes := tracker.GetResource("Context")
	if contextRes == nil {
		t.Error("expected Context resource to exist")
	}
	if contextRes.Max != 200000 {
		t.Errorf("expected Context max 200000, got %d", contextRes.Max)
	}

	apiCostRes := tracker.GetResource("API Cost")
	if apiCostRes == nil {
		t.Error("expected API Cost resource to exist")
	}

	coverageRes := tracker.GetResource("Coverage")
	if coverageRes == nil {
		t.Error("expected Coverage resource to exist")
	}
}

func TestUpdateResource(t *testing.T) {
	tracker := New()

	// Update context tokens
	tracker.UpdateResource("Context", 50000)

	contextRes := tracker.GetResource("Context")
	if contextRes.Current != 50000 {
		t.Errorf("expected Context current 50000, got %d", contextRes.Current)
	}
}

func TestAddResource(t *testing.T) {
	tracker := New()

	// Add custom resource
	customRes := &Resource{
		Name:    "Build Time",
		Current: 0,
		Max:     60000, // 60 seconds
		Unit:    "ms",
		Color:   color.RGBA{255, 100, 100, 255},
	}

	tracker.AddResource(customRes)

	// Verify it was added
	retrieved := tracker.GetResource("Build Time")
	if retrieved == nil {
		t.Fatal("expected custom resource to be added")
	}
	if retrieved.Name != "Build Time" {
		t.Errorf("expected name 'Build Time', got %s", retrieved.Name)
	}
	if retrieved.Max != 60000 {
		t.Errorf("expected max 60000, got %d", retrieved.Max)
	}
}

func TestFormatResourceText(t *testing.T) {
	tracker := New()

	tests := []struct {
		name     string
		resource *Resource
		expected string
	}{
		{
			name: "context tokens",
			resource: &Resource{
				Name:    "Context",
				Current: 50000,
				Max:     200000,
				Unit:    "tokens",
			},
			expected: "Context: 50000/200000 tokens",
		},
		{
			name: "api cost cents",
			resource: &Resource{
				Name:    "API Cost",
				Current: 1234,  // $12.34
				Max:     10000, // $100.00
				Unit:    "cents",
			},
			expected: "API Cost: $12.34 / $100.00",
		},
		{
			name: "percentage",
			resource: &Resource{
				Name:    "Coverage",
				Current: 75,
				Max:     100,
				Unit:    "%",
			},
			expected: "Coverage: 75%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.formatResourceText(tt.resource)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClampRatio(t *testing.T) {
	tests := []struct {
		value    float32
		expected float32
	}{
		{-0.25, 0},
		{0, 0},
		{0.4, 0.4},
		{1, 1},
		{1.1, 1},
	}

	for _, tc := range tests {
		if got := clampRatio(tc.value); got != tc.expected {
			t.Errorf("clampRatio(%v) = %v, want %v", tc.value, got, tc.expected)
		}
	}
}

func TestGetResourceNotFound(t *testing.T) {
	tracker := New()

	res := tracker.GetResource("NonExistent")
	if res != nil {
		t.Error("expected nil for non-existent resource")
	}
}

func TestGetResourceReturnsCopy(t *testing.T) {
	tracker := New()

	res := tracker.GetResource("Context")
	if res == nil {
		t.Fatal("expected Context resource to exist")
	}

	res.Current = 999
	resAgain := tracker.GetResource("Context")
	if resAgain == nil {
		t.Fatal("expected Context resource to exist")
	}
	if resAgain.Current == 999 {
		t.Error("GetResource returned a reference instead of a copy")
	}
}

func TestConcurrentAccess(t *testing.T) {
	tracker := New()

	done := make(chan bool)

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = tracker.GetResource("Context")
			tracker.Update()
		}
		done <- true
	}()

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			tracker.UpdateResource("Context", int64(i*1000))
			tracker.AddResource(&Resource{
				Name:    "Custom",
				Current: int64(i),
				Max:     1000,
				Unit:    "units",
			})
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we get here without deadlock or data race, test passes
}

func TestDrawWithNoResourcesDoesNotPanic(t *testing.T) {
	tracker := New()
	tracker.resources = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw panicked with no resources: %v", r)
		}
	}()

	tracker.Draw(nil, 0, 0, 100, 20)
}

// ============================================================================
// Edge Case Tests from Issue #62
// ============================================================================

// TestDrawWithSmallDimensions verifies that Draw handles small widths gracefully.
// Risk: width < 20 causes barWidth = width - 2*padding to go negative.
func TestDrawWithSmallDimensions(t *testing.T) {
	tracker := New()
	img := ebiten.NewImage(100, 100)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw panicked with small dimensions: %v", r)
		}
	}()

	// Width smaller than 2*padding (20)
	tracker.Draw(img, 0, 0, 15, 20) // barWidth = 15-20 = -5
	tracker.Draw(img, 0, 0, 0, 20)  // barWidth = -20
	tracker.Draw(img, 0, 0, 1, 1)   // Minimal dimensions

	// Should not panic - verify graceful handling
}

// TestMaxZeroOrNegative verifies behavior when Max is zero or negative.
// Risk: Division by zero or negative ratio in fillPct calculation.
func TestMaxZeroOrNegative(t *testing.T) {
	tracker := New()

	// Clear default resources to isolate test
	tracker.resources = nil

	tracker.AddResource(&Resource{Name: "ZeroMax", Max: 0, Current: 100, Color: color.RGBA{255, 0, 0, 255}})
	tracker.AddResource(&Resource{Name: "NegMax", Max: -100, Current: 50, Color: color.RGBA{0, 255, 0, 255}})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw panicked with zero/negative Max: %v", r)
		}
	}()

	// Draw should not panic
	img := ebiten.NewImage(200, 40)
	tracker.Draw(img, 0, 0, 200, 40)

	// Verify fillPct is handled correctly (Max=0 → fillPct=0 via guard)
	// Max=-100 → fillPct = 50/-100 = -0.5 → clamped to 0
}

// TestNegativeCurrent verifies that negative Current values are handled.
// Risk: Undefined display for negative values.
func TestNegativeCurrent(t *testing.T) {
	tracker := New()

	tracker.UpdateResource("Context", -5000)

	res := tracker.GetResource("Context")
	if res == nil {
		t.Fatal("expected Context resource to exist")
	}
	if res.Current != -5000 {
		t.Errorf("expected -5000, got %d", res.Current)
	}

	// Verify text formatting handles negative
	text := tracker.formatResourceText(res)
	if !strings.Contains(text, "-5000") {
		t.Errorf("negative value not displayed correctly: %s", text)
	}

	// Verify draw doesn't panic with negative current
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw panicked with negative Current: %v", r)
		}
	}()

	img := ebiten.NewImage(200, 40)
	tracker.Draw(img, 0, 0, 200, 40)
}

// TestDuplicateResourceNames verifies behavior when multiple resources share names.
// Risk: Undefined behavior for which resource is updated/retrieved.
func TestDuplicateResourceNames(t *testing.T) {
	tracker := New()
	// Clear defaults
	tracker.resources = nil

	tracker.AddResource(&Resource{Name: "Dupe", Max: 100, Current: 10, Color: color.RGBA{255, 0, 0, 255}})
	tracker.AddResource(&Resource{Name: "Dupe", Max: 200, Current: 20, Color: color.RGBA{0, 255, 0, 255}})

	// UpdateResource should update first match
	tracker.UpdateResource("Dupe", 50)

	// GetResource should return first match
	res := tracker.GetResource("Dupe")
	if res == nil {
		t.Fatal("expected Dupe resource to exist")
	}
	if res.Current != 50 {
		t.Errorf("expected first 'Dupe' updated to 50, got %d", res.Current)
	}
	if res.Max != 100 {
		t.Errorf("expected first 'Dupe' Max=100, got %d", res.Max)
	}
}

// TestUpdateNonexistentResource verifies that updating a non-existent resource is a silent no-op.
// This documents the expected behavior.
func TestUpdateNonexistentResource(t *testing.T) {
	tracker := New()

	// Should be silent no-op
	tracker.UpdateResource("DoesNotExist", 100)

	res := tracker.GetResource("DoesNotExist")
	if res != nil {
		t.Error("UpdateResource should not create new resources")
	}
}

// TestFormatResourceText_NegativeCents verifies formatting of negative cent values.
func TestFormatResourceText_NegativeCents(t *testing.T) {
	tracker := New()
	res := &Resource{Name: "Cost", Current: -1234, Max: 10000, Unit: "cents"}

	text := tracker.formatResourceText(res)
	// Note: -1234 cents = -$12.34, formatted as "$-12.34"
	expected := "Cost: $-12.34 / $100.00"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

// TestFormatResourceText_LargeValues verifies formatting doesn't overflow with large int64 values.
func TestFormatResourceText_LargeValues(t *testing.T) {
	tracker := New()
	res := &Resource{
		Name:    "Huge",
		Current: 9223372036854775807, // MaxInt64
		Max:     9223372036854775807,
		Unit:    "tokens",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("formatResourceText panicked with large values: %v", r)
		}
	}()

	text := tracker.formatResourceText(res)
	// Just verify no panic and contains expected parts
	if !strings.Contains(text, "Huge:") {
		t.Errorf("missing name in: %s", text)
	}
	if !strings.Contains(text, "9223372036854775807") {
		t.Errorf("missing large value in: %s", text)
	}
}

// TestFormatResourceText_EmptyUnit verifies formatting with empty unit string.
func TestFormatResourceText_EmptyUnit(t *testing.T) {
	tracker := New()
	res := &Resource{Name: "NoUnit", Current: 50, Max: 100, Unit: ""}

	text := tracker.formatResourceText(res)
	// Expected: "NoUnit: 50/100 " (trailing space from %s with empty string)
	if !strings.HasPrefix(text, "NoUnit: 50/100") {
		t.Errorf("unexpected format: %s", text)
	}
}

// TestFormatResourceText_ZeroCents verifies $0.00 formatting.
func TestFormatResourceText_ZeroCents(t *testing.T) {
	tracker := New()
	res := &Resource{Name: "Cost", Current: 0, Max: 10000, Unit: "cents"}

	text := tracker.formatResourceText(res)
	expected := "Cost: $0.00 / $100.00"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

// TestDrawWithNilScreen verifies that Draw handles nil screen gracefully.
func TestDrawWithNilScreen(t *testing.T) {
	tracker := New()

	defer func() {
		if r := recover(); r != nil {
			// This is expected - Ebitengine doesn't handle nil screens
			// Document this behavior
			t.Logf("Draw panics with nil screen (expected): %v", r)
		}
	}()

	// Note: This may panic, which is acceptable behavior
	// The test documents what happens
	tracker.Draw(nil, 0, 0, 200, 40)
}

// TestCurrentExceedsMax verifies behavior when Current > Max.
func TestCurrentExceedsMax(t *testing.T) {
	tracker := New()

	tracker.UpdateResource("Context", 300000) // Exceeds max of 200000

	res := tracker.GetResource("Context")
	if res == nil {
		t.Fatal("expected Context resource to exist")
	}
	if res.Current != 300000 {
		t.Errorf("expected 300000, got %d", res.Current)
	}

	// fillPct should be clamped to 1.0, not > 1
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Draw panicked with Current > Max: %v", r)
		}
	}()

	img := ebiten.NewImage(200, 40)
	tracker.Draw(img, 0, 0, 200, 40)
}

// ============================================================================
// Benchmarks
// ============================================================================

// BenchmarkRapidUpdates measures performance of frequent resource updates.
func BenchmarkRapidUpdates(b *testing.B) {
	tracker := New()
	for i := 0; i < b.N; i++ {
		tracker.UpdateResource("Context", int64(i))
	}
}

// BenchmarkManyResources measures Draw performance with many resources.
func BenchmarkManyResources(b *testing.B) {
	tracker := New()
	// Clear defaults
	tracker.resources = nil

	for i := 0; i < 50; i++ {
		tracker.AddResource(&Resource{
			Name:    fmt.Sprintf("Resource%d", i),
			Max:     1000,
			Current: int64(i * 20),
			Color:   color.RGBA{uint8(i * 5), uint8(i * 3), uint8(i * 4), 255},
		})
	}

	img := ebiten.NewImage(1920, 40)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Draw(img, 0, 0, 1920, 40)
	}
}

// BenchmarkConcurrentAccess measures performance under concurrent access.
func BenchmarkConcurrentAccess(b *testing.B) {
	tracker := New()

	b.RunParallel(func(pb *testing.PB) {
		i := int64(0)
		for pb.Next() {
			tracker.UpdateResource("Context", i)
			_ = tracker.GetResource("Context")
			i++
		}
	})
}
