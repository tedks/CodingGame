// Package build provides adapters for various build systems to extract real
// build metrics. It implements a common interface that works across npm, Bazel,
// cargo, and other build tools.
//
// The build adapters extract actual metrics rather than synthetic values:
// - Build duration (real wall-clock time)
// - Cache hit rates (from build system statistics)
// - Success/failure status (exit codes and output parsing)
// - Target dependencies (from build graph)
//
// Each adapter handles the specifics of its build system while providing a
// consistent interface for the game visualization layer.
package build

import (
	"fmt"
	"time"
)

// Adapter is the interface that all build system adapters must implement.
// It provides a common abstraction for detecting, querying, and executing
// builds across different build systems.
type Adapter interface {
	// Name returns the name of this build system (e.g., "bazel", "npm", "cargo")
	Name() string

	// Detect checks if this build system is present in the given project.
	// Returns true if the build system's configuration files are found.
	//
	// Assumptions:
	// - projectPath is an absolute path to a valid directory
	// - We have read permissions on the directory
	//
	// Edge cases:
	// - projectPath doesn't exist -> return false
	// - No read permissions -> return false
	// - Multiple build systems present -> each returns true independently
	Detect(projectPath string) (bool, error)

	// GetTargets lists all available build targets in the project.
	//
	// Assumptions:
	// - Detect() has returned true for this project
	// - Build system is properly configured
	//
	// Edge cases:
	// - No targets defined -> return empty slice, no error
	// - Build system not installed -> return error
	// - Malformed configuration -> return error with details
	GetTargets(projectPath string) ([]Target, error)

	// Build executes a build for the given target and returns detailed metrics.
	//
	// Assumptions:
	// - Target exists (from GetTargets)
	// - Build system is available in PATH
	// - Project is in a buildable state
	//
	// Edge cases:
	// - Build fails -> Result.Success = false, error describes why
	// - Build times out -> error after reasonable duration
	// - Build system not found -> error immediately
	// - Invalid target -> error with suggestion if possible
	// - Network failure (for dependency fetching) -> error with details
	Build(projectPath string, targetID string, opts *BuildOptions) (*Result, error)
}

// Target represents a buildable target (e.g., a Bazel target, npm script, or cargo package)
type Target struct {
	// ID is the unique identifier for this target within the build system.
	// Format is build-system specific (e.g., "//cmd/foo:foo" for Bazel)
	ID string

	// Name is the human-readable display name
	Name string

	// Type categorizes the target (binary, library, test, etc.)
	Type TargetType

	// Description provides optional human-readable documentation
	Description string

	// Dependencies lists the IDs of targets this target depends on.
	// This forms the build graph used for visualization.
	Dependencies []string

	// Tags are optional labels for filtering/grouping (e.g., "integration-test")
	Tags []string
}

// TargetType categorizes build targets
type TargetType string

const (
	TargetTypeBinary  TargetType = "binary"  // Executable
	TargetTypeLibrary TargetType = "library" // Library/package
	TargetTypeTest    TargetType = "test"    // Test target
	TargetTypeUnknown TargetType = "unknown" // Could not determine type
)

// BuildOptions configures how a build is executed
type BuildOptions struct {
	// Timeout specifies maximum duration for the build.
	// Zero means use adapter's default timeout.
	Timeout time.Duration

	// Clean forces a clean build (no cache)
	Clean bool

	// Jobs specifies parallelism level (0 = adapter default)
	Jobs int
}

// Result contains detailed metrics from a build execution
type Result struct {
	// Success indicates if the build completed without errors
	Success bool

	// Duration is the wall-clock time taken for the build
	Duration time.Duration

	// CacheHits is the number of cached artifacts reused (if supported)
	CacheHits int64

	// CacheMisses is the number of artifacts built from scratch
	CacheMisses int64

	// TargetsBuilt is the total number of targets processed
	TargetsBuilt int

	// Output captures stdout+stderr from the build command
	// Useful for error diagnosis when Success = false
	Output string

	// StartTime is when the build began
	StartTime time.Time

	// EndTime is when the build completed (successfully or not)
	EndTime time.Time

	// ErrorMessage contains human-readable error description if Success = false
	ErrorMessage string
}

// CacheHitRate returns the percentage of cache hits (0-100)
// Returns 0 if there were no cache operations
func (r *Result) CacheHitRate() float64 {
	total := r.CacheHits + r.CacheMisses
	if total == 0 {
		return 0
	}
	return float64(r.CacheHits) / float64(total) * 100.0
}

// Registry manages available build adapters
type Registry struct {
	adapters []Adapter
}

// NewRegistry creates a new adapter registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make([]Adapter, 0),
	}
}

// Register adds a build adapter to the registry
func (r *Registry) Register(adapter Adapter) {
	r.adapters = append(r.adapters, adapter)
}

// Detect finds all build systems present in the given project
//
// Returns:
// - Slice of detected adapters (may be empty if no build systems found)
// - Error if directory access fails (nil for "no build systems found")
func (r *Registry) Detect(projectPath string) ([]Adapter, error) {
	var detected []Adapter

	for _, adapter := range r.adapters {
		present, err := adapter.Detect(projectPath)
		if err != nil {
			// Log error but continue checking other adapters
			// This allows partial success if one adapter fails
			continue
		}
		if present {
			detected = append(detected, adapter)
		}
	}

	return detected, nil
}

// GetAdapter returns a specific adapter by name, or error if not found
func (r *Registry) GetAdapter(name string) (Adapter, error) {
	for _, adapter := range r.adapters {
		if adapter.Name() == name {
			return adapter, nil
		}
	}
	return nil, fmt.Errorf("build adapter %q not found", name)
}
