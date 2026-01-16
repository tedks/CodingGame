package unit

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	u := New("test_login", "Login Test", 100.0, 200.0)

	if u.ID() != "test_login" {
		t.Errorf("ID() = %q, want test_login", u.ID())
	}
	if u.Name() != "Login Test" {
		t.Errorf("Name() = %q, want Login Test", u.Name())
	}

	x, y := u.Position()
	if x != 100.0 || y != 200.0 {
		t.Errorf("Position() = (%v, %v), want (100, 200)", x, y)
	}

	if u.State() != UnitStateIdle {
		t.Errorf("initial State() = %q, want %q", u.State(), UnitStateIdle)
	}
}

func TestNew_EmptyID(t *testing.T) {
	u := New("", "Fallback Name", 0, 0)

	if u.ID() != "Fallback Name" {
		t.Errorf("ID() = %q, want Fallback Name (should use name when id is empty)", u.ID())
	}
}

func TestNew_EmptyName(t *testing.T) {
	u := New("fallback-id", "", 0, 0)

	if u.Name() != "fallback-id" {
		t.Errorf("Name() = %q, want fallback-id (should use id when name is empty)", u.Name())
	}
}

func TestUnitStartTest(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Initially idle
	if u.State() != UnitStateIdle {
		t.Errorf("initial State() = %q, want %q", u.State(), UnitStateIdle)
	}

	// Start test
	u.StartTest()

	if u.State() != UnitStateRunning {
		t.Errorf("after StartTest(), State() = %q, want %q", u.State(), UnitStateRunning)
	}
}

func TestUnitRecordTest_Passed(t *testing.T) {
	u := New("test", "test", 0, 0)

	u.StartTest()

	// Record passing test
	result := &TestResult{
		Passed:    true,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	}

	u.RecordTest(result)

	// Check state updated
	if u.State() != UnitStatePassed {
		t.Errorf("after passing test, State() = %q, want %q", u.State(), UnitStatePassed)
	}

	// Check last result stored
	lastResult := u.LastTestResult()
	if lastResult == nil {
		t.Fatal("LastTestResult() returned nil")
	}
	if !lastResult.Passed {
		t.Error("LastTestResult().Passed = false, want true")
	}

	// Check metrics updated
	metrics := u.Metrics()
	if metrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", metrics.TotalRuns)
	}
	if metrics.PassCount != 1 {
		t.Errorf("PassCount = %d, want 1", metrics.PassCount)
	}
	if metrics.FailCount != 0 {
		t.Errorf("FailCount = %d, want 0", metrics.FailCount)
	}
	if metrics.PassRate != 100.0 {
		t.Errorf("PassRate = %v, want 100.0", metrics.PassRate)
	}
}

func TestUnitRecordTest_Failed(t *testing.T) {
	u := New("test", "test", 0, 0)

	u.StartTest()

	// Record failing test
	result := &TestResult{
		Passed:       false,
		Duration:     50 * time.Millisecond,
		Output:       "assertion failed: expected 5, got 3",
		ErrorMessage: "test failed",
		Timestamp:    time.Now(),
	}

	u.RecordTest(result)

	// Check state updated
	if u.State() != UnitStateFailed {
		t.Errorf("after failing test, State() = %q, want %q", u.State(), UnitStateFailed)
	}

	// Check metrics updated
	metrics := u.Metrics()
	if metrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", metrics.TotalRuns)
	}
	if metrics.PassCount != 0 {
		t.Errorf("PassCount = %d, want 0", metrics.PassCount)
	}
	if metrics.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", metrics.FailCount)
	}
	if metrics.PassRate != 0.0 {
		t.Errorf("PassRate = %v, want 0.0", metrics.PassRate)
	}
}

func TestUnitRecordTest_NilResult(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Recording nil should be safe (no-op)
	u.RecordTest(nil)

	// State should remain idle
	if u.State() != UnitStateIdle {
		t.Errorf("after RecordTest(nil), State() = %q, want %q", u.State(), UnitStateIdle)
	}

	// Metrics should be empty
	metrics := u.Metrics()
	if metrics.TotalRuns != 0 {
		t.Errorf("TotalRuns = %d, want 0", metrics.TotalRuns)
	}
}

func TestUnitMetrics_MultipleRuns(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Record multiple test runs
	runs := []struct {
		duration time.Duration
		passed   bool
	}{
		{100 * time.Millisecond, true},
		{150 * time.Millisecond, true},
		{200 * time.Millisecond, false},
		{120 * time.Millisecond, true},
		{180 * time.Millisecond, true},
	}

	for _, run := range runs {
		result := &TestResult{
			Passed:    run.passed,
			Duration:  run.duration,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()

	// Check counts
	if metrics.TotalRuns != 5 {
		t.Errorf("TotalRuns = %d, want 5", metrics.TotalRuns)
	}
	if metrics.PassCount != 4 {
		t.Errorf("PassCount = %d, want 4", metrics.PassCount)
	}
	if metrics.FailCount != 1 {
		t.Errorf("FailCount = %d, want 1", metrics.FailCount)
	}

	// Check pass rate
	expectedPassRate := 80.0 // 4/5 = 80%
	if metrics.PassRate != expectedPassRate {
		t.Errorf("PassRate = %v, want %v", metrics.PassRate, expectedPassRate)
	}

	// Check duration stats
	// Durations: 100, 150, 200, 120, 180
	expectedAvg := (100 + 150 + 200 + 120 + 180) * time.Millisecond / 5 // 750/5 = 150ms
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

func TestUnitSetPosition(t *testing.T) {
	u := New("test", "test", 100.0, 200.0)

	x, y := u.Position()
	if x != 100.0 || y != 200.0 {
		t.Errorf("initial Position() = (%v, %v), want (100, 200)", x, y)
	}

	// Update position
	u.SetPosition(300.0, 400.0)

	x, y = u.Position()
	if x != 300.0 || y != 400.0 {
		t.Errorf("after SetPosition, Position() = (%v, %v), want (300, 400)", x, y)
	}
}

func TestUnitSetCoverage(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Initially 0
	metrics := u.Metrics()
	if metrics.CoveragePercent != 0 {
		t.Errorf("initial CoveragePercent = %v, want 0", metrics.CoveragePercent)
	}

	// Set coverage
	u.SetCoverage(75.5)

	metrics = u.Metrics()
	if metrics.CoveragePercent != 75.5 {
		t.Errorf("after SetCoverage, CoveragePercent = %v, want 75.5", metrics.CoveragePercent)
	}
}

func TestUnitRunHistory(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Initially empty
	history := u.RunHistory()
	if len(history) != 0 {
		t.Errorf("initial RunHistory length = %d, want 0", len(history))
	}

	// Record some tests
	for i := 0; i < 3; i++ {
		result := &TestResult{
			Passed:    true,
			Duration:  time.Duration(i+1) * 100 * time.Millisecond,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	history = u.RunHistory()
	if len(history) != 3 {
		t.Errorf("RunHistory length = %d, want 3", len(history))
	}

	// Verify history is a copy (mutation doesn't affect original)
	history[0].Passed = false
	historyAgain := u.RunHistory()
	if !historyAgain[0].Passed {
		t.Error("RunHistory returned a reference instead of a copy")
	}
}

func TestUnitCalculateFlakiness_NotEnoughData(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Record only 1 test (need at least 2 for flakiness)
	result := &TestResult{
		Passed:    true,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
	}
	u.RecordTest(result)

	metrics := u.Metrics()
	if metrics.FlakinessScore != 0 {
		t.Errorf("with < 2 runs, FlakinessScore = %v, want 0", metrics.FlakinessScore)
	}
}

func TestUnitCalculateFlakiness_Stable(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Record 10 passing tests (perfectly stable)
	for i := 0; i < 10; i++ {
		result := &TestResult{
			Passed:    true,
			Duration:  100 * time.Millisecond,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()
	if metrics.FlakinessScore != 0 {
		t.Errorf("with all passes, FlakinessScore = %v, want 0", metrics.FlakinessScore)
	}
}

func TestUnitCalculateFlakiness_Flaky(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Record alternating pass/fail (maximum flakiness)
	for i := 0; i < 10; i++ {
		result := &TestResult{
			Passed:    i%2 == 0, // Alternate between pass and fail
			Duration:  100 * time.Millisecond,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()
	// With alternating results, we have 9 transitions out of 9 possible
	// That's 100% flakiness
	if metrics.FlakinessScore != 100.0 {
		t.Errorf("with alternating results, FlakinessScore = %v, want 100.0", metrics.FlakinessScore)
	}
}

func TestUnitCalculateFlakiness_ModeratelyFlaky(t *testing.T) {
	u := New("test", "test", 0, 0)

	// Record: pass, pass, fail, pass, pass (2 transitions out of 4)
	results := []bool{true, true, false, true, true}
	for _, passed := range results {
		result := &TestResult{
			Passed:    passed,
			Duration:  100 * time.Millisecond,
			Timestamp: time.Now(),
		}
		u.RecordTest(result)
	}

	metrics := u.Metrics()
	// 2 transitions / 4 possible = 50% flakiness
	if metrics.FlakinessScore != 50.0 {
		t.Errorf("with 2/4 transitions, FlakinessScore = %v, want 50.0", metrics.FlakinessScore)
	}
}

func TestUnitState_Constants(t *testing.T) {
	// Verify UnitState constants are distinct
	states := map[UnitState]bool{
		UnitStateIdle:    true,
		UnitStateRunning: true,
		UnitStatePassed:  true,
		UnitStateFailed:  true,
	}

	if len(states) != 4 {
		t.Error("UnitState constants should be distinct")
	}
}

func TestUnitConcurrency(t *testing.T) {
	// Test that concurrent access doesn't cause data races
	u := New("test", "test", 0, 0)

	done := make(chan bool)

	// Start multiple goroutines accessing unit concurrently
	for i := 0; i < 10; i++ {
		go func(passed bool) {
			result := &TestResult{
				Passed:    passed,
				Duration:  100 * time.Millisecond,
				Timestamp: time.Now(),
			}
			u.RecordTest(result)
			_ = u.Metrics()
			_ = u.State()
			_ = u.RunHistory()
			done <- true
		}(i%2 == 0)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without data race, test passes
	// (run with -race flag to detect races)
}

func TestTestMetrics_ZeroValues(t *testing.T) {
	metrics := TestMetrics{}

	// Verify zero values are sensible
	if metrics.TotalRuns != 0 {
		t.Errorf("TotalRuns zero value = %d, want 0", metrics.TotalRuns)
	}
	if metrics.PassRate != 0.0 {
		t.Errorf("PassRate zero value = %v, want 0.0", metrics.PassRate)
	}
	if metrics.FlakinessScore != 0.0 {
		t.Errorf("FlakinessScore zero value = %v, want 0.0", metrics.FlakinessScore)
	}
	if metrics.CoveragePercent != 0.0 {
		t.Errorf("CoveragePercent zero value = %v, want 0.0", metrics.CoveragePercent)
	}
}

func TestTestResult_Fields(t *testing.T) {
	now := time.Now()
	result := TestResult{
		Passed:       false,
		Duration:     5 * time.Second,
		Output:       "test output",
		ErrorMessage: "error message",
		Timestamp:    now,
	}

	if result.Passed {
		t.Error("Passed should be false")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", result.Duration)
	}
	if result.Output != "test output" {
		t.Errorf("Output = %q, want \"test output\"", result.Output)
	}
	if result.ErrorMessage != "error message" {
		t.Errorf("ErrorMessage = %q, want \"error message\"", result.ErrorMessage)
	}
	if !result.Timestamp.Equal(now) {
		t.Error("Timestamp not set correctly")
	}
}
