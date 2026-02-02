package building

import (
	"math/rand"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/build"
)

// Property invariant tests verify that certain mathematical relationships
// hold regardless of the sequence of operations.

// TestProperty_MetricInvariants verifies TotalBuilds = SuccessCount + FailureCount
// for any sequence of builds.
func TestProperty_MetricInvariants(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	// Use deterministic seed for reproducibility
	rng := rand.New(rand.NewSource(42))

	// Record 1000 random builds
	for i := 0; i < 1000; i++ {
		success := rng.Float64() < 0.7 // 70% success rate
		result := &build.Result{
			Success:     success,
			Duration:    time.Duration(rng.Intn(10)+1) * time.Second,
			CacheHits:   int64(rng.Intn(100)),
			CacheMisses: int64(rng.Intn(100)),
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Second),
		}
		b.RecordBuild(result)

		// After each build, verify the invariant
		metrics := b.Metrics()

		// Invariant 1: TotalBuilds = SuccessCount + FailureCount
		if metrics.TotalBuilds != metrics.SuccessCount+metrics.FailureCount {
			t.Fatalf("Invariant violated at build %d: TotalBuilds(%d) != SuccessCount(%d) + FailureCount(%d)",
				i+1, metrics.TotalBuilds, metrics.SuccessCount, metrics.FailureCount)
		}

		// Invariant 2: TotalBuilds == i+1
		if metrics.TotalBuilds != i+1 {
			t.Fatalf("Invariant violated at build %d: TotalBuilds(%d) != expected(%d)",
				i+1, metrics.TotalBuilds, i+1)
		}
	}
}

// TestProperty_SuccessRateBounds verifies 0 <= SuccessRate <= 100 for any sequence.
func TestProperty_SuccessRateBounds(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 500; i++ {
		result := &build.Result{
			Success:   rng.Float64() < 0.5,
			Duration:  time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()

		// Invariant: 0 <= SuccessRate <= 100
		if metrics.SuccessRate < 0 || metrics.SuccessRate > 100 {
			t.Fatalf("SuccessRate out of bounds at build %d: %f (should be 0-100)",
				i+1, metrics.SuccessRate)
		}
	}
}

// TestProperty_DurationBounds verifies MinDuration <= AvgDuration <= MaxDuration.
func TestProperty_DurationBounds(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 500; i++ {
		// Random duration between 1 and 100 seconds
		duration := time.Duration(rng.Intn(100)+1) * time.Second
		result := &build.Result{
			Success:   true,
			Duration:  duration,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(duration),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()

		// Invariant: MinDuration <= AvgDuration <= MaxDuration
		if metrics.MinDuration > metrics.AvgDuration {
			t.Fatalf("Invariant violated at build %d: MinDuration(%v) > AvgDuration(%v)",
				i+1, metrics.MinDuration, metrics.AvgDuration)
		}
		if metrics.AvgDuration > metrics.MaxDuration {
			t.Fatalf("Invariant violated at build %d: AvgDuration(%v) > MaxDuration(%v)",
				i+1, metrics.AvgDuration, metrics.MaxDuration)
		}
	}
}

// TestProperty_HistoryConsistency verifies len(BuildHistory()) == TotalBuilds.
func TestProperty_HistoryConsistency(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 500; i++ {
		result := &build.Result{
			Success:     rng.Float64() < 0.7,
			Duration:    time.Duration(rng.Intn(10)+1) * time.Second,
			CacheHits:   int64(rng.Intn(50)),
			CacheMisses: int64(rng.Intn(50)),
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Second),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()
		history := b.BuildHistory()

		// Invariant: len(BuildHistory()) == TotalBuilds
		if len(history) != metrics.TotalBuilds {
			t.Fatalf("Invariant violated at build %d: len(history)(%d) != TotalBuilds(%d)",
				i+1, len(history), metrics.TotalBuilds)
		}
	}
}

// TestProperty_CacheHitRateBounds verifies 0 <= AvgCacheHitRate <= 100.
func TestProperty_CacheHitRateBounds(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 500; i++ {
		result := &build.Result{
			Success:     true,
			Duration:    time.Second,
			CacheHits:   int64(rng.Intn(100)),
			CacheMisses: int64(rng.Intn(100)),
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Second),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()

		// Invariant: 0 <= AvgCacheHitRate <= 100
		if metrics.AvgCacheHitRate < 0 || metrics.AvgCacheHitRate > 100 {
			t.Fatalf("AvgCacheHitRate out of bounds at build %d: %f (should be 0-100)",
				i+1, metrics.AvgCacheHitRate)
		}
	}
}

// TestProperty_MonotonicTotalBuilds verifies TotalBuilds never decreases.
func TestProperty_MonotonicTotalBuilds(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))
	lastTotal := 0

	for i := 0; i < 500; i++ {
		result := &build.Result{
			Success:   rng.Float64() < 0.5,
			Duration:  time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()

		// Invariant: TotalBuilds is monotonically increasing
		if metrics.TotalBuilds < lastTotal {
			t.Fatalf("TotalBuilds decreased at build %d: was %d, now %d",
				i+1, lastTotal, metrics.TotalBuilds)
		}
		if metrics.TotalBuilds != lastTotal+1 {
			t.Fatalf("TotalBuilds didn't increment at build %d: was %d, now %d",
				i+1, lastTotal, metrics.TotalBuilds)
		}
		lastTotal = metrics.TotalBuilds
	}
}

// TestProperty_LastDurationMatchesHistory verifies LastDuration matches the last history entry.
func TestProperty_LastDurationMatchesHistory(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 100; i++ {
		duration := time.Duration(rng.Intn(100)+1) * time.Second
		result := &build.Result{
			Success:   true,
			Duration:  duration,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(duration),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()
		history := b.BuildHistory()

		// Invariant: LastDuration == history[len(history)-1].Duration
		lastHistoryDuration := history[len(history)-1].Duration
		if metrics.LastDuration != lastHistoryDuration {
			t.Fatalf("Invariant violated at build %d: LastDuration(%v) != history[-1].Duration(%v)",
				i+1, metrics.LastDuration, lastHistoryDuration)
		}
	}
}

// TestProperty_SuccessRateCalculation verifies SuccessRate = (SuccessCount/TotalBuilds)*100.
func TestProperty_SuccessRateCalculation(t *testing.T) {
	target := build.Target{ID: "test", Name: "test"}
	b := New(target, "bazel", 0, 0)

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 500; i++ {
		result := &build.Result{
			Success:   rng.Float64() < 0.6,
			Duration:  time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		}
		b.RecordBuild(result)

		metrics := b.Metrics()

		// Invariant: SuccessRate == (SuccessCount / TotalBuilds) * 100
		expectedRate := float64(metrics.SuccessCount) / float64(metrics.TotalBuilds) * 100.0
		if metrics.SuccessRate != expectedRate {
			t.Fatalf("SuccessRate mismatch at build %d: got %f, expected %f",
				i+1, metrics.SuccessRate, expectedRate)
		}
	}
}
