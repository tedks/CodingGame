package building

import (
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/build"
)

// Benchmarks for building operations to track performance characteristics.

// BenchmarkRecordBuild measures time to record a single build.
func BenchmarkRecordBuild(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	result := &build.Result{
		Success:     true,
		Duration:    5 * time.Second,
		CacheHits:   50,
		CacheMisses: 10,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(5 * time.Second),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		building.RecordBuild(result)
	}
}

// BenchmarkRecordBuild_Parallel measures concurrent build recording.
func BenchmarkRecordBuild_Parallel(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	result := &build.Result{
		Success:     true,
		Duration:    5 * time.Second,
		CacheHits:   50,
		CacheMisses: 10,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(5 * time.Second),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			building.RecordBuild(result)
		}
	})
}

// BenchmarkMetrics measures time to retrieve metrics.
func BenchmarkMetrics(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	// Pre-populate with some builds
	for i := 0; i < 100; i++ {
		result := &build.Result{
			Success:   i%2 == 0,
			Duration:  time.Duration(i+1) * time.Millisecond,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Millisecond),
		}
		building.RecordBuild(result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = building.Metrics()
	}
}

// BenchmarkMetrics_LargeHistory measures metrics retrieval with 10k builds.
func BenchmarkMetrics_LargeHistory(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	// Pre-populate with many builds
	for i := 0; i < 10000; i++ {
		result := &build.Result{
			Success:     i%3 != 0,
			Duration:    time.Duration(i%100+1) * time.Millisecond,
			CacheHits:   int64(i % 100),
			CacheMisses: int64(100 - (i % 100)),
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Millisecond),
		}
		building.RecordBuild(result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = building.Metrics()
	}
}

// BenchmarkBuildHistory measures time to copy build history.
func BenchmarkBuildHistory(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	// Pre-populate with builds
	for i := 0; i < 100; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		}
		building.RecordBuild(result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = building.BuildHistory()
	}
}

// BenchmarkBuildHistory_LargeHistory measures history copy with 10k builds.
func BenchmarkBuildHistory_LargeHistory(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	// Pre-populate with many builds
	for i := 0; i < 10000; i++ {
		result := &build.Result{
			Success:     true,
			Duration:    time.Duration(i+1) * time.Microsecond,
			CacheHits:   int64(i),
			CacheMisses: int64(10000 - i),
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Microsecond),
		}
		building.RecordBuild(result)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = building.BuildHistory()
	}
}

// BenchmarkState measures time to read state.
func BenchmarkState(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = building.State()
	}
}

// BenchmarkState_Parallel measures concurrent state reads.
func BenchmarkState_Parallel(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = building.State()
		}
	})
}

// BenchmarkStartBuild measures time to start a build.
func BenchmarkStartBuild(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		building.StartBuild()
	}
}

// BenchmarkSetPosition measures time to update position.
func BenchmarkSetPosition(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		building.SetPosition(float64(i), float64(i*2))
	}
}

// BenchmarkPosition measures time to read position.
func BenchmarkPosition(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = building.Position()
	}
}

// BenchmarkFullLifecycle measures a complete build lifecycle.
func BenchmarkFullLifecycle(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}

	result := &build.Result{
		Success:     true,
		Duration:    5 * time.Second,
		CacheHits:   50,
		CacheMisses: 10,
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(5 * time.Second),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		building := New(target, "bazel", 0, 0)
		building.StartBuild()
		building.RecordBuild(result)
		_ = building.Metrics()
		_ = building.State()
	}
}

// BenchmarkNew measures time to create a new building.
func BenchmarkNew(b *testing.B) {
	target := build.Target{
		ID:   "//cmd/app:app",
		Name: "app",
		Type: build.TargetTypeBinary,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New(target, "bazel", float64(i), float64(i*2))
	}
}

// BenchmarkMetrics_Parallel measures concurrent metrics reads.
func BenchmarkMetrics_Parallel(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		result := &build.Result{
			Success:   i%2 == 0,
			Duration:  time.Duration(i+1) * time.Millisecond,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Millisecond),
		}
		building.RecordBuild(result)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = building.Metrics()
		}
	})
}

// BenchmarkDurationTrendCalculation measures trend calculation cost.
// This is interesting because trend calculation is O(n) over history.
func BenchmarkDurationTrendCalculation(b *testing.B) {
	target := build.Target{ID: "bench", Name: "bench"}
	building := New(target, "bazel", 0, 0)

	// Pre-populate with exactly 10 builds for trend calculation
	for i := 0; i < 10; i++ {
		result := &build.Result{
			Success:   true,
			Duration:  time.Duration(10-i) * time.Second, // Decreasing durations
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		}
		building.RecordBuild(result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each RecordBuild triggers trend calculation
		result := &build.Result{
			Success:   true,
			Duration:  time.Second,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Second),
		}
		building.RecordBuild(result)
	}
}
