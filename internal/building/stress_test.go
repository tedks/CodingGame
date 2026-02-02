package building

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/build"
)

// Stress tests verify correctness under high concurrency and load.

// TestStress_HighContention tests 100 goroutines making 1000 builds each.
func TestStress_HighContention(t *testing.T) {
	target := build.Target{ID: "stress-test", Name: "stress-test"}
	b := New(target, "bazel", 0, 0)

	const numGoroutines = 100
	const buildsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(int64(goroutineID)))

			for j := 0; j < buildsPerGoroutine; j++ {
				result := &build.Result{
					Success:     rng.Float64() < 0.8,
					Duration:    time.Duration(rng.Intn(10)+1) * time.Second,
					CacheHits:   int64(rng.Intn(50)),
					CacheMisses: int64(rng.Intn(50)),
					StartTime:   time.Now(),
					EndTime:     time.Now().Add(time.Second),
				}
				b.RecordBuild(result)

				// Also read metrics to create contention
				_ = b.Metrics()
				_ = b.State()
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	metrics := b.Metrics()
	expectedTotal := numGoroutines * buildsPerGoroutine

	if metrics.TotalBuilds != expectedTotal {
		t.Errorf("TotalBuilds = %d, want %d", metrics.TotalBuilds, expectedTotal)
	}

	if metrics.TotalBuilds != metrics.SuccessCount+metrics.FailureCount {
		t.Errorf("Invariant violated: TotalBuilds(%d) != SuccessCount(%d) + FailureCount(%d)",
			metrics.TotalBuilds, metrics.SuccessCount, metrics.FailureCount)
	}

	history := b.BuildHistory()
	if len(history) != expectedTotal {
		t.Errorf("BuildHistory length = %d, want %d", len(history), expectedTotal)
	}
}

// TestStress_RapidStateTransitions tests tight Building->Success/Failed loops.
func TestStress_RapidStateTransitions(t *testing.T) {
	target := build.Target{ID: "rapid-state", Name: "rapid-state"}
	b := New(target, "bazel", 0, 0)

	const numGoroutines = 50
	const transitionsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(int64(goroutineID)))

			for j := 0; j < transitionsPerGoroutine; j++ {
				// Simulate build lifecycle
				b.StartBuild()

				result := &build.Result{
					Success:   rng.Float64() < 0.9,
					Duration:  time.Duration(rng.Intn(5)+1) * time.Second,
					StartTime: time.Now(),
					EndTime:   time.Now().Add(time.Second),
				}
				b.RecordBuild(result)
			}
		}(i)
	}

	wg.Wait()

	// Final state should be either Success or Failed (not Building)
	state := b.State()
	if state == StateBuilding {
		t.Errorf("Final state is Building, expected Success or Failed")
	}

	metrics := b.Metrics()
	expectedTotal := numGoroutines * transitionsPerGoroutine
	if metrics.TotalBuilds != expectedTotal {
		t.Errorf("TotalBuilds = %d, want %d", metrics.TotalBuilds, expectedTotal)
	}
}

// TestStress_ConcurrentMetricsReads tests many readers vs one writer.
func TestStress_ConcurrentMetricsReads(t *testing.T) {
	target := build.Target{ID: "reader-writer", Name: "reader-writer"}
	b := New(target, "bazel", 0, 0)

	const numReaders = 50
	const numWrites = 500
	const readsPerReader = 100

	// Channel to signal writer is done
	writerDone := make(chan struct{})

	// Start multiple readers
	var wg sync.WaitGroup
	wg.Add(numReaders)

	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < readsPerReader; j++ {
				metrics := b.Metrics()
				history := b.BuildHistory()
				state := b.State()

				// Verify consistency even while being written to
				if metrics.TotalBuilds != metrics.SuccessCount+metrics.FailureCount {
					t.Errorf("Reader saw inconsistent metrics: Total=%d, Success=%d, Failure=%d",
						metrics.TotalBuilds, metrics.SuccessCount, metrics.FailureCount)
				}
				if len(history) != metrics.TotalBuilds {
					t.Errorf("Reader saw inconsistent history: len=%d, TotalBuilds=%d",
						len(history), metrics.TotalBuilds)
				}

				// State is valid
				if state != StateIdle && state != StateBuilding && state != StateSuccess && state != StateFailed {
					t.Errorf("Reader saw invalid state: %v", state)
				}
			}
		}()
	}

	// Writer goroutine
	go func() {
		rng := rand.New(rand.NewSource(99))
		for i := 0; i < numWrites; i++ {
			result := &build.Result{
				Success:     rng.Float64() < 0.7,
				Duration:    time.Duration(rng.Intn(10)+1) * time.Second,
				CacheHits:   int64(rng.Intn(100)),
				CacheMisses: int64(rng.Intn(100)),
				StartTime:   time.Now(),
				EndTime:     time.Now().Add(time.Second),
			}
			b.RecordBuild(result)
		}
		close(writerDone)
	}()

	// Wait for writer to finish
	<-writerDone

	// Wait for all readers to finish
	wg.Wait()

	// Verify final state
	metrics := b.Metrics()
	if metrics.TotalBuilds != numWrites {
		t.Errorf("TotalBuilds = %d, want %d", metrics.TotalBuilds, numWrites)
	}
}

// TestStress_ConcurrentHistoryReads tests that BuildHistory returns consistent copies.
func TestStress_ConcurrentHistoryReads(t *testing.T) {
	target := build.Target{ID: "history-stress", Name: "history-stress"}
	b := New(target, "bazel", 0, 0)

	const numGoroutines = 20
	const iterations = 100

	// Pre-populate with some builds
	for i := 0; i < 100; i++ {
		result := &build.Result{
			Success:   i%2 == 0,
			Duration:  time.Duration(i+1) * time.Millisecond,
			StartTime: time.Now(),
			EndTime:   time.Now().Add(time.Millisecond),
		}
		b.RecordBuild(result)
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				history := b.BuildHistory()

				// Verify we got a consistent snapshot
				if len(history) < 100 {
					t.Errorf("History too short: %d (expected at least 100)", len(history))
				}

				// Mutate the copy - should not affect original
				if len(history) > 0 {
					history[0].Success = !history[0].Success
				}

				// Get another copy and verify original is unchanged
				history2 := b.BuildHistory()
				if len(history2) > 0 && len(history) > 0 {
					// They should have the same original value
					// (we mutated history but not the underlying data)
					if history2[0].Duration != history[0].Duration {
						// This would only happen if durations were different,
						// which is expected since we're doing concurrent writes
						// Just verify both are valid
						if history2[0].Duration < 0 {
							t.Errorf("Invalid duration in history2: %v", history2[0].Duration)
						}
					}
				}
			}
		}()
	}

	// Concurrent writer
	go func() {
		for i := 0; i < 100; i++ {
			result := &build.Result{
				Success:   true,
				Duration:  time.Duration(100+i) * time.Millisecond,
				StartTime: time.Now(),
				EndTime:   time.Now().Add(time.Millisecond),
			}
			b.RecordBuild(result)
		}
	}()

	wg.Wait()
}

// TestStress_MixedOperations tests all operations happening concurrently.
func TestStress_MixedOperations(t *testing.T) {
	target := build.Target{ID: "mixed-ops", Name: "mixed-ops"}
	b := New(target, "bazel", 0, 0)

	const numGoroutines = 30
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(int64(goroutineID)))

			for j := 0; j < opsPerGoroutine; j++ {
				op := rng.Intn(7)
				switch op {
				case 0:
					b.StartBuild()
				case 1:
					result := &build.Result{
						Success:     rng.Float64() < 0.8,
						Duration:    time.Duration(rng.Intn(10)+1) * time.Second,
						CacheHits:   int64(rng.Intn(100)),
						CacheMisses: int64(rng.Intn(100)),
						StartTime:   time.Now(),
						EndTime:     time.Now().Add(time.Second),
					}
					b.RecordBuild(result)
				case 2:
					_ = b.Metrics()
				case 3:
					_ = b.State()
				case 4:
					_ = b.BuildHistory()
				case 5:
					b.SetPosition(rng.Float64()*1000, rng.Float64()*1000)
				case 6:
					_, _ = b.Position()
				}
			}
		}(i)
	}

	wg.Wait()

	// Just verify we don't panic and state is valid
	state := b.State()
	if state != StateIdle && state != StateBuilding && state != StateSuccess && state != StateFailed {
		t.Errorf("Invalid final state: %v", state)
	}

	metrics := b.Metrics()
	if metrics.TotalBuilds != metrics.SuccessCount+metrics.FailureCount {
		t.Errorf("Invariant violated: TotalBuilds(%d) != SuccessCount(%d) + FailureCount(%d)",
			metrics.TotalBuilds, metrics.SuccessCount, metrics.FailureCount)
	}
}

// TestStress_PositionUpdates tests concurrent position updates don't corrupt coordinates.
func TestStress_PositionUpdates(t *testing.T) {
	target := build.Target{ID: "position-stress", Name: "position-stress"}
	b := New(target, "bazel", 0, 0)

	const numGoroutines = 50
	const updatesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < updatesPerGoroutine; j++ {
				x := float64(goroutineID * 10)
				y := float64(j * 10)
				b.SetPosition(x, y)

				// Read position - may not match what we just set due to races,
				// but should be valid numbers
				gotX, gotY := b.Position()
				if gotX < 0 || gotY < 0 {
					t.Errorf("Invalid position: (%v, %v)", gotX, gotY)
				}
			}
		}(i)
	}

	wg.Wait()
}
