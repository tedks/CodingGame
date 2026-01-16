package resources

import (
	"image/color"
	"testing"
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

func TestGetResourceNotFound(t *testing.T) {
	tracker := New()

	res := tracker.GetResource("NonExistent")
	if res != nil {
		t.Error("expected nil for non-existent resource")
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
