package production

import (
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	svc := NewService("api-gateway", "API Gateway", ServiceTypeHTTP, "http://localhost:8080")

	if svc.ID != "api-gateway" {
		t.Errorf("expected ID 'api-gateway', got %q", svc.ID)
	}
	if svc.Name != "API Gateway" {
		t.Errorf("expected Name 'API Gateway', got %q", svc.Name)
	}
	if svc.Type != ServiceTypeHTTP {
		t.Errorf("expected Type HTTP, got %v", svc.Type)
	}
	if svc.Endpoint != "http://localhost:8080" {
		t.Errorf("expected Endpoint 'http://localhost:8080', got %q", svc.Endpoint)
	}
	if svc.Health != HealthUnknown {
		t.Errorf("expected Health Unknown, got %v", svc.Health)
	}
	if svc.Weather != WeatherClear {
		t.Errorf("expected Weather Clear, got %v", svc.Weather)
	}
}

func TestServiceBuilderMethods(t *testing.T) {
	svc := NewService("test", "Test", ServiceTypeHTTP, "http://test").
		WithDependencies("dep1", "dep2").
		WithSource("/path/to/config")

	if len(svc.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(svc.Dependencies))
	}
	if svc.Dependencies[0] != "dep1" {
		t.Errorf("expected first dependency 'dep1', got %q", svc.Dependencies[0])
	}
	if svc.Source != "/path/to/config" {
		t.Errorf("expected Source '/path/to/config', got %q", svc.Source)
	}
}

func TestUpdateMetrics(t *testing.T) {
	svc := NewService("test", "Test", ServiceTypeHTTP, "http://test")

	metrics := TrafficMetrics{
		RequestsPerSecond: 100,
		ErrorRate:         0.5,
		LatencyP50:        50 * time.Millisecond,
		LatencyP99:        200 * time.Millisecond,
	}
	svc.UpdateMetrics(metrics)

	if svc.Metrics.RequestsPerSecond != 100 {
		t.Errorf("expected RPS 100, got %f", svc.Metrics.RequestsPerSecond)
	}
	if svc.Weather != WeatherClear {
		t.Errorf("expected Clear weather for healthy metrics, got %v", svc.Weather)
	}
}

func TestUpdateHealth(t *testing.T) {
	svc := NewService("test", "Test", ServiceTypeHTTP, "http://test")

	svc.UpdateHealth(HealthHealthy, "")
	if svc.Health != HealthHealthy {
		t.Errorf("expected Healthy, got %v", svc.Health)
	}
	if svc.LastChecked.IsZero() {
		t.Error("expected LastChecked to be set")
	}

	svc.UpdateHealth(HealthUnhealthy, "connection refused")
	if svc.Health != HealthUnhealthy {
		t.Errorf("expected Unhealthy, got %v", svc.Health)
	}
	if svc.LastError != "connection refused" {
		t.Errorf("expected LastError 'connection refused', got %q", svc.LastError)
	}
}

func TestDeriveWeather(t *testing.T) {
	tests := []struct {
		name     string
		metrics  TrafficMetrics
		expected Weather
	}{
		{
			name:     "clear - nominal metrics",
			metrics:  TrafficMetrics{RequestsPerSecond: 50, ErrorRate: 0.5, LatencyP99: 100 * time.Millisecond},
			expected: WeatherClear,
		},
		{
			name:     "cloudy - elevated error rate",
			metrics:  TrafficMetrics{RequestsPerSecond: 50, ErrorRate: 2.0, LatencyP99: 100 * time.Millisecond},
			expected: WeatherCloudy,
		},
		{
			name:     "cloudy - elevated latency",
			metrics:  TrafficMetrics{RequestsPerSecond: 50, ErrorRate: 0.5, LatencyP99: 700 * time.Millisecond},
			expected: WeatherCloudy,
		},
		{
			name:     "storm - high error rate",
			metrics:  TrafficMetrics{RequestsPerSecond: 50, ErrorRate: 10.0, LatencyP99: 100 * time.Millisecond},
			expected: WeatherStorm,
		},
		{
			name:     "storm - very high latency",
			metrics:  TrafficMetrics{RequestsPerSecond: 50, ErrorRate: 0.5, LatencyP99: 2 * time.Second},
			expected: WeatherStorm,
		},
		{
			name:     "drought - very low traffic",
			metrics:  TrafficMetrics{RequestsPerSecond: 0.05, ErrorRate: 0, LatencyP99: 0},
			expected: WeatherDrought,
		},
		{
			name:     "flood - high traffic",
			metrics:  TrafficMetrics{RequestsPerSecond: 1500, ErrorRate: 0.5, LatencyP99: 100 * time.Millisecond},
			expected: WeatherFlood,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveWeather(tc.metrics)
			if got != tc.expected {
				t.Errorf("deriveWeather() = %v, expected %v", got, tc.expected)
			}
		})
	}
}

func TestAllServiceTypes(t *testing.T) {
	types := AllServiceTypes()
	if len(types) != 5 {
		t.Errorf("expected 5 service types, got %d", len(types))
	}
}

func TestAllHealthStatuses(t *testing.T) {
	statuses := AllHealthStatuses()
	if len(statuses) != 4 {
		t.Errorf("expected 4 health statuses, got %d", len(statuses))
	}
}

func TestAllWeathers(t *testing.T) {
	weathers := AllWeathers()
	if len(weathers) != 5 {
		t.Errorf("expected 5 weather types, got %d", len(weathers))
	}
}
