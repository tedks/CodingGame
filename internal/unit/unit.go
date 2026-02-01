// Package unit provides visualization of tests as "units" that battle against bugs
// in a PvE (Player vs Environment) metaphor. Each unit represents a test that runs
// to validate code correctness.
//
// Units display real test metrics:
// - Execution time (actual wall-clock time to run)
// - Pass/fail status (real test results)
// - Flakiness score (variance in results over time)
// - Coverage (what code paths this test exercises)
//
// The unit metaphor makes test suites tangible and helps identify problematic tests
// that are slow, flaky, or provide insufficient coverage.
package unit

import (
	"sync"
	"time"
)

// Unit represents a test visualized as a combatant
type Unit struct {
	mu sync.RWMutex

	// Identity
	id   string // Unique identifier (test name/path)
	name string // Display name

	// Location (for map rendering)
	x float64
	y float64

	// State
	state   UnitState
	lastRun *TestResult

	// Metrics (historical)
	runHistory []TestRun
	metrics    TestMetrics
}

// UnitState represents the current state of a unit
type UnitState string

const (
	UnitStateIdle    UnitState = "idle"    // Not running
	UnitStateRunning UnitState = "running" // Test in progress
	UnitStatePassed  UnitState = "passed"  // Last run passed
	UnitStateFailed  UnitState = "failed"  // Last run failed
)

// TestResult represents the result of running a test
type TestResult struct {
	Passed       bool
	Duration     time.Duration
	Output       string // Test output (for failures)
	ErrorMessage string
	Timestamp    time.Time
}

// TestRun stores historical test execution information
type TestRun struct {
	Timestamp time.Time
	Duration  time.Duration
	Passed    bool
	Output    string
}

// TestMetrics contains aggregated test statistics
type TestMetrics struct {
	// Counts
	TotalRuns int
	PassCount int
	FailCount int

	// Timing statistics
	AvgDuration  time.Duration
	MinDuration  time.Duration
	MaxDuration  time.Duration
	LastDuration time.Duration

	// Reliability metrics
	PassRate       float64 // Percentage of runs that passed
	FlakinessScore float64 // 0-100, higher = more flaky

	// Coverage (if available from test runner)
	CoveragePercent float64
}

// New creates a new unit for a test
//
// Assumptions:
// - id and name are non-empty
// - x, y are valid map coordinates
//
// Edge cases:
// - id is empty -> use name as fallback
// - name is empty -> use id as fallback
func New(id, name string, x, y float64) *Unit {
	if id == "" {
		id = name
	}
	if name == "" {
		name = id
	}

	return &Unit{
		id:         id,
		name:       name,
		x:          x,
		y:          y,
		state:      UnitStateIdle,
		runHistory: make([]TestRun, 0),
		metrics: TestMetrics{
			MinDuration: time.Duration(1<<63 - 1), // Max int64
		},
	}
}

// ID returns the unit's unique identifier
func (u *Unit) ID() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.id
}

// Name returns the unit's display name
func (u *Unit) Name() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.name
}

// Position returns the unit's map coordinates
func (u *Unit) Position() (x, y float64) {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.x, u.y
}

// State returns the current unit state
func (u *Unit) State() UnitState {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.state
}

// Metrics returns the aggregated test metrics
func (u *Unit) Metrics() TestMetrics {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.metrics
}

// LastTestResult returns the most recent test result, or nil if never run.
// Returns a copy so callers cannot mutate internal state.
func (u *Unit) LastTestResult() *TestResult {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if u.lastRun == nil {
		return nil
	}
	resultCopy := *u.lastRun
	return &resultCopy
}

// StartTest marks the unit as currently running
func (u *Unit) StartTest() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.state = UnitStateRunning
}

// RecordTest records a completed test run and updates metrics
//
// Assumptions:
// - result is not nil
// - result.Duration is non-negative
//
// Edge cases:
// - Called when not running -> still record
// - result.Duration is 0 -> accept (very fast test)
// - Multiple rapid calls -> all recorded
func (u *Unit) RecordTest(result *TestResult) {
	if result == nil {
		return
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	resultCopy := *result

	// Update state based on result
	if resultCopy.Passed {
		u.state = UnitStatePassed
	} else {
		u.state = UnitStateFailed
	}

	// Store last result
	u.lastRun = &resultCopy

	// Add to history
	run := TestRun{
		Timestamp: resultCopy.Timestamp,
		Duration:  resultCopy.Duration,
		Passed:    resultCopy.Passed,
		Output:    resultCopy.Output,
	}
	u.runHistory = append(u.runHistory, run)

	// Update metrics
	u.updateMetrics()
}

// SetPosition updates the unit's map coordinates
func (u *Unit) SetPosition(x, y float64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.x = x
	u.y = y
}

// SetCoverage updates the code coverage percentage for this test
func (u *Unit) SetCoverage(percent float64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.metrics.CoveragePercent = percent
}

// updateMetrics recalculates aggregated metrics from run history
// Must be called with lock held
func (u *Unit) updateMetrics() {
	if len(u.runHistory) == 0 {
		return
	}

	var totalDuration time.Duration
	var passCount int

	minDuration := time.Duration(1<<63 - 1)
	var maxDuration time.Duration

	for _, run := range u.runHistory {
		// Duration stats
		totalDuration += run.Duration
		if run.Duration < minDuration {
			minDuration = run.Duration
		}
		if run.Duration > maxDuration {
			maxDuration = run.Duration
		}

		// Pass count
		if run.Passed {
			passCount++
		}
	}

	historyLen := len(u.runHistory)

	// Update metrics
	u.metrics.TotalRuns = historyLen
	u.metrics.PassCount = passCount
	u.metrics.FailCount = historyLen - passCount
	u.metrics.AvgDuration = totalDuration / time.Duration(historyLen)
	u.metrics.MinDuration = minDuration
	u.metrics.MaxDuration = maxDuration
	u.metrics.LastDuration = u.runHistory[historyLen-1].Duration

	// Pass rate
	u.metrics.PassRate = float64(passCount) / float64(historyLen) * 100.0

	// Calculate flakiness score
	u.metrics.FlakinessScore = u.calculateFlakiness()
}

// calculateFlakiness computes a flakiness score based on result variance
// Must be called with lock held
//
// Algorithm:
// - Count number of pass/fail transitions in recent history
// - More transitions = higher flakiness
// - Score is 0-100 (0 = stable, 100 = maximum flakiness)
//
// Assumptions:
// - runHistory has at least 2 entries
//
// Edge cases:
// - Less than 2 runs -> return 0 (not enough data)
// - All passes or all fails -> return 0 (perfectly stable)
// - Alternating pass/fail -> return 100 (maximum flakiness)
func (u *Unit) calculateFlakiness() float64 {
	if len(u.runHistory) < 2 {
		return 0
	}

	// Count state transitions in recent history (last 20 runs or all if less)
	historyWindow := 20
	startIdx := len(u.runHistory) - historyWindow
	if startIdx < 0 {
		startIdx = 0
	}

	recentHistory := u.runHistory[startIdx:]
	transitions := 0

	for i := 1; i < len(recentHistory); i++ {
		if recentHistory[i].Passed != recentHistory[i-1].Passed {
			transitions++
		}
	}

	// Maximum possible transitions = len(recentHistory) - 1
	maxTransitions := len(recentHistory) - 1
	if maxTransitions == 0 {
		return 0
	}

	// Normalize to 0-100 scale
	return float64(transitions) / float64(maxTransitions) * 100.0
}

// RunHistory returns a copy of the test run history
func (u *Unit) RunHistory() []TestRun {
	u.mu.RLock()
	defer u.mu.RUnlock()

	// Return a copy to prevent mutation
	history := make([]TestRun, len(u.runHistory))
	copy(history, u.runHistory)
	return history
}
