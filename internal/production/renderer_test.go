package production

import (
	"image/color"
	"testing"

	"github.com/tedks/CodingGame/internal/theme"
)

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	if r == nil {
		t.Fatal("NewRenderer returned nil")
	}
}

func TestGetHealthColor(t *testing.T) {
	tests := []struct {
		health   HealthStatus
		expected color.RGBA
	}{
		{HealthHealthy, theme.StatusSuccess},
		{HealthDegraded, theme.StatusWarning},
		{HealthUnhealthy, theme.StatusError},
		{HealthUnknown, theme.StatusNeutral},
	}

	for _, tc := range tests {
		c := GetHealthColor(tc.health)
		if c != tc.expected {
			t.Errorf("GetHealthColor(%v) = %v, expected %v", tc.health, c, tc.expected)
		}
	}
}

func TestWeatherDisplay(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		weather  Weather
		contains string
	}{
		{WeatherClear, "Clear"},
		{WeatherCloudy, "Cloudy"},
		{WeatherStorm, "STORM"},
		{WeatherDrought, "Drought"},
		{WeatherFlood, "FLOOD"},
	}

	for _, tc := range tests {
		display := r.weatherDisplay(tc.weather)
		if len(display) == 0 {
			t.Errorf("weatherDisplay(%v) returned empty string", tc.weather)
		}
		// Check that the weather name is in the display
		found := false
		for i := 0; i <= len(display)-len(tc.contains); i++ {
			if display[i:i+len(tc.contains)] == tc.contains {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("weatherDisplay(%v) = %q, expected to contain %q", tc.weather, display, tc.contains)
		}
	}
}

func TestWeatherSummaryXGuard(t *testing.T) {
	x := 10

	if _, ok := weatherSummaryX(x, 100); ok {
		t.Error("expected guard to skip weather summary for narrow width")
	}

	width := 500
	want := x + width - theme.ProductionWeatherSummaryWidth
	got, ok := weatherSummaryX(x, width)
	if !ok {
		t.Fatal("expected weather summary to render for wide width")
	}
	if got != want {
		t.Errorf("weatherSummaryX = %d, want %d", got, want)
	}
}

func TestLayoutConstants(t *testing.T) {
	// Verify layout constants are reasonable
	if theme.ProductionCityWidth <= 0 {
		t.Error("ProductionCityWidth should be positive")
	}
	if theme.ProductionCityHeight <= 0 {
		t.Error("ProductionCityHeight should be positive")
	}
	if theme.ProductionCitiesPerRow <= 0 {
		t.Error("ProductionCitiesPerRow should be positive")
	}
	if theme.PanelHeaderHeight <= 0 {
		t.Error("PanelHeaderHeight should be positive")
	}
}

func TestHealthColorsMapComplete(t *testing.T) {
	// Verify all health statuses have colors defined
	for _, status := range AllHealthStatuses() {
		if _, ok := healthColors[status]; !ok {
			t.Errorf("healthColors missing entry for %v", status)
		}
	}
}
