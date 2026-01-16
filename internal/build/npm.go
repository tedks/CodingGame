package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NpmAdapter implements the Adapter interface for npm (Node.js) projects
type NpmAdapter struct {
	npmBin string // Path to npm executable (default: "npm")
}

// NewNpmAdapter creates a new npm adapter
func NewNpmAdapter() *NpmAdapter {
	return &NpmAdapter{
		npmBin: "npm",
	}
}

// Name returns the adapter name
func (n *NpmAdapter) Name() string {
	return "npm"
}

// Detect checks if npm is used in the project by looking for package.json
//
// Assumptions:
// - projectPath is an absolute path to a directory
// - We have read permissions
//
// Edge cases:
// - Directory doesn't exist -> return false, nil
// - No read permissions -> return false, error
// - package.json exists but is a directory -> return false, nil
func (n *NpmAdapter) Detect(projectPath string) (bool, error) {
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

	// Check for package.json
	packageJSONPath := filepath.Join(projectPath, "package.json")
	fileInfo, err := os.Stat(packageJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("cannot access package.json: %w", err)
	}

	// Ensure it's a file, not a directory
	if fileInfo.IsDir() {
		return false, nil
	}

	return true, nil
}

// packageJSON represents the structure of package.json (partial)
type packageJSON struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

// GetTargets lists all npm scripts defined in package.json
//
// Assumptions:
// - package.json exists (Detect returned true)
// - package.json is valid JSON
//
// Edge cases:
// - package.json is malformed -> error
// - No scripts defined -> empty slice, no error
// - Scripts field is null -> empty slice, no error
func (n *NpmAdapter) GetTargets(projectPath string) ([]Target, error) {
	// Read package.json
	packageJSONPath := filepath.Join(projectPath, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	// Parse package.json
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	// Convert scripts to targets
	var targets []Target
	for scriptName, scriptCommand := range pkg.Scripts {
		// Infer target type from script name and command
		targetType := n.inferTargetType(scriptName, scriptCommand)

		targets = append(targets, Target{
			ID:          scriptName,
			Name:        scriptName,
			Type:        targetType,
			Description: fmt.Sprintf("npm script: %s", scriptCommand),
		})
	}

	return targets, nil
}

// inferTargetType infers target type from script name and command
func (n *NpmAdapter) inferTargetType(scriptName, scriptCommand string) TargetType {
	name := strings.ToLower(scriptName)
	command := strings.ToLower(scriptCommand)

	// Check for test scripts
	if strings.Contains(name, "test") || strings.Contains(command, "jest") ||
		strings.Contains(command, "mocha") || strings.Contains(command, "vitest") ||
		strings.Contains(command, "jasmine") {
		return TargetTypeTest
	}

	// Check for build scripts
	if strings.Contains(name, "build") || strings.Contains(name, "compile") ||
		strings.Contains(command, "webpack") || strings.Contains(command, "tsc") ||
		strings.Contains(command, "vite build") || strings.Contains(command, "esbuild") {
		return TargetTypeBinary
	}

	// Development/utility scripts
	return TargetTypeUnknown
}

// Build executes an npm script and returns metrics
//
// Assumptions:
// - npm is installed and in PATH
// - Target (script) exists in package.json
// - node_modules are installed
//
// Edge cases:
// - Script doesn't exist -> error with helpful message
// - npm not installed -> error immediately
// - Script fails -> Success=false, error message from npm
// - Script times out -> error after timeout
// - node_modules missing -> script may fail, captured in output
func (n *NpmAdapter) Build(projectPath string, targetID string, opts *BuildOptions) (*Result, error) {
	if opts == nil {
		opts = &BuildOptions{}
	}

	// Set default timeout if not specified
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute // Default 10 minute timeout
	}

	// Verify npm is available
	if err := n.checkNpmInstalled(); err != nil {
		return nil, err
	}

	// Prepare npm command
	// We use "npm run <script>" for all scripts
	args := []string{"run", targetID}

	// npm doesn't have built-in parallelism flags like Bazel
	// Jobs option is ignored for npm

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute build
	startTime := time.Now()
	cmd := exec.CommandContext(ctx, n.npmBin, args...)
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
		// npm doesn't provide cache statistics by default
		CacheHits:   0,
		CacheMisses: 0,
		// Count as 1 target (the script)
		TargetsBuilt: 1,
	}

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		result.Success = false
		result.ErrorMessage = fmt.Sprintf("script timed out after %v", timeout)
		return result, fmt.Errorf("script timed out")
	}

	// Set success based on exit code
	if err != nil {
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ErrorMessage = fmt.Sprintf("script failed with exit code %d", exitErr.ExitCode())
		} else {
			result.ErrorMessage = fmt.Sprintf("script failed: %v", err)
		}

		// Check if script doesn't exist
		if strings.Contains(output, "missing script") || strings.Contains(output, "Unknown script") {
			result.ErrorMessage = fmt.Sprintf("script %q not found in package.json", targetID)
		}
	} else {
		result.Success = true
	}

	return result, nil
}

// checkNpmInstalled verifies that npm is available in PATH
func (n *NpmAdapter) checkNpmInstalled() error {
	cmd := exec.Command(n.npmBin, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm not found in PATH (tried %q): %w", n.npmBin, err)
	}
	return nil
}
