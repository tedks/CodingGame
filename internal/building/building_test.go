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
