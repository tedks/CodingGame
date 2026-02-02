package capability

import (
	"sync"
	"testing"
	"time"
)

// TestRegistry_ConcurrentReadWrite tests that concurrent reads and writes to the
// registry don't race. Run with -race flag to detect data races.
func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	r := NewRegistry()

	// Pre-populate with some capabilities
	mock := &mockDiscoverer{
		name: "concurrent-test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
			NewCapability("cap2", "Cap 2", TypeMCP, DomainBuild),
		},
	}
	r.RegisterDiscoverer(mock)
	r.Refresh()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 4) // 4 operation types

	// Concurrent reads via GetAll
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.GetAll()
		}()
	}

	// Concurrent reads via Get
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.Get("cap1")
		}()
	}

	// Concurrent reads via Count
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.Count()
		}()
	}

	// Concurrent reads via GetByDomain
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.GetByDomain(DomainCore)
		}()
	}

	wg.Wait()
}

// TestRegistry_ConcurrentRefreshAndGet tests that Refresh and Get operations
// don't race when executed concurrently.
func TestRegistry_ConcurrentRefreshAndGet(t *testing.T) {
	r := NewRegistry()

	mock := &mockDiscoverer{
		name: "concurrent-refresh",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	}
	r.RegisterDiscoverer(mock)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent refreshes
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.Refresh()
		}()
	}

	// Concurrent reads during refresh
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.GetAll()
			_ = r.Get("cap1")
			_ = r.Count()
		}()
	}

	wg.Wait()

	// Verify registry is in valid state after concurrent operations
	if r.Count() != 1 {
		t.Errorf("expected 1 capability after concurrent operations, got %d", r.Count())
	}
}

// TestRegistry_ConcurrentListenerModification tests that adding and removing
// listeners concurrently with refresh doesn't cause races.
func TestRegistry_ConcurrentListenerModification(t *testing.T) {
	r := NewRegistry()

	mock := &mockDiscoverer{
		name: "listener-test",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	}
	r.RegisterDiscoverer(mock)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Create a pool of listeners to add/remove
	listeners := make([]*testListener, goroutines)
	for i := 0; i < goroutines; i++ {
		listeners[i] = &testListener{}
	}

	// Concurrent listener additions
	for i := 0; i < goroutines; i++ {
		listener := listeners[i]
		go func() {
			defer wg.Done()
			r.AddListener(listener)
		}()
	}

	// Concurrent refreshes (which notify listeners)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.Refresh()
		}()
	}

	// Concurrent listener removals
	for i := 0; i < goroutines; i++ {
		listener := listeners[i]
		go func() {
			defer wg.Done()
			r.RemoveListener(listener)
		}()
	}

	wg.Wait()
}

// TestRegistry_ConcurrentDiscovererRegistration tests that registering discoverers
// concurrently with refresh doesn't race.
func TestRegistry_ConcurrentDiscovererRegistration(t *testing.T) {
	r := NewRegistry()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent discoverer registrations
	for i := 0; i < goroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			mock := &mockDiscoverer{
				name: "concurrent-" + string(rune('a'+idx%26)),
				capabilities: []*Capability{
					NewCapability("cap-"+string(rune('a'+idx%26)), "Cap", TypeTool, DomainCore),
				},
			}
			r.RegisterDiscoverer(mock)
		}()
	}

	// Concurrent refreshes
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.Refresh()
		}()
	}

	wg.Wait()

	// Final refresh to get all capabilities
	r.Refresh()

	// Should have capabilities from at least some discoverers
	if r.Count() == 0 {
		t.Error("expected some capabilities after concurrent registration")
	}
}

// TestWatcher_ConcurrentStartStop tests that starting and stopping the watcher
// concurrently doesn't cause races or panics.
func TestWatcher_ConcurrentStartStop(t *testing.T) {
	r := NewRegistry()
	w := NewWatcher(r)

	// Use a very fast ticker to stress the system
	w.SetPollInterval(1 * time.Millisecond)
	w.SetTickerFactory(func(d time.Duration) ticker {
		return newManualTicker()
	})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent starts
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = w.Start()
		}()
	}

	// Concurrent stops
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = w.Stop()
		}()
	}

	wg.Wait()

	// Ensure we can cleanly stop (no matter what state we're in)
	w.Stop()

	// And start again
	if err := w.Start(); err != nil {
		t.Errorf("Start() after concurrent operations returned error: %v", err)
	}
	if err := w.Stop(); err != nil {
		t.Errorf("Stop() after concurrent operations returned error: %v", err)
	}
}

// TestWatcher_ConcurrentSnapshotAndStateChange tests that taking snapshots
// while state changes doesn't race.
func TestWatcher_ConcurrentSnapshotAndStateChange(t *testing.T) {
	r := NewRegistry()
	w := NewWatcher(r)
	w.SetTickerFactory(func(d time.Duration) ticker {
		return newManualTicker()
	})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent snapshots
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = w.Snapshot()
		}()
	}

	// Concurrent IsRunning checks
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = w.IsRunning()
		}()
	}

	// Concurrent start/stop
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				_ = w.Start()
			} else {
				_ = w.Stop()
			}
		}()
	}

	wg.Wait()
	w.Stop()
}

// TestRegistry_ConcurrentWatchPaths tests that WatchPaths can be called
// concurrently with discoverer registration.
func TestRegistry_ConcurrentWatchPaths(t *testing.T) {
	r := NewRegistry()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent discoverer registrations with watch paths
	for i := 0; i < goroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			mock := &mockDiscoverer{
				name:       "path-test-" + string(rune('a'+idx%26)),
				watchPaths: []string{"/path/" + string(rune('a'+idx%26))},
			}
			r.RegisterDiscoverer(mock)
		}()
	}

	// Concurrent WatchPaths reads
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.WatchPaths()
		}()
	}

	wg.Wait()
}

// TestRegistry_ConcurrentLastErrors tests that LastErrors and HasErrors
// can be called concurrently with Refresh.
func TestRegistry_ConcurrentLastErrors(t *testing.T) {
	r := NewRegistry()

	// Add mix of working and failing discoverers
	r.RegisterDiscoverer(&mockDiscoverer{
		name: "working",
		capabilities: []*Capability{
			NewCapability("cap1", "Cap 1", TypeTool, DomainCore),
		},
	})
	r.RegisterDiscoverer(&errorDiscoverer{
		name: "failing",
		err:  errTestDiscovery,
	})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent refreshes
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.Refresh()
		}()
	}

	// Concurrent LastErrors reads
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.LastErrors()
		}()
	}

	// Concurrent HasErrors reads
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = r.HasErrors()
		}()
	}

	wg.Wait()
}

// errTestDiscovery is a sentinel error for testing.
var errTestDiscovery = &testError{msg: "test discovery error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
