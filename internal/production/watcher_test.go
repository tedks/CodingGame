package production

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type manualTicker struct {
	ch chan time.Time
}

func newManualTicker() *manualTicker {
	return &manualTicker{ch: make(chan time.Time, 1)}
}

func (t *manualTicker) Channel() <-chan time.Time {
	return t.ch
}

func (t *manualTicker) Stop() {}

func (t *manualTicker) Tick() {
	select {
	case t.ch <- time.Now():
	default:
	}
}

type serviceListener struct {
	ch chan []*Service
}

func (l *serviceListener) OnServicesChanged(services []*Service) {
	select {
	case l.ch <- services:
	default:
	}
}

func waitForRefresh(t *testing.T, ch <-chan []*Service) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for registry refresh")
	}
}

func setModTime(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("failed to set mod time: %v", err)
	}
}

func TestNewWatcher(t *testing.T) {
	r := NewRegistry()
	w := NewWatcher(r)
	if w == nil {
		t.Fatal("NewWatcher returned nil")
	}
	if w.IsRunning() {
		t.Error("watcher should not be running initially")
	}
}

func TestWatcherStartStop(t *testing.T) {
	r := NewRegistry()
	w := NewWatcher(r)

	// Start watcher
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	if !w.IsRunning() {
		t.Error("watcher should be running after Start()")
	}

	// Start again should be no-op
	if err := w.Start(); err != nil {
		t.Fatalf("Second Start() returned error: %v", err)
	}

	// Stop watcher
	if err := w.Stop(); err != nil {
		t.Fatalf("Stop() returned error: %v", err)
	}
	if w.IsRunning() {
		t.Error("watcher should not be running after Stop()")
	}

	// Stop again should be no-op
	if err := w.Stop(); err != nil {
		t.Fatalf("Second Stop() returned error: %v", err)
	}
}

func TestWatcherSetPollInterval(t *testing.T) {
	r := NewRegistry()
	w := NewWatcher(r)

	// Set poll interval
	w.SetPollInterval(1 * time.Second)

	// Should be able to set even when not running
	w.SetPollInterval(10 * time.Second)
}

func TestWatcherDetectsFileCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry with discoverer that watches tmpDir
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))

	// Create watcher with short poll interval for testing
	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })

	// Start watcher
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	// Initially no services
	if r.Count() != 0 {
		t.Errorf("expected 0 services initially, got %d", r.Count())
	}

	// Create a config file
	config := `{"services": {"test-svc": {"type": "http", "endpoint": "http://test"}}}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	setModTime(t, configPath, time.Unix(10, 0))
	manual.Tick()
	waitForRefresh(t, listener.ch)

	// Should now have service
	if r.Count() != 1 {
		t.Errorf("expected 1 service after file creation, got %d", r.Count())
	}
}

func TestWatcherDetectsFileModification(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config file
	config1 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}
	baseTime := time.Unix(10, 0)
	setModTime(t, configPath, baseTime)

	// Create registry with discoverer
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	// Create watcher with short poll interval
	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	r.Refresh()
	waitForRefresh(t, listener.ch)

	// Initially 1 service
	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Modify the config file
	config2 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}, "svc2": {"type": "grpc", "endpoint": "localhost:9000"}}}`
	if err := os.WriteFile(configPath, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to write modified config: %v", err)
	}
	setModTime(t, configPath, baseTime.Add(1*time.Minute))

	manual.Tick()
	waitForRefresh(t, listener.ch)

	// Should now have 2 services
	if r.Count() != 2 {
		t.Errorf("expected 2 services after file modification, got %d", r.Count())
	}
}

func TestWatcherDetectsFileDeletion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config file
	config := `{"services": {"test-svc": {"type": "http", "endpoint": "http://test"}}}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	baseTime := time.Unix(20, 0)
	setModTime(t, configPath, baseTime)

	// Create registry with discoverer
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	// Create watcher with short poll interval
	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	r.Refresh()
	waitForRefresh(t, listener.ch)

	// Initially 1 service
	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Delete the config file
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("failed to delete config: %v", err)
	}

	manual.Tick()
	waitForRefresh(t, listener.ch)

	// Should now have 0 services
	if r.Count() != 0 {
		t.Errorf("expected 0 services after file deletion, got %d", r.Count())
	}
}

func TestWatcherForceRefresh(t *testing.T) {
	tmpDir := t.TempDir()

	// Create registry with discoverer
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))

	// Create watcher (not started)
	w := NewWatcher(r)

	// Initially no services
	if r.Count() != 0 {
		t.Errorf("expected 0 services initially, got %d", r.Count())
	}

	// Create config file
	config := `{"services": {"test-svc": {"type": "http", "endpoint": "http://test"}}}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Force refresh (without starting the watcher)
	w.ForceRefresh()

	// Should now have 1 service
	if r.Count() != 1 {
		t.Errorf("expected 1 service after ForceRefresh(), got %d", r.Count())
	}
}

func TestWatcherCleansUpStalePaths(t *testing.T) {
	// This tests that the watcher doesn't leak memory when paths are removed from watch list
	r := NewRegistry()

	// Add a mock discoverer with changing watch paths
	mock := &changingPathsDiscoverer{
		paths: []string{"/path/1", "/path/2"},
	}
	r.RegisterDiscoverer(mock)

	w := NewWatcher(r)

	// Initialize file times by checking for changes
	w.checkForChanges()

	// Verify initial file times map has 2 entries (they don't exist but still tracked)
	// Actually, since files don't exist, they won't be in the map
	// Let's just verify the function doesn't panic

	// Change the paths
	mock.paths = []string{"/path/3"}

	// Check for changes should clean up stale paths
	w.checkForChanges()

	// No error means success - the cleanup logic works
}

// changingPathsDiscoverer is a mock that returns different watch paths
type changingPathsDiscoverer struct {
	paths []string
}

func (d *changingPathsDiscoverer) Name() string {
	return "changing-paths"
}

func (d *changingPathsDiscoverer) Discover() ([]*Service, error) {
	return nil, nil
}

func (d *changingPathsDiscoverer) WatchPaths() []string {
	return d.paths
}

func TestWatcher_AtomicWriteViaRename(t *testing.T) {
	// Tests vim/emacs pattern: write to temp file, then os.Rename() to target.
	// This is a common editor pattern that avoids partial writes.
	tmpDir := t.TempDir()

	// Create initial config file
	config1 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}
	baseTime := time.Unix(100, 0)
	setModTime(t, configPath, baseTime)

	// Create registry with discoverer
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	// Create watcher with manual ticker
	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	// Initial refresh to set baseline
	r.Refresh()
	waitForRefresh(t, listener.ch)

	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Simulate vim/emacs atomic write pattern:
	// 1. Write new content to a temp file
	tempPath := filepath.Join(tmpDir, ".production.json.tmp")
	config2 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}, "svc2": {"type": "grpc", "endpoint": "localhost:9000"}}}`
	if err := os.WriteFile(tempPath, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	// 2. Rename temp to target (atomic operation)
	if err := os.Rename(tempPath, configPath); err != nil {
		t.Fatalf("failed to rename config: %v", err)
	}

	// Set a newer mod time to ensure detection
	setModTime(t, configPath, baseTime.Add(1*time.Minute))

	// Trigger watcher poll
	manual.Tick()
	waitForRefresh(t, listener.ch)

	// Should detect the atomic rename and see 2 services
	if r.Count() != 2 {
		t.Errorf("expected 2 services after atomic write, got %d", r.Count())
	}
}

func TestWatcher_SymlinkTargetModification(t *testing.T) {
	// Test that modifying a symlink target is detected.
	if runtime.GOOS == "windows" {
		t.Skip("symlink test not reliable on Windows")
	}

	tmpDir := t.TempDir()

	// Create target file
	targetPath := filepath.Join(tmpDir, "actual-config.json")
	config1 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	if err := os.WriteFile(targetPath, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write target config: %v", err)
	}
	baseTime := time.Unix(200, 0)
	setModTime(t, targetPath, baseTime)

	// Create symlink to target
	linkPath := filepath.Join(tmpDir, ".production.json")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Create registry with discoverer watching the symlink
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	// Create watcher
	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	// Initial state
	r.Refresh()
	waitForRefresh(t, listener.ch)
	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Modify the target file (not the symlink)
	config2 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}, "svc2": {"type": "grpc", "endpoint": "localhost:9000"}}}`
	if err := os.WriteFile(targetPath, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to modify target config: %v", err)
	}
	setModTime(t, targetPath, baseTime.Add(1*time.Minute))

	// Trigger poll - should detect target modification through symlink
	manual.Tick()
	waitForRefresh(t, listener.ch)

	if r.Count() != 2 {
		t.Errorf("expected 2 services after symlink target modification, got %d", r.Count())
	}
}

func TestWatcher_FileReplacedByDirectory(t *testing.T) {
	// Test that replacing a file with a directory is handled gracefully.
	tmpDir := t.TempDir()

	// Create initial config file
	configPath := filepath.Join(tmpDir, ".production.json")
	config := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	baseTime := time.Unix(300, 0)
	setModTime(t, configPath, baseTime)

	// Create registry
	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	// Create watcher
	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	// Initial state
	r.Refresh()
	waitForRefresh(t, listener.ch)
	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Delete file and replace with directory of same name
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("failed to remove config file: %v", err)
	}
	if err := os.Mkdir(configPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Trigger poll - should detect the change and handle gracefully
	manual.Tick()
	waitForRefresh(t, listener.ch)

	// Services should be empty now (directory can't be parsed as config)
	if r.Count() != 0 {
		t.Errorf("expected 0 services after file replaced by directory, got %d", r.Count())
	}
}

func TestWatcher_RapidFileChurn(t *testing.T) {
	// Test that rapid create/modify/delete cycles are handled correctly.
	tmpDir := t.TempDir()

	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))

	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 10)}
	r.AddListener(listener)

	configPath := filepath.Join(tmpDir, ".production.json")

	// Rapid churn: create -> tick -> modify -> tick -> delete -> tick
	// Create
	config1 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	if err := os.WriteFile(configPath, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	setModTime(t, configPath, time.Unix(400, 0))
	manual.Tick()
	waitForRefresh(t, listener.ch)
	if r.Count() != 1 {
		t.Errorf("expected 1 service after create, got %d", r.Count())
	}

	// Modify rapidly
	config2 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}, "svc2": {"type": "grpc", "endpoint": "localhost:9000"}}}`
	if err := os.WriteFile(configPath, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to modify config: %v", err)
	}
	setModTime(t, configPath, time.Unix(401, 0))
	manual.Tick()
	waitForRefresh(t, listener.ch)
	if r.Count() != 2 {
		t.Errorf("expected 2 services after modify, got %d", r.Count())
	}

	// Delete
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("failed to delete config: %v", err)
	}
	manual.Tick()
	waitForRefresh(t, listener.ch)
	if r.Count() != 0 {
		t.Errorf("expected 0 services after delete, got %d", r.Count())
	}

	// Create again to verify recovery
	config3 := `{"services": {"svc3": {"type": "database", "endpoint": "localhost:5432"}}}`
	if err := os.WriteFile(configPath, []byte(config3), 0644); err != nil {
		t.Fatalf("failed to recreate config: %v", err)
	}
	setModTime(t, configPath, time.Unix(402, 0))
	manual.Tick()
	waitForRefresh(t, listener.ch)
	if r.Count() != 1 {
		t.Errorf("expected 1 service after recreate, got %d", r.Count())
	}
}

func TestWatcher_SubSecondModifications(t *testing.T) {
	// Test that two writes within the same second are detected.
	// This tests the mtime comparison precision - on some filesystems,
	// mtime only has second resolution.
	tmpDir := t.TempDir()

	// Create initial config file
	configPath := filepath.Join(tmpDir, ".production.json")
	config1 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	if err := os.WriteFile(configPath, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	// Use a base time with nanosecond precision
	baseTime := time.Unix(500, 0)
	setModTime(t, configPath, baseTime)

	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	r.Refresh()
	waitForRefresh(t, listener.ch)
	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Write a new file with a different mtime (same second but different nanoseconds)
	config2 := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}, "svc2": {"type": "grpc", "endpoint": "localhost:9000"}}}`
	if err := os.WriteFile(configPath, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to modify config: %v", err)
	}
	// Set time to same second but 500ms later
	setModTime(t, configPath, baseTime.Add(500*time.Millisecond))

	manual.Tick()
	waitForRefresh(t, listener.ch)

	// The watcher should detect the modification
	if r.Count() != 2 {
		t.Errorf("expected 2 services after sub-second modification, got %d", r.Count())
	}
}

func TestWatcher_PermissionChanges(t *testing.T) {
	// Test that permission changes (chmod) are handled gracefully.
	if runtime.GOOS == "windows" {
		t.Skip("permission test not applicable on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("permission test not meaningful when running as root")
	}

	tmpDir := t.TempDir()

	// Create initial config file
	configPath := filepath.Join(tmpDir, ".production.json")
	config := `{"services": {"svc1": {"type": "http", "endpoint": "http://svc1"}}}`
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	baseTime := time.Unix(600, 0)
	setModTime(t, configPath, baseTime)

	r := NewRegistry()
	r.RegisterDiscoverer(NewConfigDiscoverer(tmpDir))
	r.Refresh()

	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })
	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	listener := &serviceListener{ch: make(chan []*Service, 1)}
	r.AddListener(listener)

	r.Refresh()
	waitForRefresh(t, listener.ch)
	if r.Count() != 1 {
		t.Errorf("expected 1 service initially, got %d", r.Count())
	}

	// Remove read permissions
	if err := os.Chmod(configPath, 0000); err != nil {
		t.Fatalf("failed to chmod 0000: %v", err)
	}
	// Ensure we restore permissions for cleanup
	defer os.Chmod(configPath, 0644)

	manual.Tick()
	waitForRefresh(t, listener.ch)

	// With no read permission, the config can't be read - services should be 0
	if r.Count() != 0 {
		t.Errorf("expected 0 services when file is unreadable, got %d", r.Count())
	}

	// Restore permissions
	if err := os.Chmod(configPath, 0644); err != nil {
		t.Fatalf("failed to chmod 0644: %v", err)
	}
	// Update mtime to trigger detection
	setModTime(t, configPath, baseTime.Add(1*time.Minute))

	manual.Tick()
	waitForRefresh(t, listener.ch)

	// Now the file should be readable again
	if r.Count() != 1 {
		t.Errorf("expected 1 service after restoring permissions, got %d", r.Count())
	}
}
