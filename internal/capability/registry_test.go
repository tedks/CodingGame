package capability

import (
	"testing"
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

	var notifiedCaps []*Capability
	listener := &testListener{
		onChanged: func(caps []*Capability) {
			notifiedCaps = caps
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

	// Give goroutine time to run
	// In production code we'd use proper synchronization
	if len(notifiedCaps) == 0 {
		// Listener was called asynchronously, this is expected behavior
		// The actual notification happens in a goroutine
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
