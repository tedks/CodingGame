package build

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CargoAdapter implements the Adapter interface for Cargo (Rust) projects
type CargoAdapter struct {
	cargoBin string // Path to cargo executable (default: "cargo")
}

// NewCargoAdapter creates a new Cargo adapter
func NewCargoAdapter() *CargoAdapter {
	return &CargoAdapter{
		cargoBin: "cargo",
	}
}

// Name returns the adapter name
func (c *CargoAdapter) Name() string {
	return "cargo"
}

// Detect checks if Cargo is used in the project by looking for Cargo.toml
//
// Assumptions:
// - projectPath is an absolute path to a directory
// - We have read permissions
//
// Edge cases:
// - Directory doesn't exist -> return false, nil
// - No read permissions -> return false, error
// - Cargo.toml exists but is a directory -> return false, nil
func (c *CargoAdapter) Detect(projectPath string) (bool, error) {
	// Check if directory exists
	info, err := os.Stat(projectPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot access project path: %w", err)
	}

	if !info.IsDir() {
		return false, fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	// Check for Cargo.toml
	cargoTomlPath := filepath.Join(projectPath, "Cargo.toml")
	fileInfo, err := os.Stat(cargoTomlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot access Cargo.toml: %w", err)
	}

	// Ensure it's a file, not a directory
	if fileInfo.IsDir() {
		return false, nil
	}

	return true, nil
}

// GetTargets lists all cargo targets (binaries, libraries, tests)
//
// Assumptions:
// - Cargo.toml exists and is valid
// - cargo is installed and in PATH
//
// Edge cases:
// - cargo not installed -> error
// - Cargo.toml is malformed -> cargo will error, we propagate it
// - No targets defined -> cargo still reports package, we return it
func (c *CargoAdapter) GetTargets(projectPath string) ([]Target, error) {
	// Verify cargo is available
	if err := c.checkCargoInstalled(); err != nil {
		return nil, err
	}

	// Run "cargo metadata" to get structured information about targets
	// This is more reliable than parsing Cargo.toml manually
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.cargoBin, "metadata", "--no-deps", "--format-version=1")
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("cargo metadata timed out after 30s")
		}
		return nil, fmt.Errorf("cargo metadata failed: %w\nOutput: %s", err, string(output))
	}

	// Parse metadata output (it's JSON, but we'll use simple parsing)
	targets, err := c.parseMetadataOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse cargo metadata: %w", err)
	}

	return targets, nil
}

// parseMetadataOutput parses "cargo metadata" JSON output
// We use regex for simplicity since we only need target information
func (c *CargoAdapter) parseMetadataOutput(output string) ([]Target, error) {
	var targets []Target

	// Look for target blocks in the JSON
	// Format: "kind": ["bin"|"lib"|"test"|...]
	kindRe := regexp.MustCompile(`"kind":\s*\[\s*"([^"]+)"`)
	nameRe := regexp.MustCompile(`"name":\s*"([^"]+)"`)

	// Split by target blocks (simple heuristic)
	lines := strings.Split(output, "\n")
	var currentName string
	var currentKind string

	for _, line := range lines {
		// Extract name
		if matches := nameRe.FindStringSubmatch(line); len(matches) > 1 {
			currentName = matches[1]
		}

		// Extract kind
		if matches := kindRe.FindStringSubmatch(line); len(matches) > 1 {
			currentKind = matches[1]

			// If we have both name and kind, create a target
			if currentName != "" && currentKind != "" {
				targetType := c.inferTargetType(currentKind)

				targets = append(targets, Target{
					ID:          currentName,
					Name:        currentName,
					Type:        targetType,
					Description: fmt.Sprintf("cargo %s", currentKind),
				})

				// Reset for next target
				currentName = ""
				currentKind = ""
			}
		}
	}

	return targets, nil
}

// inferTargetType infers TargetType from cargo target kind
func (c *CargoAdapter) inferTargetType(kind string) TargetType {
	switch kind {
	case "bin":
		return TargetTypeBinary
	case "lib", "rlib", "dylib", "cdylib", "staticlib", "proc-macro":
		return TargetTypeLibrary
	case "test", "bench":
		return TargetTypeTest
	default:
		return TargetTypeUnknown
	}
}

// Build executes a cargo build and extracts metrics
//
// Assumptions:
// - cargo is installed
// - Target is valid (or "all" for workspace build)
// - Project dependencies are fetchable
//
// Edge cases:
// - Build fails -> Success=false, error details in output
// - Network failure -> build fails with error
// - Compilation errors -> build fails, errors in output
// - Target doesn't exist -> cargo error
func (c *CargoAdapter) Build(projectPath string, targetID string, opts *BuildOptions) (*Result, error) {
	if opts == nil {
		opts = &BuildOptions{}
	}

	// Set default timeout if not specified
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 15 * time.Minute // Rust builds can be slow
	}

	// Verify cargo is available
	if err := c.checkCargoInstalled(); err != nil {
		return nil, err
	}

	// Prepare cargo command
	args := []string{"build"}

	if opts.Clean {
		// Run clean first
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		cleanCmd := exec.CommandContext(cleanCtx, c.cargoBin, "clean")
		cleanCmd.Dir = projectPath
		_ = cleanCmd.Run() // Ignore clean errors
		cleanCancel()
	}

	if opts.Jobs > 0 {
		args = append(args, fmt.Sprintf("--jobs=%d", opts.Jobs))
	}

	// Add release flag for optimized builds (common pattern)
	// Note: This could be made configurable via BuildOptions
	// args = append(args, "--release")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute build
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, c.cargoBin, args...)
	cmd.Dir = projectPath

	var outputBuf bytes.Buffer
	cmd.Stdout = &outputBuf
	cmd.Stderr = &outputBuf

	err := cmd.Run()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	output := outputBuf.String()

	// Build result
	result := &Result{
		Duration:  duration,
		StartTime: startTime,
		EndTime:   endTime,
		Output:    output,
	}

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("build timed out after %v", timeout)
		return result, fmt.Errorf("build timed out")
	}

	// Extract metrics from cargo output
	c.extractMetrics(output, result)

	// Set success based on exit code
	if err != nil {
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ErrorMessage = fmt.Sprintf("build failed with exit code %d", exitErr.ExitCode())
		} else {
			result.ErrorMessage = fmt.Errorf("build failed: %w", err).Error()
		}
	} else {
		result.Success = true
	}

	return result, nil
}

// extractMetrics parses cargo build output to extract metrics
//
// Cargo output patterns:
// - "Compiling <crate> v<version>"
// - "Finished dev [unoptimized + debuginfo] target(s) in 1.23s"
// - "Fresh <crate> v<version>"
func (c *CargoAdapter) extractMetrics(output string, result *Result) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	compilingRe := regexp.MustCompile(`^\s*Compiling\s+`)
	freshRe := regexp.MustCompile(`^\s*Fresh\s+`)
	finishedRe := regexp.MustCompile(`Finished.*in\s+([\d.]+)s`)

	var compiledCount int64
	var freshCount int64

	for scanner.Scan() {
		line := scanner.Text()

		// Count "Compiling" lines (cache misses)
		if compilingRe.MatchString(line) {
			compiledCount++
		}

		// Count "Fresh" lines (cache hits)
		if freshRe.MatchString(line) {
			freshCount++
		}

		// Extract total build time from "Finished" line
		if matches := finishedRe.FindStringSubmatch(line); len(matches) > 1 {
			if duration, err := strconv.ParseFloat(matches[1], 64); err == nil {
				// Store as duration (already have wall-clock time, but this is cargo's reported time)
				_ = duration // We already have Duration from our timing
			}
		}
	}

	result.CacheHits = freshCount
	result.CacheMisses = compiledCount
	result.TargetsBuilt = int(freshCount + compiledCount)

	// If we didn't find any targets, default to 1 (the crate itself)
	if result.TargetsBuilt == 0 {
		result.TargetsBuilt = 1
	}
}

// checkCargoInstalled verifies that cargo is available in PATH
func (c *CargoAdapter) checkCargoInstalled() error {
	cmd := exec.Command(c.cargoBin, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo not found in PATH (tried %q): %w", c.cargoBin, err)
	}
	return nil
}
