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

// BazelAdapter implements the Adapter interface for Bazel build system
type BazelAdapter struct {
	// bazelBin is the path to the bazel executable (default: "bazel")
	bazelBin string
}

// NewBazelAdapter creates a new Bazel adapter
func NewBazelAdapter() *BazelAdapter {
	return &BazelAdapter{
		bazelBin: "bazel",
	}
}

// Name returns the adapter name
func (b *BazelAdapter) Name() string {
	return "bazel"
}

// Detect checks if Bazel is used in the project
//
// Assumptions:
// - projectPath is an absolute path to a directory
// - We have read permissions
//
// Edge cases:
// - Directory doesn't exist -> return false, nil
// - No read permissions -> return false, error
// - WORKSPACE exists but is not a file -> return false, nil
func (b *BazelAdapter) Detect(projectPath string) (bool, error) {
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

	// Check for WORKSPACE or WORKSPACE.bazel
	workspaceFiles := []string{"WORKSPACE", "WORKSPACE.bazel"}
	for _, filename := range workspaceFiles {
		path := filepath.Join(projectPath, filename)
		if fileInfo, err := os.Stat(path); err == nil && !fileInfo.IsDir() {
			return true, nil
		}
	}

	return false, nil
}

// GetTargets lists all Bazel targets in the project
//
// Assumptions:
// - Bazel is installed and in PATH
// - Project has valid WORKSPACE
// - bazel query command works
//
// Edge cases:
// - Bazel not in PATH -> error
// - Query fails (syntax error, etc.) -> error with output
// - No targets found -> empty slice, no error
// - Query times out -> error after timeout
func (b *BazelAdapter) GetTargets(projectPath string) ([]Target, error) {
	// Verify bazel is available
	if err := b.checkBazelInstalled(); err != nil {
		return nil, err
	}

	// Run bazel query to get all targets
	// Using --output=label_kind to get target type information
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, b.bazelBin, "query", "--output=label_kind", "//...")
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Query failed - check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("bazel query timed out after 30s")
		}
		return nil, fmt.Errorf("bazel query failed: %w\nOutput: %s", err, string(output))
	}

	// Parse query output
	targets, err := b.parseQueryOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse bazel query output: %w", err)
	}

	return targets, nil
}

// parseQueryOutput parses the output of "bazel query --output=label_kind"
//
// Format: "<rule_type> rule <target_label>"
// Example: "go_binary rule //cmd/foo:foo"
func (b *BazelAdapter) parseQueryOutput(output string) ([]Target, error) {
	var targets []Target
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex to parse "rule_type rule //target:name"
	re := regexp.MustCompile(`^(\S+)\s+rule\s+(.+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) != 3 {
			// Line doesn't match expected format, skip it
			continue
		}

		ruleType := matches[1]
		targetLabel := matches[2]

		// Extract display name from target label
		// "//cmd/foo:foo" -> "foo"
		name := targetLabel
		if idx := strings.LastIndex(targetLabel, ":"); idx != -1 {
			name = targetLabel[idx+1:]
		} else if idx := strings.LastIndex(targetLabel, "/"); idx != -1 {
			name = targetLabel[idx+1:]
		}

		// Determine target type from rule type
		targetType := b.inferTargetType(ruleType)

		targets = append(targets, Target{
			ID:          targetLabel,
			Name:        name,
			Type:        targetType,
			Description: fmt.Sprintf("%s target", ruleType),
			// Dependencies will be populated separately if needed
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading query output: %w", err)
	}

	return targets, nil
}

// inferTargetType infers TargetType from Bazel rule type
func (b *BazelAdapter) inferTargetType(ruleType string) TargetType {
	ruleType = strings.ToLower(ruleType)

	if strings.Contains(ruleType, "test") {
		return TargetTypeTest
	}
	if strings.Contains(ruleType, "binary") {
		return TargetTypeBinary
	}
	if strings.Contains(ruleType, "library") {
		return TargetTypeLibrary
	}

	// Heuristics for common rules
	switch {
	case strings.HasSuffix(ruleType, "_test"):
		return TargetTypeTest
	case strings.HasSuffix(ruleType, "_binary"):
		return TargetTypeBinary
	case strings.HasSuffix(ruleType, "_library"):
		return TargetTypeLibrary
	default:
		return TargetTypeUnknown
	}
}

// Build executes a Bazel build and extracts metrics
//
// Assumptions:
// - Target is valid (from GetTargets)
// - Bazel is available
// - Project is buildable
//
// Edge cases:
// - Build fails -> Success=false, ErrorMessage populated
// - Build times out -> error
// - Can't parse build output -> return partial metrics
// - Network failure (for external deps) -> build fails with error
func (b *BazelAdapter) Build(projectPath string, targetID string, opts *BuildOptions) (*Result, error) {
	if opts == nil {
		opts = &BuildOptions{}
	}

	// Set default timeout if not specified
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute // Default 10 minute timeout
	}

	// Verify bazel is available
	if err := b.checkBazelInstalled(); err != nil {
		return nil, err
	}

	// Prepare build command
	args := []string{"build"}

	if opts.Clean {
		// Run clean first
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 30*time.Second)
		cleanCmd := exec.CommandContext(cleanCtx, b.bazelBin, "clean")
		cleanCmd.Dir = projectPath
		_ = cleanCmd.Run() // Ignore clean errors
		cleanCancel()
	}

	if opts.Jobs > 0 {
		args = append(args, fmt.Sprintf("--jobs=%d", opts.Jobs))
	}

	// Add target
	args = append(args, targetID)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute build
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, b.bazelBin, args...)
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

	// Parse build output for metrics
	b.extractMetrics(output, result)

	// Set success based on exit code
	if err != nil {
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ErrorMessage = fmt.Sprintf("build failed with exit code %d", exitErr.ExitCode())
		} else {
			result.ErrorMessage = fmt.Sprintf("build failed: %v", err)
		}
	} else {
		result.Success = true
	}

	return result, nil
}

// extractMetrics parses Bazel build output to extract metrics
//
// Bazel output patterns we look for:
// - "INFO: Build completed successfully, N total actions"
// - "INFO: ... remote cache hit, ... processes"
// - "[N / M] Still waiting for N actions"
func (b *BazelAdapter) extractMetrics(output string, result *Result) {
	scanner := bufio.NewScanner(strings.NewReader(output))

	// Regex patterns for extracting metrics
	totalActionsRe := regexp.MustCompile(`(\d+)\s+total actions?`)
	cacheHitRe := regexp.MustCompile(`(\d+)\s+(?:(?:remote|local)\s+)?cache\s+hits?`)
	processesRe := regexp.MustCompile(`(\d+)\s+processes?`)

	for scanner.Scan() {
		line := scanner.Text()

		// Extract total actions built
		if matches := totalActionsRe.FindStringSubmatch(line); len(matches) > 1 {
			if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				result.TargetsBuilt = int(count)
			}
		}

		// Extract cache hits
		if matches := cacheHitRe.FindStringSubmatch(line); len(matches) > 1 {
			if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				result.CacheHits += count
			}
		}

		// Extract processes (as cache misses - things that were actually built)
		if matches := processesRe.FindStringSubmatch(line); len(matches) > 1 {
			if count, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
				result.CacheMisses += count
			}
		}
	}

	// If we got total actions but no cache breakdown, assume all were cache misses
	if result.TargetsBuilt > 0 && result.CacheHits == 0 && result.CacheMisses == 0 {
		result.CacheMisses = int64(result.TargetsBuilt)
	}
}

// checkBazelInstalled verifies that bazel is available in PATH
func (b *BazelAdapter) checkBazelInstalled() error {
	cmd := exec.Command(b.bazelBin, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("bazel not found in PATH (tried %q): %w", b.bazelBin, err)
	}
	return nil
}
