package capability

import (
	"os"
	"sync"
	"time"
)

// Watcher monitors configuration files for changes and triggers registry refresh.
//
// Design note: Uses polling instead of inotify/fsnotify for simplicity and portability.
// Config files change infrequently, so the 5-second default interval is appropriate.
// Benefits: No file handle exhaustion, works on all platforms, simple implementation.
// Trade-off: Small delay (configurable) between file change and detection.
type Watcher struct {
	mu sync.Mutex

	registry     *Registry
	pollInterval time.Duration
	running      bool
	stopCh       chan struct{}
	wg           sync.WaitGroup
	newTicker    func(time.Duration) ticker

	// File modification times for change detection
	fileTimesMu sync.Mutex
	fileTimes   map[string]time.Time
}

type ticker interface {
	Channel() <-chan time.Time
	Stop()
}

type realTicker struct {
	*time.Ticker
}

func (t realTicker) Channel() <-chan time.Time {
	return t.Ticker.C
}

func newRealTicker(interval time.Duration) ticker {
	return realTicker{Ticker: time.NewTicker(interval)}
}

// NewWatcher creates a new watcher for the given registry.
func NewWatcher(registry *Registry) *Watcher {
	return &Watcher{
		registry:     registry,
		pollInterval: 5 * time.Second,
		fileTimes:    make(map[string]time.Time),
		newTicker:    newRealTicker,
	}
}

// SetTickerFactory configures the ticker creation function (for testing).
// Changes take effect on the next Start() call; do not call while running.
func (w *Watcher) SetTickerFactory(factory func(time.Duration) ticker) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		return
	}
	w.newTicker = factory
}

// SetPollInterval configures the polling interval.
// Note: Changes to the interval only take effect on the next Start() call.
// Calling SetPollInterval while the watcher is running will not affect the current
// polling rate; you must Stop() and Start() again for the new interval to apply.
func (w *Watcher) SetPollInterval(interval time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pollInterval = interval
}

// Start begins watching for file changes.
func (w *Watcher) Start() error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil // Already running
	}
	if w.newTicker == nil {
		w.newTicker = newRealTicker
	}
	interval := w.pollInterval
	newTicker := w.newTicker
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	// Initialize file times
	w.updateFileTimes()

	// Start polling goroutine
	w.wg.Add(1)
	go w.poll(interval, newTicker)

	return nil
}

// Stop stops watching for file changes.
func (w *Watcher) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil // Not running
	}
	w.running = false
	close(w.stopCh)
	w.mu.Unlock()

	w.wg.Wait()
	return nil
}

// IsRunning returns whether the watcher is currently running.
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// poll is the main polling loop.
func (w *Watcher) poll(interval time.Duration, newTicker func(time.Duration) ticker) {
	defer w.wg.Done()

	ticker := newTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.Channel():
			if w.checkForChanges() {
				w.registry.Refresh()
			}
		}
	}
}

// updateFileTimes records the current modification times of all watch paths.
func (w *Watcher) updateFileTimes() {
	paths := w.registry.WatchPaths()
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			// File doesn't exist or can't be accessed
			w.fileTimesMu.Lock()
			delete(w.fileTimes, path)
			w.fileTimesMu.Unlock()
			continue
		}
		modTime := info.ModTime()
		w.fileTimesMu.Lock()
		w.fileTimes[path] = modTime
		w.fileTimesMu.Unlock()
	}
}

// checkForChanges checks if any watched files have changed.
// Returns true if changes were detected.
func (w *Watcher) checkForChanges() bool {
	paths := w.registry.WatchPaths()
	changed := false

	// Build set of current watch paths for efficient lookup
	currentPaths := make(map[string]bool, len(paths))
	for _, p := range paths {
		currentPaths[p] = true
	}

	// Clean up paths no longer being watched (prevents memory leak)
	w.fileTimesMu.Lock()
	for path := range w.fileTimes {
		if !currentPaths[path] {
			delete(w.fileTimes, path)
		}
	}
	w.fileTimesMu.Unlock()

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			// File doesn't exist now
			w.fileTimesMu.Lock()
			if _, existed := w.fileTimes[path]; existed {
				// File was deleted
				delete(w.fileTimes, path)
				changed = true
			}
			w.fileTimesMu.Unlock()
			continue
		}

		modTime := info.ModTime()
		w.fileTimesMu.Lock()
		if prevTime, exists := w.fileTimes[path]; exists {
			if !modTime.Equal(prevTime) {
				// File was modified
				w.fileTimes[path] = modTime
				changed = true
			}
		} else {
			// File is new
			w.fileTimes[path] = modTime
			changed = true
		}
		w.fileTimesMu.Unlock()
	}

	return changed
}

// ForceRefresh triggers an immediate refresh of the registry.
func (w *Watcher) ForceRefresh() {
	w.updateFileTimes()
	w.registry.Refresh()
}
