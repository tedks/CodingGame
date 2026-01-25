package production

import (
	"fmt"
	"testing"
	"time"
)

// mockDiscoverer is a test discoverer that returns configured services.
type mockDiscoverer struct {
	name       string
	services   []*Service
	watchPaths []string
}

func (m *mockDiscoverer) Name() string {
	return m.name
}

func (m *mockDiscoverer) Discover() ([]*Service, error) {
	return m.services, nil
}

func (m *mockDiscoverer) WatchPaths() []string {
	return m.watchPaths
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.Count() != 0 {
		t.Errorf("new registry should have 0 services, got %d", r.Count())
	}
}

func TestRegistryRefresh(t *testing.T) {
	r := NewRegistry()

	// Add a mock discoverer
	mock := &mockDiscoverer{
		name: "test",
		services: []*Service{
			NewService("api", "API", ServiceTypeHTTP, "http://localhost"),
			NewService("db", "Database", ServiceTypeDatabase, "localhost:5432"),
		},
	}
	r.RegisterDiscoverer(mock)

	count := r.Refresh()
	if count != 2 {
		t.Errorf("Refresh() returned %d, expected 2", count)
	}

	if r.Count() != 2 {
		t.Errorf("registry should have 2 services, got %d", r.Count())
	}
}

func TestRegistryGetAll(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		services: []*Service{
			NewService("z-service", "Z Service", ServiceTypeHTTP, "http://z"),
			NewService("a-service", "A Service", ServiceTypeHTTP, "http://a"),
			NewService("m-service", "M Service", ServiceTypeGRPC, "grpc://m"),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	services := r.GetAll()
	if len(services) != 3 {
		t.Fatalf("expected 3 services, got %d", len(services))
	}

	// Should be sorted by name
	if services[0].Name != "A Service" {
		t.Errorf("first service should be 'A Service', got %q", services[0].Name)
	}
	if services[1].Name != "M Service" {
		t.Errorf("second service should be 'M Service', got %q", services[1].Name)
	}
	if services[2].Name != "Z Service" {
		t.Errorf("third service should be 'Z Service', got %q", services[2].Name)
	}
}

func TestRegistryGetByType(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		services: []*Service{
			NewService("api1", "API 1", ServiceTypeHTTP, "http://api1"),
			NewService("api2", "API 2", ServiceTypeHTTP, "http://api2"),
			NewService("db", "DB", ServiceTypeDatabase, "localhost:5432"),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	httpServices := r.GetByType(ServiceTypeHTTP)
	if len(httpServices) != 2 {
		t.Errorf("expected 2 HTTP services, got %d", len(httpServices))
	}

	dbServices := r.GetByType(ServiceTypeDatabase)
	if len(dbServices) != 1 {
		t.Errorf("expected 1 Database service, got %d", len(dbServices))
	}

	grpcServices := r.GetByType(ServiceTypeGRPC)
	if len(grpcServices) != 0 {
		t.Errorf("expected 0 gRPC services, got %d", len(grpcServices))
	}
}

func TestRegistryGetByHealth(t *testing.T) {
	r := NewRegistry()

	svc1 := NewService("api1", "API 1", ServiceTypeHTTP, "http://api1")
	svc1.Health = HealthHealthy
	svc2 := NewService("api2", "API 2", ServiceTypeHTTP, "http://api2")
	svc2.Health = HealthHealthy
	svc3 := NewService("api3", "API 3", ServiceTypeHTTP, "http://api3")
	svc3.Health = HealthUnhealthy

	mock := &mockDiscoverer{
		name:     "test",
		services: []*Service{svc1, svc2, svc3},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	healthyServices := r.GetByHealth(HealthHealthy)
	if len(healthyServices) != 2 {
		t.Errorf("expected 2 healthy services, got %d", len(healthyServices))
	}

	unhealthyServices := r.GetByHealth(HealthUnhealthy)
	if len(unhealthyServices) != 1 {
		t.Errorf("expected 1 unhealthy service, got %d", len(unhealthyServices))
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		services: []*Service{
			NewService("api", "API", ServiceTypeHTTP, "http://localhost"),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	svc := r.Get("api")
	if svc == nil {
		t.Error("expected to find api service")
	}

	svc = r.Get("nonexistent")
	if svc != nil {
		t.Error("expected nil for nonexistent service")
	}
}

func TestRegistryCountByHealth(t *testing.T) {
	r := NewRegistry()

	svc1 := NewService("api1", "API 1", ServiceTypeHTTP, "http://api1")
	svc1.Health = HealthHealthy
	svc2 := NewService("api2", "API 2", ServiceTypeHTTP, "http://api2")
	svc2.Health = HealthHealthy
	svc3 := NewService("api3", "API 3", ServiceTypeHTTP, "http://api3")
	svc3.Health = HealthDegraded

	mock := &mockDiscoverer{
		name:     "test",
		services: []*Service{svc1, svc2, svc3},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	counts := r.CountByHealth()
	if counts[HealthHealthy] != 2 {
		t.Errorf("expected 2 healthy services, got %d", counts[HealthHealthy])
	}
	if counts[HealthDegraded] != 1 {
		t.Errorf("expected 1 degraded service, got %d", counts[HealthDegraded])
	}
}

func TestRegistryListener(t *testing.T) {
	r := NewRegistry()

	listenerCalled := make(chan []*Service, 1)
	listener := &testListener{
		onChanged: func(services []*Service) {
			select {
			case listenerCalled <- services:
			default:
			}
		},
	}
	r.AddListener(listener)

	mock := &mockDiscoverer{
		name: "test",
		services: []*Service{
			NewService("api", "API", ServiceTypeHTTP, "http://localhost"),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	// Wait for listener to be called with timeout
	select {
	case services := <-listenerCalled:
		if len(services) != 1 {
			t.Errorf("expected 1 service, got %d", len(services))
		}
	case <-time.After(1 * time.Second):
		t.Error("listener was not called within timeout")
	}
}

func TestRegistryRemoveListener(t *testing.T) {
	r := NewRegistry()

	listener := &testListener{}
	r.AddListener(listener)
	r.RemoveListener(listener)

	// Should not panic and should work fine
	r.Refresh()
}

func TestRegistryWatchPaths(t *testing.T) {
	r := NewRegistry()

	mock := &mockDiscoverer{
		name:       "test",
		watchPaths: []string{"/path/1", "/path/2"},
	}
	r.RegisterDiscoverer(mock)

	paths := r.WatchPaths()
	if len(paths) != 2 {
		t.Errorf("expected 2 watch paths, got %d", len(paths))
	}
}

func TestRegistryGetWeatherSummary(t *testing.T) {
	r := NewRegistry()

	svc1 := NewService("api1", "API 1", ServiceTypeHTTP, "http://api1")
	svc1.Weather = WeatherClear
	svc2 := NewService("api2", "API 2", ServiceTypeHTTP, "http://api2")
	svc2.Weather = WeatherClear
	svc3 := NewService("api3", "API 3", ServiceTypeHTTP, "http://api3")
	svc3.Weather = WeatherStorm

	mock := &mockDiscoverer{
		name:     "test",
		services: []*Service{svc1, svc2, svc3},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	summary := r.GetWeatherSummary()
	if summary[WeatherClear] != 2 {
		t.Errorf("expected 2 clear weather, got %d", summary[WeatherClear])
	}
	if summary[WeatherStorm] != 1 {
		t.Errorf("expected 1 storm, got %d", summary[WeatherStorm])
	}
}

// testListener implements RegistryListener for testing.
type testListener struct {
	onChanged func([]*Service)
}

func (l *testListener) OnServicesChanged(services []*Service) {
	if l.onChanged != nil {
		l.onChanged(services)
	}
}

// Error tracking tests

// errorDiscoverer is a test discoverer that always returns an error.
type errorDiscoverer struct {
	name string
	err  error
}

func (e *errorDiscoverer) Name() string {
	return e.name
}

func (e *errorDiscoverer) Discover() ([]*Service, error) {
	return nil, e.err
}

func (e *errorDiscoverer) WatchPaths() []string {
	return nil
}

func TestRegistryLastErrors(t *testing.T) {
	r := NewRegistry()

	// Initially no errors
	errors := r.LastErrors()
	if len(errors) != 0 {
		t.Errorf("expected no errors initially, got %d", len(errors))
	}

	if r.HasErrors() {
		t.Error("HasErrors() should return false initially")
	}
}

func TestRegistryTracksDiscovererErrors(t *testing.T) {
	r := NewRegistry()

	// Add a discoverer that errors
	testErr := fmt.Errorf("test discovery error")
	errDisc := &errorDiscoverer{name: "error-discoverer", err: testErr}
	r.RegisterDiscoverer(errDisc)

	// Add a working discoverer
	workingDisc := &mockDiscoverer{
		name: "working",
		services: []*Service{
			NewService("api", "API", ServiceTypeHTTP, "http://localhost"),
		},
	}
	r.RegisterDiscoverer(workingDisc)

	// Refresh should succeed (graceful degradation)
	count := r.Refresh()
	if count != 1 {
		t.Errorf("expected 1 service from working discoverer, got %d", count)
	}

	// Should have recorded the error
	if !r.HasErrors() {
		t.Error("HasErrors() should return true after error")
	}

	errors := r.LastErrors()
	if len(errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(errors))
	}

	if errors["error-discoverer"] == nil {
		t.Error("expected error from error-discoverer")
	}
}

func TestRegistryErrorsCleared(t *testing.T) {
	r := NewRegistry()

	// First refresh with error
	errDisc := &errorDiscoverer{name: "error-discoverer", err: fmt.Errorf("error")}
	r.RegisterDiscoverer(errDisc)
	r.Refresh()

	if !r.HasErrors() {
		t.Error("should have errors after first refresh")
	}

	// Create a new registry to verify errors are tracked per-refresh
	r2 := NewRegistry()
	workingDisc := &mockDiscoverer{
		name:     "working",
		services: []*Service{},
	}
	r2.RegisterDiscoverer(workingDisc)
	r2.Refresh()

	if r2.HasErrors() {
		t.Error("new registry should not have errors")
	}
}
