package production

import (
	"testing"
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
		expected [4]uint8 // R, G, B, A
	}{
		{HealthHealthy, [4]uint8{0x2E, 0x7D, 0x32, 0xFF}},   // Green
		{HealthDegraded, [4]uint8{0xF5, 0x7F, 0x17, 0xFF}},  // Orange
		{HealthUnhealthy, [4]uint8{0xC6, 0x28, 0x28, 0xFF}}, // Red
		{HealthUnknown, [4]uint8{0x75, 0x75, 0x75, 0xFF}},   // Gray
	}

	for _, tc := range tests {
		c := GetHealthColor(tc.health)
		if c.R != tc.expected[0] || c.G != tc.expected[1] || c.B != tc.expected[2] || c.A != tc.expected[3] {
			t.Errorf("GetHealthColor(%v) = (%d,%d,%d,%d), expected (%d,%d,%d,%d)",
				tc.health, c.R, c.G, c.B, c.A,
				tc.expected[0], tc.expected[1], tc.expected[2], tc.expected[3])
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

func TestLayoutConstants(t *testing.T) {
	// Verify layout constants are reasonable
	if prodCityWidth <= 0 {
		t.Error("prodCityWidth should be positive")
	}
	if prodCityHeight <= 0 {
		t.Error("prodCityHeight should be positive")
	}
	if prodCitiesPerRow <= 0 {
		t.Error("prodCitiesPerRow should be positive")
	}
	if prodHeaderHeight <= 0 {
		t.Error("prodHeaderHeight should be positive")
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

func TestWeatherSymbolsMapComplete(t *testing.T) {
	// Verify all weathers have symbols defined
	for _, weather := range AllWeathers() {
		if _, ok := weatherSymbols[weather]; !ok {
			t.Errorf("weatherSymbols missing entry for %v", weather)
		}
	}
}
