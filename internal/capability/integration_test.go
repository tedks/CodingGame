package capability

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestFullLifecycle_WatcherRegistryDiscoverer tests the complete integration
// between Watcher, Registry, and Discoverers through a realistic workflow.
func TestFullLifecycle_WatcherRegistryDiscoverer(t *testing.T) {
	// Setup: Create temp directory and config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "mcp.json")

	// Write initial config with one capability indicator
	if err := os.WriteFile(configPath, []byte(`{"servers":["server1"]}`), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}
	// Set initial mod time
	baseTime := time.Unix(100, 0)
	if err := os.Chtimes(configPath, baseTime, baseTime); err != nil {
		t.Fatalf("failed to set initial mod time: %v", err)
	}

	// Step 1: Create registry
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry returned nil")
	}

	// Step 2: Register discoverers
	// Add builtin discoverer
	builtinDisc := NewBuiltinToolDiscoverer()
	registry.RegisterDiscoverer(builtinDisc)

	// Add a file-based discoverer for testing
	fileDisc := &configFileDiscoverer{path: configPath}
	registry.RegisterDiscoverer(fileDisc)

	// Step 3: Add listener to track changes
	listener := &lifecycleListener{
		changes: make(chan capabilityChangeEvent, 10),
	}
	registry.AddListener(listener)

	// Step 4: Initial refresh
	count := registry.Refresh()
	if count == 0 {
		t.Error("expected some capabilities after initial refresh")
	}

	// Wait for listener notification
	select {
	case event := <-listener.changes:
		if len(event.capabilities) == 0 {
			t.Error("listener received empty capabilities on initial refresh")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("listener was not notified of initial refresh")
	}

	initialCount := registry.Count()
	t.Logf("Initial capability count: %d", initialCount)

	// Step 5: Start watcher
	watcher := NewWatcher(registry)
	manualTicker := newManualTicker()
	watcher.SetTickerFactory(func(time.Duration) ticker { return manualTicker })

	if err := watcher.Start(); err != nil {
		t.Fatalf("watcher.Start() returned error: %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("watcher should be running after Start()")
	}

	// Verify snapshot
	snapshot := watcher.Snapshot()
	if !snapshot.Running {
		t.Error("snapshot should show watcher as running")
	}

	// Step 6: Modify config file
	if err := os.WriteFile(configPath, []byte(`{"servers":["server1","server2"]}`), 0644); err != nil {
		t.Fatalf("failed to write modified config: %v", err)
	}
	// Update mod time to trigger change detection
	if err := os.Chtimes(configPath, baseTime.Add(time.Minute), baseTime.Add(time.Minute)); err != nil {
		t.Fatalf("failed to set new mod time: %v", err)
	}

	// Trigger polling
	manualTicker.Tick()

	// Step 7: Verify listener receives update
	select {
	case event := <-listener.changes:
		// The configFileDiscoverer should now return 2 capabilities from the file
		t.Logf("Received update with %d capabilities", len(event.capabilities))
	case <-time.After(1 * time.Second):
		t.Error("listener was not notified after config file change")
	}

	// Step 8: Stop watcher
	if err := watcher.Stop(); err != nil {
		t.Fatalf("watcher.Stop() returned error: %v", err)
	}

	if watcher.IsRunning() {
		t.Error("watcher should not be running after Stop()")
	}

	// Step 9: Restart watcher (verify restart capability)
	if err := watcher.Start(); err != nil {
		t.Fatalf("watcher.Start() after stop returned error: %v", err)
	}

	if !watcher.IsRunning() {
		t.Error("watcher should be running after restart")
	}

	// Step 10: ForceRefresh
	watcher.ForceRefresh()

	select {
	case <-listener.changes:
		// Good, received refresh notification
	case <-time.After(1 * time.Second):
		t.Error("listener was not notified after ForceRefresh")
	}

	// Cleanup
	if err := watcher.Stop(); err != nil {
		t.Errorf("final watcher.Stop() returned error: %v", err)
	}
}

// TestLifecycle_MultipleDiscoverersWithErrors tests graceful degradation
// when some discoverers fail.
func TestLifecycle_MultipleDiscoverersWithErrors(t *testing.T) {
	registry := NewRegistry()

	// Add working discoverer
	workingDisc := &mockDiscoverer{
		name: "working",
		capabilities: []*Capability{
			NewCapability("working-cap", "Working Cap", TypeTool, DomainCore),
		},
	}
	registry.RegisterDiscoverer(workingDisc)

	// Add failing discoverer
	failingDisc := &errorDiscoverer{
		name: "failing",
		err:  errTestDiscovery,
	}
	registry.RegisterDiscoverer(failingDisc)

	// Add another working discoverer
	anotherWorking := &mockDiscoverer{
		name: "another-working",
		capabilities: []*Capability{
			NewCapability("another-cap", "Another Cap", TypeMCP, DomainBuild),
		},
	}
	registry.RegisterDiscoverer(anotherWorking)

	// Refresh should still work (graceful degradation)
	count := registry.Refresh()
	if count != 2 {
		t.Errorf("expected 2 capabilities from working discoverers, got %d", count)
	}

	// Should have recorded the error
	if !registry.HasErrors() {
		t.Error("registry should report errors from failing discoverer")
	}

	errors := registry.LastErrors()
	if errors["failing"] == nil {
		t.Error("expected error from 'failing' discoverer")
	}
}

// TestLifecycle_ListenerRemovalDuringNotification tests that removing a listener
// during notification doesn't cause issues.
func TestLifecycle_ListenerRemovalDuringNotification(t *testing.T) {
	registry := NewRegistry()

	var removeCalled sync.WaitGroup
	removeCalled.Add(1)

	// Create a listener that removes itself when notified
	var selfRemovingListener *selfRemovingTestListener
	selfRemovingListener = &selfRemovingTestListener{
		registry: registry,
		onNotify: func() {
			registry.RemoveListener(selfRemovingListener)
			removeCalled.Done()
		},
	}

	registry.AddListener(selfRemovingListener)

	// Add a normal listener to verify notifications still work
	normalListener := &lifecycleListener{
		changes: make(chan capabilityChangeEvent, 1),
	}
	registry.AddListener(normalListener)

	// Add a discoverer
	registry.RegisterDiscoverer(&mockDiscoverer{
		name: "test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	})

	// Refresh triggers notification
	registry.Refresh()

	// Wait for self-removing listener
	removeCalled.Wait()

	// Verify normal listener still received notification
	select {
	case <-normalListener.changes:
		// Good
	case <-time.After(1 * time.Second):
		t.Error("normal listener should have received notification")
	}
}

// TestLifecycle_WatcherWithNoDiscoverers tests watcher behavior with empty registry.
func TestLifecycle_WatcherWithNoDiscoverers(t *testing.T) {
	registry := NewRegistry()
	watcher := NewWatcher(registry)

	manualTicker := newManualTicker()
	watcher.SetTickerFactory(func(time.Duration) ticker { return manualTicker })

	// Should start without error
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start() with empty registry returned error: %v", err)
	}

	// Tick should not panic with no watch paths
	manualTicker.Tick()

	// Snapshot should work
	snapshot := watcher.Snapshot()
	if !snapshot.Running {
		t.Error("watcher should be running")
	}
	if snapshot.TrackedPaths != 0 {
		t.Errorf("expected 0 tracked paths, got %d", snapshot.TrackedPaths)
	}

	if err := watcher.Stop(); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}
}

// configFileDiscoverer is a discoverer that reads from a config file.
// It simulates an MCP config discoverer.
type configFileDiscoverer struct {
	path string
}

func (d *configFileDiscoverer) Name() string {
	return "config-file"
}

func (d *configFileDiscoverer) Discover() ([]*Capability, error) {
	data, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Simple parsing: count "server" occurrences
	content := string(data)
	caps := []*Capability{}

	// Create a capability for each server mention
	count := 0
	for i := 0; i < len(content); i++ {
		if i+6 < len(content) && content[i:i+6] == "server" {
			count++
			caps = append(caps, NewCapability(
				"mcp-server-"+string(rune('0'+count)),
				"MCP Server "+string(rune('0'+count)),
				TypeMCP,
				DomainCore,
			).WithSource(d.path))
		}
	}

	return caps, nil
}

func (d *configFileDiscoverer) WatchPaths() []string {
	return []string{d.path}
}

// lifecycleListener records capability changes for testing.
type lifecycleListener struct {
	changes chan capabilityChangeEvent
}

type capabilityChangeEvent struct {
	capabilities []*Capability
	timestamp    time.Time
}

func (l *lifecycleListener) OnCapabilitiesChanged(capabilities []*Capability) {
	select {
	case l.changes <- capabilityChangeEvent{
		capabilities: capabilities,
		timestamp:    time.Now(),
	}:
	default:
		// Drop if channel is full
	}
}

// selfRemovingTestListener removes itself when notified.
type selfRemovingTestListener struct {
	registry *Registry
	onNotify func()
}

func (l *selfRemovingTestListener) OnCapabilitiesChanged([]*Capability) {
	if l.onNotify != nil {
		l.onNotify()
	}
}
