// Package building provides visualization of build targets as "buildings" in the
// RTS-style game interface. Buildings represent real build targets (Bazel packages,
// npm scripts, cargo crates) and display actual build metrics rather than synthetic stats.
//
// Each building tracks:
// - Build duration (real wall-clock time)
// - Cache hit rates (from build system)
// - Success/failure history
// - Dependency relationships
//
// The building metaphor makes build infrastructure tangible and navigable, helping
// developers understand project structure and identify optimization opportunities.
package building

import (
	"sync"
	"time"

	"github.com/tedks/CodingGame/internal/build"
)

// Building represents a build target visualized as a structure
type Building struct {
	mu sync.RWMutex

	// Identity
	id          string           // Unique ID (target ID from build system)
	name        string           // Display name
	targetType  build.TargetType // Binary, library, test, etc.
	buildSystem string           // "bazel", "npm", "cargo", etc.

	// Location (for map rendering)
	x float64
	y float64

	// State
	state      BuildState
	lastBuild  *build.Result
	buildQueue []BuildRequest

	// Metrics (historical)
	buildHistory []BuildRecord
	metrics      Metrics
}

// BuildState represents the current state of a building
type BuildState string

const (
	StateIdle     BuildState = "idle"     // Not building
	StateBuilding BuildState = "building" // Build in progress
	StateSuccess  BuildState = "success"  // Last build succeeded
	StateFailed   BuildState = "failed"   // Last build failed
)

// BuildRequest represents a queued build operation
type BuildRequest struct {
	Options   *build.BuildOptions
	Timestamp time.Time
}

// BuildRecord stores historical build information
type BuildRecord struct {
	Timestamp   time.Time
	Duration    time.Duration
	Success     bool
	CacheHits   int64
	CacheMisses int64
	ErrorMsg    string
}

// Metrics contains aggregated statistics over build history
type Metrics struct {
	// Counts
	TotalBuilds  int
	SuccessCount int
	FailureCount int

	// Timing statistics
	AvgDuration  time.Duration
	MinDuration  time.Duration
	MaxDuration  time.Duration
	LastDuration time.Duration

	// Cache statistics
	AvgCacheHitRate float64

	// Trend indicators
	DurationTrend Trend // Getting faster, slower, or stable
	SuccessRate   float64
}

// Trend indicates whether a metric is improving, degrading, or stable
type Trend string

const (
	TrendImproving Trend = "improving" // Getting better
	TrendStable    Trend = "stable"    // No significant change
	TrendDegrading Trend = "degrading" // Getting worse
)

// New creates a new building from a build target
//
// Assumptions:
// - target is valid (from build adapter)
// - buildSystem is one of: "bazel", "npm", "cargo"
// - x, y are valid coordinates on the game map
//
// Edge cases:
// - target.ID is empty -> use target.Name as fallback
// - buildSystem is unknown -> store as-is, display will handle
func New(target build.Target, buildSystem string, x, y float64) *Building {
	id := target.ID
	if id == "" {
		id = target.Name
	}

	return &Building{
		id:           id,
		name:         target.Name,
		targetType:   target.Type,
		buildSystem:  buildSystem,
		x:            x,
		y:            y,
		state:        StateIdle,
		buildHistory: make([]BuildRecord, 0),
		buildQueue:   make([]BuildRequest, 0),
		metrics: Metrics{
			MinDuration: time.Duration(1<<63 - 1), // Max int64
		},
	}
}

// ID returns the building's unique identifier
func (b *Building) ID() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.id
}

// Name returns the building's display name
func (b *Building) Name() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.name
}

// TargetType returns the type of build target
func (b *Building) TargetType() build.TargetType {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.targetType
}

// BuildSystem returns which build system this building uses
func (b *Building) BuildSystem() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buildSystem
}

// Position returns the building's map coordinates
func (b *Building) Position() (x, y float64) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.x, b.y
}

// State returns the current build state
func (b *Building) State() BuildState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// Metrics returns the aggregated build metrics
func (b *Building) Metrics() Metrics {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.metrics
}

// LastBuildResult returns the most recent build result, or nil if never built
func (b *Building) LastBuildResult() *build.Result {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastBuild
}

// RecordBuild records a completed build and updates metrics
//
// Assumptions:
// - result is not nil
// - result.Duration is positive
// - State should be StateBuilding before calling this
//
// Edge cases:
// - Called when not building -> still record, but log warning would be good
// - result.Duration is 0 -> accept it (very fast build)
// - Multiple rapid calls -> all recorded in history
func (b *Building) RecordBuild(result *build.Result) {
	if result == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Update state based on result
	if result.Success {
		b.state = StateSuccess
	} else {
		b.state = StateFailed
	}

	// Store last build
	b.lastBuild = result

	// Add to history
	record := BuildRecord{
		Timestamp:   result.EndTime,
		Duration:    result.Duration,
		Success:     result.Success,
		CacheHits:   result.CacheHits,
		CacheMisses: result.CacheMisses,
		ErrorMsg:    result.ErrorMessage,
	}
	b.buildHistory = append(b.buildHistory, record)

	// Update metrics
	b.updateMetrics()
}

// StartBuild marks the building as currently building
func (b *Building) StartBuild() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateBuilding
}

// SetPosition updates the building's map coordinates
func (b *Building) SetPosition(x, y float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.x = x
	b.y = y
}

// updateMetrics recalculates aggregated metrics from build history
// Must be called with lock held
func (b *Building) updateMetrics() {
	if len(b.buildHistory) == 0 {
		return
	}

	var totalDuration time.Duration
	var totalCacheOps int64
	var totalCacheHits int64
	var successCount int

	minDuration := time.Duration(1<<63 - 1)
	var maxDuration time.Duration

	for _, record := range b.buildHistory {
		// Duration stats
		totalDuration += record.Duration
		if record.Duration < minDuration {
			minDuration = record.Duration
		}
		if record.Duration > maxDuration {
			maxDuration = record.Duration
		}

		// Success count
		if record.Success {
			successCount++
		}

		// Cache stats
		cacheOps := record.CacheHits + record.CacheMisses
		totalCacheOps += cacheOps
		totalCacheHits += record.CacheHits
	}

	historyLen := len(b.buildHistory)

	// Update metrics
	b.metrics.TotalBuilds = historyLen
	b.metrics.SuccessCount = successCount
	b.metrics.FailureCount = historyLen - successCount
	b.metrics.AvgDuration = totalDuration / time.Duration(historyLen)
	b.metrics.MinDuration = minDuration
	b.metrics.MaxDuration = maxDuration
	b.metrics.LastDuration = b.buildHistory[historyLen-1].Duration

	// Success rate
	b.metrics.SuccessRate = float64(successCount) / float64(historyLen) * 100.0

	// Average cache hit rate
	if totalCacheOps > 0 {
		b.metrics.AvgCacheHitRate = float64(totalCacheHits) / float64(totalCacheOps) * 100.0
	}

	// Calculate duration trend (compare last 5 builds to previous builds)
	b.metrics.DurationTrend = b.calculateDurationTrend()
}

// calculateDurationTrend determines if build times are improving, stable, or degrading
// Must be called with lock held
//
// Assumptions:
// - buildHistory has at least 2 entries
// - Recent builds are at the end of the slice
//
// Algorithm:
// - Compare average of last 5 builds to average of previous builds
// - If difference is less than 10%, consider stable
// - If recent builds are faster by >10%, improving
// - If recent builds are slower by >10%, degrading
func (b *Building) calculateDurationTrend() Trend {
	if len(b.buildHistory) < 6 {
		return TrendStable // Not enough data
	}

	// Get last 5 builds
	recentBuilds := b.buildHistory[len(b.buildHistory)-5:]
	var recentTotal time.Duration
	for _, build := range recentBuilds {
		recentTotal += build.Duration
	}
	recentAvg := recentTotal / 5

	// Get previous builds (everything before last 5)
	previousBuilds := b.buildHistory[:len(b.buildHistory)-5]
	var previousTotal time.Duration
	for _, build := range previousBuilds {
		previousTotal += build.Duration
	}
	previousAvg := previousTotal / time.Duration(len(previousBuilds))

	// Compare with 10% threshold
	threshold := float64(previousAvg) * 0.1
	diff := float64(recentAvg - previousAvg)

	if diff < -threshold {
		return TrendImproving // Getting faster
	} else if diff > threshold {
		return TrendDegrading // Getting slower
	}
	return TrendStable
}

// BuildHistory returns a copy of the build history
func (b *Building) BuildHistory() []BuildRecord {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Return a copy to prevent mutation
	history := make([]BuildRecord, len(b.buildHistory))
	copy(history, b.buildHistory)
	return history
}
