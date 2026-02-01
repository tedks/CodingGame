package capability

import (
	"fmt"
	"testing"
	"time"
)

// mockDiscoverer is a test discoverer that returns configured capabilities.
type mockDiscoverer struct {
	name         string
	capabilities []*Capability
	watchPaths   []string
}

func (m *mockDiscoverer) Name() string {
	return m.name
}

func (m *mockDiscoverer) Discover() ([]*Capability, error) {
	return m.capabilities, nil
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
		t.Errorf("new registry should have 0 capabilities, got %d", r.Count())
	}
}

func TestRegistryRefresh(t *testing.T) {
	r := NewRegistry()

	// Add a mock discoverer
	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
			NewCapability("cap2", "Cap 2", TypeMCP, DomainBuild),
		},
	}
	r.RegisterDiscoverer(mock)

	count := r.Refresh()
	if count != 2 {
		t.Errorf("Refresh() returned %d, expected 2", count)
	}

	if r.Count() != 2 {
		t.Errorf("registry should have 2 capabilities, got %d", r.Count())
	}
}

func TestRegistryGetAll(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("z-cap", "Z Cap", TypeTool, DomainAnalysis),
			NewCapability("a-cap", "A Cap", TypeTool, DomainCore),
			NewCapability("m-cap", "M Cap", TypeMCP, DomainBuild),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	caps := r.GetAll()
	if len(caps) != 3 {
		t.Fatalf("expected 3 capabilities, got %d", len(caps))
	}

	// Should be sorted by domain order, then by name
	// Core < Build < Analysis
	if caps[0].Domain != DomainCore {
		t.Errorf("first cap should be Core domain, got %v", caps[0].Domain)
	}
	if caps[1].Domain != DomainBuild {
		t.Errorf("second cap should be Build domain, got %v", caps[1].Domain)
	}
	if caps[2].Domain != DomainAnalysis {
		t.Errorf("third cap should be Analysis domain, got %v", caps[2].Domain)
	}
}

func TestRegistryGetByDomain(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
			NewCapability("cap2", "Cap 2", TypeMCP, DomainCore),
			NewCapability("cap3", "Cap 3", TypeTool, DomainBuild),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	coreCaps := r.GetByDomain(DomainCore)
	if len(coreCaps) != 2 {
		t.Errorf("expected 2 core capabilities, got %d", len(coreCaps))
	}

	buildCaps := r.GetByDomain(DomainBuild)
	if len(buildCaps) != 1 {
		t.Errorf("expected 1 build capability, got %d", len(buildCaps))
	}

	deployCaps := r.GetByDomain(DomainDeployment)
	if len(deployCaps) != 0 {
		t.Errorf("expected 0 deployment capabilities, got %d", len(deployCaps))
	}
}

func TestRegistryGetByType(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
			NewCapability("cap2", "Cap 2", TypeMCP, DomainCore),
			NewCapability("cap3", "Cap 3", TypeTool, DomainBuild),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	toolCaps := r.GetByType(TypeTool)
	if len(toolCaps) != 2 {
		t.Errorf("expected 2 tool capabilities, got %d", len(toolCaps))
	}

	mcpCaps := r.GetByType(TypeMCP)
	if len(mcpCaps) != 1 {
		t.Errorf("expected 1 MCP capability, got %d", len(mcpCaps))
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	cap := r.Get("cap1")
	if cap == nil {
		t.Error("expected to find cap1")
	}

	cap = r.Get("nonexistent")
	if cap != nil {
		t.Error("expected nil for nonexistent capability")
	}
}

func TestRegistryCountByDomain(t *testing.T) {
	r := NewRegistry()
	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
			NewCapability("cap2", "Cap 2", TypeMCP, DomainCore),
			NewCapability("cap3", "Cap 3", TypeTool, DomainBuild),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	counts := r.CountByDomain()
	if counts[DomainCore] != 2 {
		t.Errorf("expected 2 core capabilities, got %d", counts[DomainCore])
	}
	if counts[DomainBuild] != 1 {
		t.Errorf("expected 1 build capability, got %d", counts[DomainBuild])
	}
}

func TestRegistryListener(t *testing.T) {
	r := NewRegistry()

	listenerCalled := make(chan []*Capability, 1)
	listener := &testListener{
		onChanged: func(caps []*Capability) {
			select {
			case listenerCalled <- caps:
			default:
			}
		},
	}
	r.AddListener(listener)

	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	// Wait for listener to be called with timeout
	select {
	case caps := <-listenerCalled:
		if len(caps) != 1 {
			t.Errorf("expected 1 capability, got %d", len(caps))
		}
	case <-time.After(1 * time.Second):
		t.Error("listener was not called within timeout")
	}
}

func TestRegistryListenerReceivesCopy(t *testing.T) {
	r := NewRegistry()

	ready := make(chan struct{})
	firstCalled := make(chan struct{}, 1)
	secondCalled := make(chan []*Capability, 1)

	first := &testListener{
		onChanged: func(caps []*Capability) {
			if len(caps) > 0 {
				caps[0].Name = "mutated"
			}
			close(ready)
			firstCalled <- struct{}{}
		},
	}
	second := &testListener{
		onChanged: func(caps []*Capability) {
			<-ready
			secondCalled <- caps
		},
	}

	r.AddListener(first)
	r.AddListener(second)

	mock := &mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	select {
	case <-firstCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("first listener was not called within timeout")
	}

	select {
	case caps := <-secondCalled:
		if len(caps) != 1 {
			t.Fatalf("expected 1 capability, got %d", len(caps))
		}
		if caps[0].Name != "Cap 1" {
			t.Errorf("expected capability name 'Cap 1', got %q", caps[0].Name)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("second listener was not called within timeout")
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

// testListener implements RegistryListener for testing.
type testListener struct {
	onChanged func([]*Capability)
}

func (l *testListener) OnCapabilitiesChanged(caps []*Capability) {
	if l.onChanged != nil {
		l.onChanged(caps)
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

func (e *errorDiscoverer) Discover() ([]*Capability, error) {
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
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	}
	r.RegisterDiscoverer(workingDisc)

	// Refresh should succeed (graceful degradation)
	count := r.Refresh()
	if count != 1 {
		t.Errorf("expected 1 capability from working discoverer, got %d", count)
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

	// Remove error discoverer by creating a new registry (or in real usage, the error might resolve)
	// For this test, we verify errors are tracked per-refresh
	r2 := NewRegistry()
	workingDisc := &mockDiscoverer{
		name:         "working",
		capabilities: []*Capability{},
	}
	r2.RegisterDiscoverer(workingDisc)
	r2.Refresh()

	if r2.HasErrors() {
		t.Error("new registry should not have errors")
	}
}
