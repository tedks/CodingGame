package building

import (
	"sync"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/build"
)

// TestIntegration_BuildingLifecycle tests a complete building lifecycle
func TestIntegration_BuildingLifecycle(t *testing.T) {
	target := build.Target{
		ID:   "//app:server",
		Name: "server",
		Type: build.TargetTypeBinary,
	}

	b := New(target, "bazel", 100.0, 200.0)

	// 1. Initial state should be idle
	if b.State() != StateIdle {
		t.Errorf("Initial state = %v, want %v", b.State(), StateIdle)
	}

	// 2. Start a build
	b.StartBuild()
	if b.State() != StateBuilding {
		t.Errorf("After StartBuild, state = %v, want %v", b.State(), StateBuilding)
	}

	// 3. Simulate a successful build
	result := &build.Result{
		Success:      true,
		Duration:     3 * time.Second,
		CacheHits:    50,
		CacheMisses:  10,
		TargetsBuilt: 60,
		StartTime:    time.Now().Add(-3 * time.Second),
		EndTime:      time.Now(),
	}
	b.RecordBuild(result)

	// 4. State should be success
	if b.State() != StateSuccess {
		t.Errorf("After successful build, state = %v, want %v", b.State(), StateSuccess)
	}

	// 5. Metrics should be updated
	metrics := b.Metrics()
	if metrics.TotalBuilds != 1 {
		t.Errorf("TotalBuilds = %d, want 1", metrics.TotalBuilds)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", metrics.SuccessCount)
	}

	// 6. Start another build and fail it
	b.StartBuild()
	failResult := &build.Result{
		Success:      false,
		Duration:     1 * time.Second,
		ErrorMessage: "compilation failed",
		StartTime:    time.Now().Add(-1 * time.Second),
		EndTime:      time.Now(),
	}
	b.RecordBuild(failResult)

	// 7. State should be failed
	if b.State() != StateFailed {
		t.Errorf("After failed build, state = %v, want %v", b.State(), StateFailed)
	}

	// 8. Metrics should reflect both builds
	metrics = b.Metrics()
	if metrics.TotalBuilds != 2 {
		t.Errorf("TotalBuilds = %d, want 2", metrics.TotalBuilds)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", metrics.SuccessCount)
	}
	if metrics.FailureCount != 1 {
		t.Errorf("FailureCount = %d, want 1", metrics.FailureCount)
	}
	if metrics.SuccessRate != 50.0 {
		t.Errorf("SuccessRate = %v, want 50.0", metrics.SuccessRate)
	}
}

// TestIntegration_MultipleBuildings tests managing multiple buildings
func TestIntegration_MultipleBuildings(t *testing.T) {
	buildings := make([]*Building, 10)

	// Create 10 buildings
	for i := 0; i < 10; i++ {
		target := build.Target{
			ID:   string(rune('A' + i)),
			Name: string(rune('A' + i)),
			Type: build.TargetTypeBinary,
		}
		buildings[i] = New(target, "bazel", float64(i*100), float64(i*100))
	}

	// Build all of them concurrently
	var wg sync.WaitGroup
	for i, bldg := range buildings {
		wg.Add(1)
		go func(idx int, b *Building) {
			defer wg.Done()

			b.StartBuild()

			result := &build.Result{
				Success:      idx%2 == 0, // Alternate success/failure
				Duration:     time.Duration(idx+1) * time.Second,
				CacheHits:    int64(idx * 10),
				CacheMisses:  int64(idx * 5),
				TargetsBuilt: idx + 1,
				StartTime:    time.Now(),
				EndTime:      time.Now().Add(time.Duration(idx+1) * time.Second),
			}
			b.RecordBuild(result)
		}(i, bldg)
	}

	wg.Wait()

	// Verify all buildings were updated
	successCount := 0
	failCount := 0
	for _, b := range buildings {
		state := b.State()
		if state == StateSuccess {
			successCount++
		} else if state == StateFailed {
			failCount++
		}

		metrics := b.Metrics()
		if metrics.TotalBuilds != 1 {
			t.Errorf("Building %s has TotalBuilds = %d, want 1", b.ID(), metrics.TotalBuilds)
		}
	}

	if successCount != 5 || failCount != 5 {
		t.Errorf("Expected 5 successes and 5 failures, got %d successes and %d failures", successCount, failCount)
	}
}

// TestIntegration_BuildingTrendAnalysis tests trend detection over time
func TestIntegration_BuildingTrendAnalysis(t *testing.T) {
	target := build.Target{
		ID:   "//lib:core",
		Name: "core",
		Type: build.TargetTypeLibrary,
	}

	b := New(target, "cargo", 0, 0)

	// Simulate 10 builds that get progressively faster
	for i := 0; i < 10; i++ {
		duration := time.Duration(10-i) * time.Second // 10s, 9s, 8s, ..., 1s
		result := &build.Result{
			Success:      true,
			Duration:     duration,
			CacheHits:    int64(i * 5),
			CacheMisses:  int64(10 - i),
			TargetsBuilt: 10,
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(duration),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()

	// With 10 builds, we should have enough data for trend
	if metrics.TotalBuilds != 10 {
		t.Fatalf("TotalBuilds = %d, want 10", metrics.TotalBuilds)
	}

	// Trend should show improvement (builds getting faster)
	if metrics.DurationTrend != TrendImproving {
		t.Errorf("DurationTrend = %v, want %v (builds are getting faster)", metrics.DurationTrend, TrendImproving)
	}

	// Average should be around 5.5s (average of 10,9,8...1)
	expectedAvg := 55 * time.Second / 10 // (10+9+8+...+1)/10 = 55/10 = 5.5
	if metrics.AvgDuration != expectedAvg {
		t.Errorf("AvgDuration = %v, want %v", metrics.AvgDuration, expectedAvg)
	}

	// Min should be 1s, max should be 10s
	if metrics.MinDuration != 1*time.Second {
		t.Errorf("MinDuration = %v, want 1s", metrics.MinDuration)
	}
	if metrics.MaxDuration != 10*time.Second {
		t.Errorf("MaxDuration = %v, want 10s", metrics.MaxDuration)
	}
}

// TestIntegration_BuildingCacheMetrics tests cache metrics aggregation
func TestIntegration_BuildingCacheMetrics(t *testing.T) {
	target := build.Target{
		ID:   "//app:frontend",
		Name: "frontend",
		Type: build.TargetTypeBinary,
	}

	b := New(target, "npm", 0, 0)

	// Simulate builds with varying cache performance
	builds := []struct {
		cacheHits   int64
		cacheMisses int64
	}{
		{80, 20}, // 80% hit rate
		{90, 10}, // 90% hit rate
		{70, 30}, // 70% hit rate
		{85, 15}, // 85% hit rate
		{75, 25}, // 75% hit rate
	}

	for _, bld := range builds {
		result := &build.Result{
			Success:      true,
			Duration:     5 * time.Second,
			CacheHits:    bld.cacheHits,
			CacheMisses:  bld.cacheMisses,
			TargetsBuilt: int(bld.cacheHits + bld.cacheMisses),
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(5 * time.Second),
		}
		b.RecordBuild(result)
	}

	metrics := b.Metrics()

	// Calculate expected average cache hit rate
	// Total hits: 80+90+70+85+75 = 400
	// Total ops: (80+20)+(90+10)+(70+30)+(85+15)+(75+25) = 500
	// Rate: 400/500 = 80%
	expectedRate := 80.0
	if metrics.AvgCacheHitRate != expectedRate {
		t.Errorf("AvgCacheHitRate = %v, want %v", metrics.AvgCacheHitRate, expectedRate)
	}
}

// TestIntegration_BuildingPositionTracking tests position updates
func TestIntegration_BuildingPositionTracking(t *testing.T) {
	target := build.Target{
		ID:   "//test:unit",
		Name: "unit",
		Type: build.TargetTypeTest,
	}

	b := New(target, "bazel", 0, 0)

	// Move the building around the map
	positions := [][2]float64{
		{100, 100},
		{200, 150},
		{300, 200},
		{150, 250},
		{50, 50},
	}

	for _, pos := range positions {
		b.SetPosition(pos[0], pos[1])
		x, y := b.Position()
		if x != pos[0] || y != pos[1] {
			t.Errorf("After SetPosition(%v, %v), Position() = (%v, %v)", pos[0], pos[1], x, y)
		}
	}
}

// TestIntegration_BuildingHistoryManagement tests build history
func TestIntegration_BuildingHistoryManagement(t *testing.T) {
	target := build.Target{
		ID:   "//lib:database",
		Name: "database",
		Type: build.TargetTypeLibrary,
	}

	b := New(target, "bazel", 0, 0)

	// Record many builds
	const numBuilds = 100
	for i := 0; i < numBuilds; i++ {
		result := &build.Result{
			Success:      i%3 != 0, // Fail every 3rd build
			Duration:     time.Duration(i+1) * time.Millisecond,
			CacheHits:    int64(i),
			CacheMisses:  int64(100 - i),
			TargetsBuilt: 100,
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(time.Duration(i+1) * time.Millisecond),
		}
		b.RecordBuild(result)
	}

	// Verify history length
	history := b.BuildHistory()
	if len(history) != numBuilds {
		t.Errorf("BuildHistory length = %d, want %d", len(history), numBuilds)
	}

	// Verify history is ordered (oldest to newest)
	for i := 1; i < len(history); i++ {
		if history[i].Timestamp.Before(history[i-1].Timestamp) {
			t.Error("Build history is not ordered chronologically")
			break
		}
	}

	// Verify metrics
	metrics := b.Metrics()
	if metrics.TotalBuilds != numBuilds {
		t.Errorf("TotalBuilds = %d, want %d", metrics.TotalBuilds, numBuilds)
	}

	// Expected: 66 successes (100 - 34 failures where i%3==0)
	// Count of values where i%3==0 for i in [0, 99]: 0, 3, 6, ..., 99 = 34 values
	expectedSuccesses := 66
	if metrics.SuccessCount != expectedSuccesses {
		t.Errorf("SuccessCount = %d, want %d", metrics.SuccessCount, expectedSuccesses)
	}
}

// TestIntegration_BuildingStateTransitions tests all state transitions
func TestIntegration_BuildingStateTransitions(t *testing.T) {
	target := build.Target{
		ID:   "//service:api",
		Name: "api",
		Type: build.TargetTypeBinary,
	}

	b := New(target, "cargo", 0, 0)

	transitions := []struct {
		action        func()
		expectedState BuildState
	}{
		{
			action:        func() {}, // Initial state
			expectedState: StateIdle,
		},
		{
			action:        func() { b.StartBuild() },
			expectedState: StateBuilding,
		},
		{
			action: func() {
				b.RecordBuild(&build.Result{
					Success:   true,
					Duration:  1 * time.Second,
					StartTime: time.Now(),
					EndTime:   time.Now().Add(1 * time.Second),
				})
			},
			expectedState: StateSuccess,
		},
		{
			action:        func() { b.StartBuild() },
			expectedState: StateBuilding,
		},
		{
			action: func() {
				b.RecordBuild(&build.Result{
					Success:      false,
					Duration:     1 * time.Second,
					ErrorMessage: "build failed",
					StartTime:    time.Now(),
					EndTime:      time.Now().Add(1 * time.Second),
				})
			},
			expectedState: StateFailed,
		},
		{
			action:        func() { b.StartBuild() },
			expectedState: StateBuilding,
		},
		{
			action: func() {
				b.RecordBuild(&build.Result{
					Success:   true,
					Duration:  2 * time.Second,
					StartTime: time.Now(),
					EndTime:   time.Now().Add(2 * time.Second),
				})
			},
			expectedState: StateSuccess,
		},
	}

	for i, transition := range transitions {
		transition.action()
		state := b.State()
		if state != transition.expectedState {
			t.Errorf("Transition %d: state = %v, want %v", i, state, transition.expectedState)
		}
	}
}

// TestIntegration_BuildingWithDifferentBuildSystems tests buildings from different build systems
func TestIntegration_BuildingWithDifferentBuildSystems(t *testing.T) {
	buildSystems := []string{"bazel", "npm", "cargo", "gradle", "maven", "make"}

	for _, system := range buildSystems {
		target := build.Target{
			ID:   system + ":target",
			Name: system + "-target",
			Type: build.TargetTypeBinary,
		}

		b := New(target, system, 0, 0)

		if b.BuildSystem() != system {
			t.Errorf("Building created with system %q has BuildSystem() = %q", system, b.BuildSystem())
		}

		// Record a build
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

		// Metrics should work regardless of build system
		metrics := b.Metrics()
		if metrics.TotalBuilds != 1 {
			t.Errorf("Building with system %q has TotalBuilds = %d, want 1", system, metrics.TotalBuilds)
		}
	}
}
