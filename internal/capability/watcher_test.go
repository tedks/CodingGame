package capability

import (
	"os"
	"path/filepath"
	"strings"
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

type capabilityListener struct {
	ch chan []*Capability
}

func (l *capabilityListener) OnCapabilitiesChanged(capabilities []*Capability) {
	select {
	case l.ch <- capabilities:
	default:
	}
}

func waitForRefresh(t *testing.T, ch <-chan []*Capability) {
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

type fileDiscoverer struct {
	path string
}

func (d *fileDiscoverer) Name() string {
	return "test"
}

func (d *fileDiscoverer) Discover() ([]*Capability, error) {
	data, err := os.ReadFile(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	caps := []*Capability{
		NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
	}
	if strings.Contains(string(data), "cap2") {
		caps = append(caps, NewCapability("cap2", "Cap 2", TypeMCP, DomainBuild))
	}

	return caps, nil
}

func (d *fileDiscoverer) WatchPaths() []string {
	return []string{d.path}
}

func TestWatcherDetectsFileCreation(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "capabilities.txt")

	r := NewRegistry()
	r.RegisterDiscoverer(&fileDiscoverer{path: configPath})

	listener := &capabilityListener{ch: make(chan []*Capability, 1)}
	r.AddListener(listener)

	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })

	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	if r.Count() != 0 {
		t.Errorf("expected 0 capabilities initially, got %d", r.Count())
	}

	if err := os.WriteFile(configPath, []byte("cap1"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	setModTime(t, configPath, time.Unix(10, 0))

	manual.Tick()
	waitForRefresh(t, listener.ch)

	if r.Count() != 1 {
		t.Errorf("expected 1 capability after file creation, got %d", r.Count())
	}
}

func TestWatcherDetectsFileModification(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "capabilities.txt")

	if err := os.WriteFile(configPath, []byte("cap1"), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}
	baseTime := time.Unix(20, 0)
	setModTime(t, configPath, baseTime)

	r := NewRegistry()
	r.RegisterDiscoverer(&fileDiscoverer{path: configPath})

	listener := &capabilityListener{ch: make(chan []*Capability, 1)}
	r.AddListener(listener)

	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })

	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	r.Refresh()
	waitForRefresh(t, listener.ch)

	if r.Count() != 1 {
		t.Errorf("expected 1 capability initially, got %d", r.Count())
	}

	if err := os.WriteFile(configPath, []byte("cap1 cap2"), 0644); err != nil {
		t.Fatalf("failed to write modified config: %v", err)
	}
	setModTime(t, configPath, baseTime.Add(1*time.Minute))

	manual.Tick()
	waitForRefresh(t, listener.ch)

	if r.Count() != 2 {
		t.Errorf("expected 2 capabilities after file modification, got %d", r.Count())
	}
}

func TestWatcherDetectsFileDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "capabilities.txt")

	if err := os.WriteFile(configPath, []byte("cap1"), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}
	baseTime := time.Unix(30, 0)
	setModTime(t, configPath, baseTime)

	r := NewRegistry()
	r.RegisterDiscoverer(&fileDiscoverer{path: configPath})

	listener := &capabilityListener{ch: make(chan []*Capability, 1)}
	r.AddListener(listener)

	w := NewWatcher(r)
	manual := newManualTicker()
	w.SetTickerFactory(func(time.Duration) ticker { return manual })

	if err := w.Start(); err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}
	defer w.Stop()

	r.Refresh()
	waitForRefresh(t, listener.ch)

	if r.Count() != 1 {
		t.Errorf("expected 1 capability initially, got %d", r.Count())
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatalf("failed to delete config: %v", err)
	}

	manual.Tick()
	waitForRefresh(t, listener.ch)

	if r.Count() != 0 {
		t.Errorf("expected 0 capabilities after file deletion, got %d", r.Count())
	}
}
