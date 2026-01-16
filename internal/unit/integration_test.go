package unit

import (
	"sync"
	"testing"
	"time"
)

// TestIntegration_UnitLifecycle tests a complete unit lifecycle
func TestIntegration_UnitLifecycle(t *testing.T) {
	u := New("test_auth", "Authentication Tests", 100.0, 200.0)

	// 1. Initial state should be idle
	if u.State() != UnitStateIdle {
		t.Errorf("Initial state = %v, want %v", u.State(), UnitStateIdle)
	}

	// 2. Start test
	u.StartTest()
	if u.State() != UnitStateRunning {
		t.Errorf("After StartTest, state = %v, want %v", u.State(), UnitStateRunning)
	}

	// 3. Simulate passing test
	result := &TestResult{
		Passed:    true,
		Duration:  150 * time.Millisecond,
		Timestamp: time.Now(),
	}
	u.RecordTest(result)

	// 4. State should be passed
	if u.State() != UnitStatePassed {
		t.Errorf("After passing test, state = %v, want %v", u.State(), UnitStatePassed)
	}

	// 5. Metrics should be updated
	metrics := u.Metrics()
	if metrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", metrics.TotalRuns)
	}
	if metrics.PassCount != 1 {
		t.Errorf("PassCount = %d, want 1", metrics.PassCount)
	}

	// 6. Start another test and fail it
	u.StartTest()
	failResult := &TestResult{
		Passed:       false,
		Duration:     100 * time.Millisecond,
		Output:       "AssertionError: expected true, got false",
		ErrorMessage: "test failed",
		Timestamp:    time.Now(),
	}
	u.RecordTest(failResult)

	// 7. State should be failed
	if u.State() != UnitStateFailed {
		t.Errorf("After failed test, state = %v, want %v", u.State(), UnitStateFailed)
	}

	// 8. Metrics should reflect both runs
	metrics = u.Metrics()
	if metrics.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", metrics.TotalRuns)
	}
	if metrics.PassRate != 50.0 {
		t.Errorf("PassRate = %v, want 50.0", metrics.PassRate)
	}
}

// TestIntegration_MultipleUnits tests managing multiple units
func TestIntegration_MultipleUnits(t *testing.T) {
	units := make([]*Unit, 20)

	// Create 20 units
	for i := 0; i < 20; i++ {
		id := string(rune('a' + i))
		units[i] = New(id, "Test "+id, float64(i*50), float64(i*50))
	}

	// Run all tests concurrently
	var wg sync.WaitGroup
	for i, unit := range units {
		wg.Add(1)
		go func(idx int, u *Unit) {
			defer wg.Done()

			u.StartTest()
			time.Sleep(time.Millisecond) // Simulate test execution

			result := &TestResult{
				Passed:    idx%3 != 0, // Fail every 3rd test
				Duration:  time.Duration(idx+1) * 10 * time.Millisecond,
				Timestamp: time.Now(),
			}
			u.RecordTest(result)
		}(i, unit)
	}

	wg.Wait()

	// Verify all units were updated
	passCount := 0
	failCount := 0
	for _, u := range units {
		state := u.State()
		if state == UnitStatePassed {
			passCount++
		} else if state == UnitStateFailed {
			failCount++
		}

		metrics := u.Metrics()
		if metrics.TotalRuns != 1 {
			t.Errorf("Unit %s has TotalRuns = %d, want 1", u.ID(), metrics.TotalRuns)
		}
	}

	// 13 passes, 7 failures (every 3rd fails: 0,3,6,9,12,15,18)
	expectedPasses := 13
	expectedFails := 7
	if passCount != expectedPasses || failCount != expectedFails {
		t.Errorf("Expected %d passes and %d fails, got %d passes and %d fails",
			expectedPasses, expectedFails, passCount, failCount)
	}
}

// TestIntegration_UnitFlakinessDetection tests flakiness detection over time
func TestIntegration_UnitFlakinessDetection(t *testing.T) {
	u := New("flaky_test", "Flaky Network Test", 0, 0)

	// Simulate a very flaky test (alternating pass/fail)
	for i := 0; i < 20; i++ {
		result := &TestResult{
			Passed:    i%2 == 0, // Alternate pass/fail
			Duration:  100 * time.Millisecond,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()

	// Should have maximum flakiness (100)
	if metrics.FlakinessScore != 100.0 {
		t.Errorf("FlakinessScore = %v, want 100.0 for alternating results", metrics.FlakinessScore)
	}

	// Now run 20 more passing tests to stabilize
	for i := 0; i < 20; i++ {
		result := &TestResult{
			Passed:    true,
			Duration:  100 * time.Millisecond,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics = u.Metrics()

	// Flakiness should decrease (recent 20 runs are all passes)
	if metrics.FlakinessScore != 0.0 {
		t.Errorf("FlakinessScore = %v, want 0.0 after stabilization", metrics.FlakinessScore)
	}
}

// TestIntegration_UnitDurationMetrics tests duration tracking
func TestIntegration_UnitDurationMetrics(t *testing.T) {
	u := New("perf_test", "Performance Test", 0, 0)

	// Simulate tests with varying durations
	durations := []time.Duration{
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
		120 * time.Millisecond,
		180 * time.Millisecond,
	}

	for _, duration := range durations {
		result := &TestResult{
			Passed:    true,
			Duration:  duration,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()

	// Calculate expected average
	totalMs := 100 + 150 + 200 + 120 + 180                     // 750ms
	expectedAvg := time.Duration(totalMs/5) * time.Millisecond // 150ms

	if metrics.AvgDuration != expectedAvg {
		t.Errorf("AvgDuration = %v, want %v", metrics.AvgDuration, expectedAvg)
	}

	if metrics.MinDuration != 100*time.Millisecond {
		t.Errorf("MinDuration = %v, want 100ms", metrics.MinDuration)
	}

	if metrics.MaxDuration != 200*time.Millisecond {
		t.Errorf("MaxDuration = %v, want 200ms", metrics.MaxDuration)
	}

	if metrics.LastDuration != 180*time.Millisecond {
		t.Errorf("LastDuration = %v, want 180ms", metrics.LastDuration)
	}
}

// TestIntegration_UnitCoverageTracking tests coverage integration
func TestIntegration_UnitCoverageTracking(t *testing.T) {
	u := New("coverage_test", "Coverage Test", 0, 0)

	// Initially no coverage
	metrics := u.Metrics()
	if metrics.CoveragePercent != 0 {
		t.Errorf("Initial coverage = %v, want 0", metrics.CoveragePercent)
	}

	// Run test and set coverage
	result := &TestResult{
		Passed:    true,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	}
	u.RecordTest(result)
	u.SetCoverage(85.5)

	// Verify coverage is tracked
	metrics = u.Metrics()
	if metrics.CoveragePercent != 85.5 {
		t.Errorf("CoveragePercent = %v, want 85.5", metrics.CoveragePercent)
	}

	// Update coverage
	u.SetCoverage(92.3)
	metrics = u.Metrics()
	if metrics.CoveragePercent != 92.3 {
		t.Errorf("CoveragePercent = %v, want 92.3", metrics.CoveragePercent)
	}
}

// TestIntegration_UnitPositionTracking tests position updates
func TestIntegration_UnitPositionTracking(t *testing.T) {
	u := New("ui_test", "UI Component Test", 0, 0)

	// Move the unit around the map
	positions := [][2]float64{
		{150, 150},
		{250, 200},
		{350, 300},
		{175, 275},
		{75, 75},
	}

	for _, pos := range positions {
		u.SetPosition(pos[0], pos[1])
		x, y := u.Position()
		if x != pos[0] || y != pos[1] {
			t.Errorf("After SetPosition(%v, %v), Position() = (%v, %v)", pos[0], pos[1], x, y)
		}
	}
}

// TestIntegration_UnitHistoryManagement tests run history
func TestIntegration_UnitHistoryManagement(t *testing.T) {
	u := New("integration_test", "Integration Test Suite", 0, 0)

	// Record many test runs
	const numRuns = 50
	for i := 0; i < numRuns; i++ {
		result := &TestResult{
			Passed:    i%4 != 0, // Fail every 4th run
			Duration:  time.Duration(i+1) * 10 * time.Millisecond,
			Output:    "test output",
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	// Verify history length
	history := u.RunHistory()
	if len(history) != numRuns {
		t.Errorf("RunHistory length = %d, want %d", len(history), numRuns)
	}

	// Verify history is ordered
	for i := 1; i < len(history); i++ {
		if history[i].Timestamp.Before(history[i-1].Timestamp) {
			t.Error("Run history is not ordered chronologically")
			break
		}
	}

	// Verify metrics
	metrics := u.Metrics()
	if metrics.TotalRuns != numRuns {
		t.Errorf("TotalRuns = %d, want %d", metrics.TotalRuns, numRuns)
	}

	// Expected: 37 passes (50 - 13 failures where i%4==0)
	// Count of values where i%4==0 for i in [0, 49]: 0, 4, 8, 12, 16, 20, 24, 28, 32, 36, 40, 44, 48 = 13 values
	expectedPasses := 37
	if metrics.PassCount != expectedPasses {
		t.Errorf("PassCount = %d, want %d", metrics.PassCount, expectedPasses)
	}
}

// TestIntegration_UnitStateTransitions tests all state transitions
func TestIntegration_UnitStateTransitions(t *testing.T) {
	u := New("state_test", "State Transition Test", 0, 0)

	transitions := []struct {
		action        func()
		expectedState UnitState
	}{
		{
			action:        func() {}, // Initial state
			expectedState: UnitStateIdle,
		},
		{
			action:        func() { u.StartTest() },
			expectedState: UnitStateRunning,
		},
		{
			action: func() {
				u.RecordTest(&TestResult{
					Passed:    true,
					Duration:  100 * time.Millisecond,
					Timestamp: time.Now(),
				})
			},
			expectedState: UnitStatePassed,
		},
		{
			action:        func() { u.StartTest() },
			expectedState: UnitStateRunning,
		},
		{
			action: func() {
				u.RecordTest(&TestResult{
					Passed:       false,
					Duration:     50 * time.Millisecond,
					ErrorMessage: "test failed",
					Timestamp:    time.Now(),
				})
			},
			expectedState: UnitStateFailed,
		},
		{
			action:        func() { u.StartTest() },
			expectedState: UnitStateRunning,
		},
		{
			action: func() {
				u.RecordTest(&TestResult{
					Passed:    true,
					Duration:  75 * time.Millisecond,
					Timestamp: time.Now(),
				})
			},
			expectedState: UnitStatePassed,
		},
	}

	for i, transition := range transitions {
		transition.action()
		state := u.State()
		if state != transition.expectedState {
			t.Errorf("Transition %d: state = %v, want %v", i, state, transition.expectedState)
		}
	}
}

// TestIntegration_UnitFlakinessScenarios tests various flakiness patterns
func TestIntegration_UnitFlakinessScenarios(t *testing.T) {
	scenarios := []struct {
		name          string
		pattern       func() []bool // Returns pass/fail pattern
		expectedScore float64
		scoreDesc     string
	}{
		{
			name: "perfectly stable (all pass)",
			pattern: func() []bool {
				result := make([]bool, 20)
				for i := range result {
					result[i] = true
				}
				return result
			},
			expectedScore: 0.0,
			scoreDesc:     "0% flaky",
		},
		{
			name: "perfectly stable (all fail)",
			pattern: func() []bool {
				result := make([]bool, 20)
				for i := range result {
					result[i] = false
				}
				return result
			},
			expectedScore: 0.0,
			scoreDesc:     "0% flaky",
		},
		{
			name: "maximum flakiness (alternating)",
			pattern: func() []bool {
				result := make([]bool, 20)
				for i := range result {
					result[i] = i%2 == 0
				}
				return result
			},
			expectedScore: 100.0,
			scoreDesc:     "100% flaky",
		},
		{
			name: "moderate flakiness (2 blocks)",
			pattern: func() []bool {
				result := make([]bool, 20)
				for i := range result {
					result[i] = i < 10 // First 10 pass, last 10 fail
				}
				return result
			},
			expectedScore: 5.263157894736842, // 1 transition / 19 possible * 100
			scoreDesc:     "~5% flaky (1 transition)",
		},
		{
			name: "random pattern (mostly pass with occasional fails)",
			pattern: func() []bool {
				// pass, pass, fail, pass, pass, pass, fail, pass, pass, pass
				// pass, pass, pass, pass, pass, fail, pass, pass, pass, pass
				// Transitions: 2->3, 3->4, 6->7, 7->8, 15->16, 16->17 = 6 transitions
				return []bool{
					true, true, false, true, true, true, false, true, true, true,
					true, true, true, true, true, false, true, true, true, true,
				}
			},
			expectedScore: 31.57894736842105, // 6 transitions / 19 possible * 100
			scoreDesc:     "~32% flaky (6 transitions)",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			u := New("flaky_"+scenario.name, scenario.name, 0, 0)

			pattern := scenario.pattern()
			for _, passed := range pattern {
				result := &TestResult{
					Passed:    passed,
					Duration:  100 * time.Millisecond,
					Timestamp: time.Now(),
				}
				u.RecordTest(result)
			}

			metrics := u.Metrics()
			if metrics.FlakinessScore != scenario.expectedScore {
				t.Errorf("%s: FlakinessScore = %v, want %v (%s)",
					scenario.name, metrics.FlakinessScore, scenario.expectedScore, scenario.scoreDesc)
			}
		})
	}
}

// TestIntegration_UnitLongRunningHistory tests behavior with extensive history
func TestIntegration_UnitLongRunningHistory(t *testing.T) {
	u := New("long_running", "Long Running Test", 0, 0)

	// Simulate 100 test runs
	const numRuns = 100
	for i := 0; i < numRuns; i++ {
		result := &TestResult{
			Passed:    i >= 20,                                 // First 20 fail, rest pass (improving over time)
			Duration:  time.Duration(100-i) * time.Millisecond, // Getting faster
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()

	// Verify total runs
	if metrics.TotalRuns != numRuns {
		t.Errorf("TotalRuns = %d, want %d", metrics.TotalRuns, numRuns)
	}

	// Verify pass rate (80 passes out of 100 = 80%)
	if metrics.PassRate != 80.0 {
		t.Errorf("PassRate = %v, want 80.0", metrics.PassRate)
	}

	// Flakiness should only look at last 20 runs (all passes)
	// So flakiness should be 0
	if metrics.FlakinessScore != 0.0 {
		t.Errorf("FlakinessScore = %v, want 0.0 (last 20 runs are all passes)", metrics.FlakinessScore)
	}

	// Duration metrics
	if metrics.MinDuration != 1*time.Millisecond {
		t.Errorf("MinDuration = %v, want 1ms", metrics.MinDuration)
	}
	if metrics.MaxDuration != 100*time.Millisecond {
		t.Errorf("MaxDuration = %v, want 100ms", metrics.MaxDuration)
	}
}

// TestIntegration_UnitConcurrentAccess tests thread safety with high concurrency
func TestIntegration_UnitConcurrentAccess(t *testing.T) {
	u := New("concurrent_test", "Concurrent Access Test", 0, 0)

	const numGoroutines = 100
	const runsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < runsPerGoroutine; j++ {
				result := &TestResult{
					Passed:    (id+j)%2 == 0,
					Duration:  time.Duration(id+j) * time.Millisecond,
					Timestamp: time.Now(),
				}
				u.RecordTest(result)

				// Also read metrics concurrently
				_ = u.Metrics()
				_ = u.State()
				_ = u.RunHistory()
			}
		}(i)
	}

	wg.Wait()

	// Verify all runs were recorded
	metrics := u.Metrics()
	expectedTotal := numGoroutines * runsPerGoroutine
	if metrics.TotalRuns != expectedTotal {
		t.Errorf("TotalRuns = %d, want %d", metrics.TotalRuns, expectedTotal)
	}

	// Verify pass + fail = total
	if metrics.PassCount+metrics.FailCount != metrics.TotalRuns {
		t.Error("PassCount + FailCount != TotalRuns")
	}
}
