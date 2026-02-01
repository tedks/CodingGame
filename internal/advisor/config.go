// Package advisor provides management of advisor subagents - specialized Claude
// instances with focused contexts that provide domain-specific insights.
//
// Advisors are configured via JSON and can:
// - Load and parse advisor configurations
// - Run in different trigger modes (manual, on_file_change, background)
// - Generate insights based on code analysis
// - Be consulted interactively for domain-specific questions
//
// This is NOT a gamification system - advisors are real Claude subagents
// with real token consumption and analysis capabilities.
package advisor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// TriggerMode defines when an advisor runs
type TriggerMode string

const (
	// TriggerManual runs only when user explicitly consults the advisor
	TriggerManual TriggerMode = "manual"
	// TriggerOnFileChange runs when main agent modifies matching files
	TriggerOnFileChange TriggerMode = "on_file_change"
	// TriggerBackground runs periodically in the background
	TriggerBackground TriggerMode = "background"
)

// Config represents a single advisor's configuration
type Config struct {
	// ID is the unique identifier for this advisor
	ID string `json:"id"`
	// Name is the display name shown in the UI
	Name string `json:"name"`
	// Icon is the icon identifier (e.g., "wrench", "shield")
	Icon string `json:"icon"`
	// SystemPrompt defines the advisor's expertise and behavior
	SystemPrompt string `json:"system_prompt"`
	// Trigger defines when the advisor runs
	Trigger TriggerMode `json:"trigger"`
	// FocusPatterns are glob patterns for files this advisor cares about
	FocusPatterns []string `json:"focus_patterns"`
	// BackgroundIntervalSecs is the interval for background trigger (if applicable)
	BackgroundIntervalSecs int `json:"background_interval_secs,omitempty"`

	// HarnessName specifies which harness to use for this advisor.
	// If empty, uses the same harness as the main agent.
	// This allows mixing advisors from different providers (e.g., a Claude advisor
	// in a Codex project, or a specialized model advisor).
	//
	// Example: Use different models for different advisors:
	//
	//     // Security advisor uses Opus for thorough analysis
	//     Config{
	//         ID:           "security",
	//         HarnessName:  "claude-code",
	//         HarnessModel: "opus",
	//     }
	//
	//     // Linting advisor uses Haiku for quick feedback
	//     Config{
	//         ID:           "linter",
	//         HarnessName:  "claude-code",
	//         HarnessModel: "haiku",
	//     }
	HarnessName string `json:"harness_name,omitempty"`

	// HarnessModel specifies which model to use for this advisor.
	// If empty, uses the harness's default model.
	// This allows using different model tiers for different advisors (e.g., opus
	// for security analysis, haiku for quick linting).
	HarnessModel string `json:"harness_model,omitempty"`
}

// ConfigFile represents the root structure of an advisor configuration file
type ConfigFile struct {
	Advisors []Config `json:"advisors"`
}

// Validate checks if the config is valid
//
// Returns an error if:
// - ID is empty
// - Name is empty
// - SystemPrompt is empty
// - Trigger is invalid
// - BackgroundIntervalSecs is <= 0 for background trigger
func (c *Config) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("advisor config: id is required")
	}
	if c.Name == "" {
		return fmt.Errorf("advisor config %q: name is required", c.ID)
	}
	if c.SystemPrompt == "" {
		return fmt.Errorf("advisor config %q: system_prompt is required", c.ID)
	}

	// Validate trigger mode
	switch c.Trigger {
	case TriggerManual, TriggerOnFileChange, TriggerBackground:
		// Valid
	case "":
		return fmt.Errorf("advisor config %q: trigger is required", c.ID)
	default:
		return fmt.Errorf("advisor config %q: invalid trigger %q (must be manual, on_file_change, or background)", c.ID, c.Trigger)
	}

	// Background trigger requires interval
	if c.Trigger == TriggerBackground && c.BackgroundIntervalSecs <= 0 {
		return fmt.Errorf("advisor config %q: background_interval_secs must be > 0 for background trigger", c.ID)
	}

	return nil
}

// MatchesFile checks if the given file path matches any of the advisor's focus patterns
//
// Assumptions:
// - filePath is a clean, normalized path
// - FocusPatterns contains valid glob patterns
//
// Edge cases:
// - Empty FocusPatterns returns true (advisor watches everything)
// - Invalid glob patterns are skipped (logged but not matched)
func (c *Config) MatchesFile(filePath string) bool {
	// No patterns means match everything
	if len(c.FocusPatterns) == 0 {
		return true
	}

	slashPath := filepath.ToSlash(filePath)
	baseName := filepath.Base(filePath)
	for _, pattern := range c.FocusPatterns {
		slashPattern := filepath.ToSlash(pattern)
		matched, err := doublestar.Match(slashPattern, slashPath)
		if err != nil {
			// Invalid pattern, skip it
			continue
		}
		if matched {
			return true
		}

		// Also try matching against the base name for patterns without path separators
		if !containsPathSeparator(pattern) {
			matched, err = doublestar.Match(slashPattern, baseName)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// containsPathSeparator checks if a string contains a path separator
func containsPathSeparator(s string) bool {
	for _, r := range s {
		if r == '/' || r == filepath.Separator {
			return true
		}
	}
	return false
}

// LoadConfig loads advisor configuration from a JSON file
//
// Assumptions:
// - path points to a readable JSON file
// - JSON structure matches ConfigFile
//
// Edge cases:
// - File doesn't exist -> returns error
// - Invalid JSON -> returns error
// - Empty advisors array -> returns empty slice (valid)
// - Invalid advisor configs -> returns error for each
func LoadConfig(path string) ([]Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open advisor config: %w", err)
	}
	defer file.Close()

	return ParseConfig(file)
}

// ParseConfig parses advisor configuration from a reader
//
// This is useful for testing and for loading from embedded configs
func ParseConfig(reader io.Reader) ([]Config, error) {
	var configFile ConfigFile

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&configFile); err != nil {
		return nil, fmt.Errorf("parse advisor config: %w", err)
	}

	// Validate each advisor config
	for i, cfg := range configFile.Advisors {
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("advisor config [%d]: %w", i, err)
		}
	}

	// Check for duplicate IDs
	seen := make(map[string]bool)
	for _, cfg := range configFile.Advisors {
		if seen[cfg.ID] {
			return nil, fmt.Errorf("duplicate advisor id: %q", cfg.ID)
		}
		seen[cfg.ID] = true
	}

	return configFile.Advisors, nil
}

// DefaultConfigs returns a set of default advisor configurations
// These serve as examples and can be customized by users
func DefaultConfigs() []Config {
	return []Config{
		{
			ID:           "security",
			Name:         "Security Advisor",
			Icon:         "shield",
			SystemPrompt: "You analyze code for security vulnerabilities including injection attacks, authentication issues, secrets exposure, and OWASP top 10 risks.",
			Trigger:      TriggerOnFileChange,
			FocusPatterns: []string{
				"**/auth/**",
				"**/api/**",
				"**/*.env",
				"**/credentials*",
			},
		},
		{
			ID:           "refactoring",
			Name:         "Refactoring Advisor",
			Icon:         "wrench",
			SystemPrompt: "You identify code smells, suggest design improvements, and recommend refactoring opportunities to improve code maintainability.",
			Trigger:      TriggerManual,
			FocusPatterns: []string{
				"**/*.go",
				"**/*.ts",
				"**/*.rs",
			},
		},
		{
			ID:           "testing",
			Name:         "Testing Advisor",
			Icon:         "check",
			SystemPrompt: "You analyze test coverage, identify missing test cases, suggest improvements to test quality, and help diagnose flaky tests.",
			Trigger:      TriggerOnFileChange,
			FocusPatterns: []string{
				"**/*_test.go",
				"**/*.test.ts",
				"**/*_test.rs",
			},
		},
	}
}
