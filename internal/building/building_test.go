package building

import (
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/build"
)

func TestNew(t *testing.T) {
	target := build.Target{
		ID:   "//cmd/app:app",
		Name: "app",
		Type: build.TargetTypeBinary,
	}

	b := New(target, "bazel", 100.0, 200.0)

	if b.ID() != "//cmd/app:app" {
		t.Errorf("ID() = %q, want //cmd/app:app", b.ID())
	}
	if b.Name() != "app" {
		t.Errorf("Name() = %q, want app", b.Name())
	}
	if b.TargetType() != build.TargetTypeBinary {
		t.Errorf("TargetType() = %q, want %q", b.TargetType(), build.TargetTypeBinary)
	}
	if b.BuildSystem() != "bazel" {
		t.Errorf("BuildSystem() = %q, want bazel", b.BuildSystem())
	}

	x, y := b.Position()
	if x != 100.0 || y != 200.0 {
		t.Errorf("Position() = (%v, %v), want (100, 200)", x, y)
	}

	if b.State() != StateIdle {
		t.Errorf("initial State() = %q, want %q", b.State(), StateIdle)
	}
}

func TestNew_EmptyID(t *testing.T) {
	// Test that empty ID falls back to Name
	target := build.Target{
		ID:   "",
		Name: "fallback-name",
		Type: build.TargetTypeLibrary,
	}

	b := New(target, "npm", 0, 0)

	if b.ID() != "fallback-name" {
		t.Errorf("ID() = %q, want fallback-name (should use Name when ID is empty)", b.ID())
	}
}

func TestBuildingStartBuild(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Initially idle
	if b.State() != StateIdle {
		t.Errorf("initial State() = %q, want %q", b.State(), StateIdle)
	}

	// Start building
	b.StartBuild()

	if b.State() != StateBuilding {
		t.Errorf("after StartBuild(), State() = %q, want %q", b.State(), StateBuilding)
	}
}

func TestBuildingRecordBuild_Success(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	b.StartBuild()

	// Record successful build
	result := &build.Result{
		Success:      true,
		Duration:     5 * time.Second,
		CacheHits:    10,
		CacheMisses:  5,
		TargetsBuilt: 15,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(5 * time.Second),
	}

	b.RecordBuild(result)

	// Check state updated
	if b.State() != StateSuccess {
		t.Errorf("after successful build, State() = %q, want %q", b.State(), StateSuccess)
	}

	// Check last build stored
	lastBuild := b.LastBuildResult()
	if lastBuild == nil {
		t.Fatal("LastBuildResult() returned nil")
	}
	if !lastBuild.Success {
		t.Error("LastBuildResult().Success = false, want true")
	}

	// Check metrics updated
	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", metrics.SuccessCount)
	}
	if metrics.FailureCount != 0 {
		t.Errorf("FailureCount = %d, want 0", metrics.FailureCount)
	}
	if metrics.SuccessRate != 100.0 {
		t.Errorf("SuccessRate = %v, want 100.0", metrics.SuccessRate)
	}
}

func TestBuildingRecordBuild_Failure(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	b.StartBuild()

	// Record failed build
	result := &build.Result{
		Success:      false,
		Duration:     2 * time.Second,
		ErrorMessage: "compilation error",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(2 * time.Second),
	}

	b.RecordBuild(result)

	// Check state updated
	if b.State() != StateFailed {
		t.Errorf("after failed build, State() = %q, want %q", b.State(), StateFailed)
	}

	// Check metrics updated
	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
	if metrics.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", metrics.SuccessCount)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", metrics.FailureCount)
	}
	if metrics.SuccessRate != 0.0 {
		t.Errorf("SuccessRate = %v, want 0.0", metrics.SuccessRate)
	}
}

func TestBuildingRecordBuild_NilResult(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Recording nil should be safe (no-op)
	b.RecordBuild(nil)

	// State should remain idle
	if b.State() != StateIdle {
		t.Errorf("after RecordBuild(nil), State() = %q, want %q", b.State(), StateIdle)
	}

	// Metrics should be empty
	metrics := b.Metrics()
	if metrics.TotalBuilds != 0 {
		t.Errorf("TotalBuilds = %d, want 0", metrics.TotalBuilds)
	}
}

func TestMetrics_ZeroValues(t *testing.T) {
	metrics := Metrics{}

	if metrics.TotalBuilds != 0 {
		t.Errorf("TotalBuilds zero value = %d, want 0", metrics.TotalBuilds)
	}
	if metrics.SuccessRate != 0.0 {
		t.Errorf("SuccessRate zero value = %v, want 0.0", metrics.SuccessRate)
	}
	if metrics.AvgCacheHitRate != 0.0 {
		t.Errorf("AvgCacheHitRate zero value = %v, want 0.0", metrics.AvgCacheHitRate)
	}
	if metrics.MinDuration != 0 {
		t.Errorf("MinDuration zero value = %v, want 0", metrics.MinDuration)
	}
	if metrics.MaxDuration != 0 {
		t.Errorf("MaxDuration zero value = %v, want 0", metrics.MaxDuration)
	}
	if metrics.AvgDuration != 0 {
		t.Errorf("AvgDuration zero value = %v, want 0", metrics.AvgDuration)
	}
	if metrics.LastDuration != 0 {
		t.Errorf("LastDuration zero value = %v, want 0", metrics.LastDuration)
	}
}

func TestBuildingMetrics_MultipleBuilds(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Record multiple builds with different durations
	builds := []struct {
		duration    time.Duration
		success     bool
		cacheHits   int64
		cacheMisses int64
	}{
		{5 * time.Second, true, 10, 5},
		{3 * time.Second, true, 12, 3},
		{7 * time.Second, false, 0, 15},
		{4 * time.Second, true, 14, 1},
		{2 * time.Second, true, 13, 2},
	}

	for _, bld := range builds {
		result := &build.Result{
			Success:     bld.success,
			Duration:    bld.duration,
			CacheHits:   bld.cacheHits,
			CacheMisses: bld.cacheMisses,
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(bld.duration),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()

	// Check counts
	if metrics.TotalBuilds != 5 {
		t.Errorf("TotalBuilds = %d, want 5", metrics.TotalBuilds)
	}
	if metrics.SuccessCount != 4 {
		t.Errorf("SuccessCount = %d, want 4", metrics.SuccessCount)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", metrics.FailureCount)
	}

	// Check success rate
	expectedSuccessRate := 80.0 // 4/5 = 80%
	if metrics.SuccessRate != expectedSuccessRate {
		t.Errorf("SuccessRate = %v, want %v", metrics.SuccessRate, expectedSuccessRate)
	}

	// Check duration stats
	expectedAvg := (5 + 3 + 7 + 4 + 2) * time.Second / 5 // 21/5 = 4.2s
	if metrics.AvgDuration != expectedAvg {
		t.Errorf("AvgDuration = %v, want %v", metrics.AvgDuration, expectedAvg)
	}

	if metrics.MinDuration != 2*time.Second {
		t.Errorf("MinDuration = %v, want 2s", metrics.MinDuration)
	}

	if metrics.MaxDuration != 7*time.Second {
		t.Errorf("MaxDuration = %v, want 7s", metrics.MaxDuration)
	}

	if metrics.LastDuration != 2*time.Second {
		t.Errorf("LastDuration = %v, want 2s", metrics.LastDuration)
	}

	// Check cache hit rate
	// Total: (10+12+0+14+13) = 49 hits, (5+3+15+1+2) = 26 misses
	// Rate: 49/(49+26) = 49/75 = 65.33...%
	expectedCacheRate := 49.0 / 75.0 * 100.0
	if metrics.AvgCacheHitRate < 65.0 || metrics.AvgCacheHitRate > 66.0 {
		t.Errorf("AvgCacheHitRate = %v, want ~%v", metrics.AvgCacheHitRate, expectedCacheRate)
	}
}

func TestBuildingSetPosition(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 100.0, 200.0)

	x, y := b.Position()
	if x != 100.0 || y != 200.0 {
		t.Errorf("initial Position() = (%v, %v), want (100, 200)", x, y)
	}

	// Update position
	b.SetPosition(300.0, 400.0)

	x, y = b.Position()
	if x != 300.0 || y != 400.0 {
		t.Errorf("after SetPosition, Position() = (%v, %v), want (300, 400)", x, y)
	}
}

func TestBuildingBuildHistory(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Initially empty
	history := b.BuildHistory()
	if len(history) != 0 {
		t.Errorf("initial BuildHistory length = %d, want 0", len(history))
	}

	// Record some builds
	for i := 0; i < 3; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  time.Duration(i+1) * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Duration(i+1) * time.Second),
		}
		b.RecordBuild(result)
	}

	history = b.BuildHistory()
	if len(history) != 3 {
		t.Errorf("BuildHistory length = %d, want 3", len(history))
	}

	// Verify history is a copy (mutation doesn't affect original)
	history[0].Success = false
	historyAgain := b.BuildHistory()
	if !historyAgain[0].Success {
		t.Error("BuildHistory returned a reference instead of a copy")
	}
}

func TestBuildingCalculateDurationTrend_NotEnoughData(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Record only 3 builds (need at least 6 for trend)
	for i := 0; i < 3; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  time.Duration(i+1) * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Duration(i+1) * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()
	if metrics.DurationTrend != TrendStable {
		t.Errorf("with < 6 builds, DurationTrend = %q, want %q", metrics.DurationTrend, TrendStable)
	}
}

func TestBuildingCalculateDurationTrend_Improving(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Record builds that get progressively faster
	// First 5: 10s each (avg = 10s)
	// Last 5: 5s each (avg = 5s)
	// Improvement: (10-5)/10 = 50% > 10% threshold
	for i := 0; i < 5; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  10 * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(10 * time.Second),
		}
		b.RecordBuild(result)
	}

	for i := 0; i < 5; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  5 * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(5 * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()
	if metrics.DurationTrend != TrendImproving {
		t.Errorf("with improving times, DurationTrend = %q, want %q", metrics.DurationTrend, TrendImproving)
	}
}

func TestBuildingCalculateDurationTrend_Degrading(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Record builds that get progressively slower
	// First 5: 5s each (avg = 5s)
	// Last 5: 10s each (avg = 10s)
	// Degradation: (10-5)/5 = 100% > 10% threshold
	for i := 0; i < 5; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  5 * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(5 * time.Second),
		}
		b.RecordBuild(result)
	}

	for i := 0; i < 5; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  10 * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(10 * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()
	if metrics.DurationTrend != TrendDegrading {
		t.Errorf("with degrading times, DurationTrend = %q, want %q", metrics.DurationTrend, TrendDegrading)
	}
}

func TestBuildingCalculateDurationTrend_Stable(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Record builds with stable times (within 10% threshold)
	// All builds: ~5s (variation less than 10%)
	for i := 0; i < 10; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  5 * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(5 * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()
	if metrics.DurationTrend != TrendStable {
		t.Errorf("with stable times, DurationTrend = %q, want %q", metrics.DurationTrend, TrendStable)
	}
}

func TestBuildState_Constants(t *testing.T) {
	// Verify BuildState constants are distinct
	states := map[BuildState]bool{
		StateIdle:     true,
		StateBuilding: true,
		StateSuccess:  true,
		StateFailed:   true,
	}

	if len(states) != 4 {
		t.Error("BuildState constants should be distinct")
	}
}

func TestTrend_Constants(t *testing.T) {
	// Verify Trend constants are distinct
	trends := map[Trend]bool{
		TrendImproving: true,
		TrendStable:    true,
		TrendDegrading: true,
	}

	if len(trends) != 3 {
		t.Error("Trend constants should be distinct")
	}
}

func TestBuildingConcurrency(t *testing.T) {
	// Test that concurrent access doesn't cause data races
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	done := make(chan bool)

	// Start multiple goroutines accessing building concurrently
	for i := 0; i < 10; i++ {
		go func() {
			result := &build.Result{
				Success:   true,
				Duration:  time.Second,
				StartTime: time.Now(),
				EndTime:   time.Now().Add(time.Second),
			}
			b.RecordBuild(result)
			_ = b.Metrics()
			_ = b.State()
			_ = b.BuildHistory()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we get here without data race, test passes
	// (run with -race flag to detect races)
}

// --- Numeric Edge Case Tests ---

// TestBuilding_ZeroDuration tests that zero duration builds are handled.
func TestBuilding_ZeroDuration(t *testing.T) {
	target := build.Target{ID: "zero-dur", Name: "zero-dur"}
	b := New(target, "bazel", 0, 0)

	// Zero duration is valid (very fast cached build)
	result := &build.Result{
		Success:   true,
		Duration:  0, // Zero duration
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	b.RecordBuild(result)

	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
	if metrics.MinDuration != 0 {
		t.Errorf("MinDuration = %v, want 0", metrics.MinDuration)
	}
	if metrics.MaxDuration != 0 {
		t.Errorf("MaxDuration = %v, want 0", metrics.MaxDuration)
	}
	if metrics.AvgDuration != 0 {
		t.Errorf("AvgDuration = %v, want 0", metrics.AvgDuration)
	}
}

// TestBuilding_ZeroCacheOperations tests builds with no cache operations.
func TestBuilding_ZeroCacheOperations(t *testing.T) {
	target := build.Target{ID: "no-cache", Name: "no-cache"}
	b := New(target, "npm", 0, 0) // npm might not report cache stats

	result := &build.Result{
		Success:     true,
		Duration:    5 * time.Second,
		CacheHits:   0, // No cache operations
		CacheMisses: 0,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(5 * time.Second),
	}
	b.RecordBuild(result)

	metrics := b.Metrics()

	// AvgCacheHitRate should be 0 when there are no cache operations
	if metrics.AvgCacheHitRate != 0 {
		t.Errorf("AvgCacheHitRate = %v, want 0 (no cache operations)", metrics.AvgCacheHitRate)
	}
}

// TestBuilding_ExactlyFiveBuilds tests the trend boundary (5 builds = stable).
func TestBuilding_ExactlyFiveBuilds(t *testing.T) {
	target := build.Target{ID: "five-builds", Name: "five-builds"}
	b := New(target, "bazel", 0, 0)

	// Record exactly 5 builds with different durations
	// This is below the threshold for trend calculation (needs 6+)
	for i := 0; i < 5; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  time.Duration(i+1) * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Duration(i+1) * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()
	if metrics.TotalBuilds != 5 {
		t.Errorf("TotalBuilds = %d, want 5", metrics.TotalBuilds)
	}
	// With exactly 5 builds, trend should be stable (not enough data)
	if metrics.DurationTrend != TrendStable {
		t.Errorf("DurationTrend = %q, want %q (need 6+ builds for trend)", metrics.DurationTrend, TrendStable)
	}
}

// TestBuilding_ExactlySixBuilds tests the minimum for trend calculation.
func TestBuilding_ExactlySixBuilds(t *testing.T) {
	target := build.Target{ID: "six-builds", Name: "six-builds"}
	b := New(target, "bazel", 0, 0)

	// First build (will be compared against last 5)
	result := &build.Result{
		Success:   true,
		Duration:  10 * time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(10 * time.Second),
	}
	b.RecordBuild(result)

	// Next 5 builds (faster)
	for i := 0; i < 5; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  5 * time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(5 * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()
	if metrics.TotalBuilds != 6 {
		t.Errorf("TotalBuilds = %d, want 6", metrics.TotalBuilds)
	}
	// With 6 builds where last 5 are faster than first, should be improving
	if metrics.DurationTrend != TrendImproving {
		t.Errorf("DurationTrend = %q, want %q", metrics.DurationTrend, TrendImproving)
	}
}

// TestBuilding_TrendBoundaryTenPercent tests exact 10% threshold behavior.
func TestBuilding_TrendBoundaryTenPercent(t *testing.T) {
	// Test exactly at the 10% threshold boundary
	testCases := []struct {
		name          string
		previousAvg   time.Duration
		recentAvg     time.Duration
		expectedTrend Trend
	}{
		{
			name:          "exactly_10_percent_faster",
			previousAvg:   100 * time.Second,
			recentAvg:     90 * time.Second, // exactly 10% faster
			expectedTrend: TrendStable,      // diff == threshold, not < threshold
		},
		{
			name:          "just_over_10_percent_faster",
			previousAvg:   100 * time.Second,
			recentAvg:     89 * time.Second, // 11% faster
			expectedTrend: TrendImproving,
		},
		{
			name:          "just_under_10_percent_faster",
			previousAvg:   100 * time.Second,
			recentAvg:     91 * time.Second, // 9% faster
			expectedTrend: TrendStable,
		},
		{
			name:          "exactly_10_percent_slower",
			previousAvg:   100 * time.Second,
			recentAvg:     110 * time.Second, // exactly 10% slower
			expectedTrend: TrendStable,       // diff == threshold, not > threshold
		},
		{
			name:          "just_over_10_percent_slower",
			previousAvg:   100 * time.Second,
			recentAvg:     111 * time.Second, // 11% slower
			expectedTrend: TrendDegrading,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := build.Target{ID: tc.name, Name: tc.name}
			b := New(target, "bazel", 0, 0)

			// Record 1 build at previousAvg (this is the "previous" builds)
			result := &build.Result{
				Success:   true,
				Duration:  tc.previousAvg,
				StartTime: time.Now(),
				EndTime:   time.Now().Add(tc.previousAvg),
			}
			b.RecordBuild(result)

			// Record 5 builds at recentAvg (these are the "last 5")
			for i := 0; i < 5; i++ {
				result := &build.Result{
					Success:   true,
					Duration:  tc.recentAvg,
					StartTime: time.Now(),
					EndTime:   time.Now().Add(tc.recentAvg),
				}
				b.RecordBuild(result)
			}

			metrics := b.Metrics()
			if metrics.DurationTrend != tc.expectedTrend {
				t.Errorf("DurationTrend = %q, want %q", metrics.DurationTrend, tc.expectedTrend)
			}
		})
	}
}

// --- State Machine Tests ---

// TestBuilding_RecordBuildWithoutStart tests RecordBuild when state is Idle.
func TestBuilding_RecordBuildWithoutStart(t *testing.T) {
	target := build.Target{ID: "no-start", Name: "no-start"}
	b := New(target, "bazel", 0, 0)

	// RecordBuild without StartBuild - should still work (permissive)
	result := &build.Result{
		Success:   true,
		Duration:  5 * time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
	}
	b.RecordBuild(result)

	// State should be success
	if b.State() != StateSuccess {
		t.Errorf("State() = %q, want %q", b.State(), StateSuccess)
	}

	// Metrics should be recorded
	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
}

// TestBuilding_DoubleStartBuild tests calling StartBuild twice.
func TestBuilding_DoubleStartBuild(t *testing.T) {
	target := build.Target{ID: "double-start", Name: "double-start"}
	b := New(target, "bazel", 0, 0)

	// StartBuild twice
	b.StartBuild()
	b.StartBuild()

	// State should still be Building
	if b.State() != StateBuilding {
		t.Errorf("State() = %q, want %q", b.State(), StateBuilding)
	}

	// Metrics should be unchanged (no builds recorded)
	metrics := b.Metrics()
	if metrics.TotalBuilds != 0 {
		t.Errorf("TotalBuilds = %d, want 0", metrics.TotalBuilds)
	}
}

// TestBuilding_DoubleRecordBuild tests calling RecordBuild twice without StartBuild.
func TestBuilding_DoubleRecordBuild(t *testing.T) {
	target := build.Target{ID: "double-record", Name: "double-record"}
	b := New(target, "bazel", 0, 0)

	b.StartBuild()

	result1 := &build.Result{
		Success:   true,
		Duration:  5 * time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
	}
	b.RecordBuild(result1)

	result2 := &build.Result{
		Success:   false,
		Duration:  3 * time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(3 * time.Second),
	}
	b.RecordBuild(result2)

	// Both builds should be recorded
	metrics := b.Metrics()
	if metrics.TotalBuilds != 2 {
		t.Errorf("TotalBuilds = %d, want 2", metrics.TotalBuilds)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", metrics.SuccessCount)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", metrics.FailureCount)
	}

	// State should reflect last build
	if b.State() != StateFailed {
		t.Errorf("State() = %q, want %q", b.State(), StateFailed)
	}
}

// TestBuilding_StateTransitionFromSuccess tests transitioning from Success to Building.
func TestBuilding_StateTransitionFromSuccess(t *testing.T) {
	target := build.Target{ID: "success-to-building", Name: "success-to-building"}
	b := New(target, "bazel", 0, 0)

	// Get to Success state
	b.StartBuild()
	result := &build.Result{
		Success:   true,
		Duration:  time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
	}
	b.RecordBuild(result)

	if b.State() != StateSuccess {
		t.Fatalf("Expected StateSuccess, got %q", b.State())
	}

	// Start a new build
	b.StartBuild()

	if b.State() != StateBuilding {
		t.Errorf("State() = %q, want %q after StartBuild from Success", b.State(), StateBuilding)
	}
}

// TestBuilding_StateTransitionFromFailed tests transitioning from Failed to Building.
func TestBuilding_StateTransitionFromFailed(t *testing.T) {
	target := build.Target{ID: "failed-to-building", Name: "failed-to-building"}
	b := New(target, "bazel", 0, 0)

	// Get to Failed state
	b.StartBuild()
	result := &build.Result{
		Success:      false,
		Duration:     time.Second,
		ErrorMessage: "test error",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(time.Second),
	}
	b.RecordBuild(result)

	if b.State() != StateFailed {
		t.Fatalf("Expected StateFailed, got %q", b.State())
	}

	// Start a new build
	b.StartBuild()

	if b.State() != StateBuilding {
		t.Errorf("State() = %q, want %q after StartBuild from Failed", b.State(), StateBuilding)
	}
}

// TestBuilding_AllStateCombinations tests all valid state transitions.
func TestBuilding_AllStateCombinations(t *testing.T) {
	transitions := []struct {
		name       string
		fromState  BuildState
		action     string
		success    bool
		expectedTo BuildState
	}{
		{"idle_to_building", StateIdle, "start", false, StateBuilding},
		{"building_to_success", StateBuilding, "record", true, StateSuccess},
		{"building_to_failed", StateBuilding, "record", false, StateFailed},
		{"success_to_building", StateSuccess, "start", false, StateBuilding},
		{"failed_to_building", StateFailed, "start", false, StateBuilding},
	}

	for _, tc := range transitions {
		t.Run(tc.name, func(t *testing.T) {
			target := build.Target{ID: tc.name, Name: tc.name}
			b := New(target, "bazel", 0, 0)

			// Get to fromState
			switch tc.fromState {
			case StateIdle:
				// Already there
			case StateBuilding:
				b.StartBuild()
			case StateSuccess:
				b.StartBuild()
				b.RecordBuild(&build.Result{Success: true, Duration: time.Second, StartTime: time.Now(), EndTime: time.Now().Add(time.Second)})
			case StateFailed:
				b.StartBuild()
				b.RecordBuild(&build.Result{Success: false, Duration: time.Second, StartTime: time.Now(), EndTime: time.Now().Add(time.Second)})
			}

			if b.State() != tc.fromState {
				t.Fatalf("Failed to reach fromState %q, got %q", tc.fromState, b.State())
			}

			// Perform action
			switch tc.action {
			case "start":
				b.StartBuild()
			case "record":
				b.RecordBuild(&build.Result{Success: tc.success, Duration: time.Second, StartTime: time.Now(), EndTime: time.Now().Add(time.Second)})
			}

			// Verify result
			if b.State() != tc.expectedTo {
				t.Errorf("After %s from %s: State() = %q, want %q", tc.action, tc.fromState, b.State(), tc.expectedTo)
			}
		})
	}
}

// TestBuilding_NegativeDuration tests that negative duration doesn't crash.
// This is a defensive edge case - shouldn't happen in practice.
func TestBuilding_NegativeDuration(t *testing.T) {
	target := build.Target{ID: "neg-dur", Name: "neg-dur"}
	b := New(target, "bazel", 0, 0)

	// Negative duration (shouldn't happen, but be defensive)
	result := &build.Result{
		Success:   true,
		Duration:  -5 * time.Second,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(-5 * time.Second),
	}
	b.RecordBuild(result)

	// Should still record the build
	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
	// MinDuration might be negative - that's the data we got
	if metrics.MinDuration != -5*time.Second {
		t.Errorf("MinDuration = %v, want -5s (preserves input)", metrics.MinDuration)
	}
}

// TestBuilding_LargeValues tests handling of large numeric values.
func TestBuilding_LargeValues(t *testing.T) {
	target := build.Target{ID: "large", Name: "large"}
	b := New(target, "bazel", 0, 0)

	// Use large but not overflow-inducing values
	// int64 max is ~9.2e18, so 1e15 + 1e15 = 2e15 is safe
	result := &build.Result{
		Success:     true,
		Duration:    24 * time.Hour * 365, // 1 year (unrealistic but valid)
		CacheHits:   int64(1e15),          // Large but sum won't overflow
		CacheMisses: int64(1e15),
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(24 * time.Hour * 365),
	}
	b.RecordBuild(result)

	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
	// Cache hit rate should be 50% (equal hits and misses)
	if metrics.AvgCacheHitRate != 50.0 {
		t.Errorf("AvgCacheHitRate = %v, want 50.0", metrics.AvgCacheHitRate)
	}
}
