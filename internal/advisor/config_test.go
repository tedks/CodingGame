package advisor

import (
	"strings"
	"testing"
)

func TestConfig_Validate_Valid(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "manual trigger",
			config: Config{
				ID:           "security",
				Name:         "Security Advisor",
				SystemPrompt: "Analyze for security issues",
				Trigger:      TriggerManual,
			},
		},
		{
			name: "on_file_change trigger",
			config: Config{
				ID:            "refactor",
				Name:          "Refactoring Advisor",
				SystemPrompt:  "Find code smells",
				Trigger:       TriggerOnFileChange,
				FocusPatterns: []string{"*.go", "*.ts"},
			},
		},
		{
			name: "background trigger with interval",
			config: Config{
				ID:                     "monitor",
				Name:                   "Monitoring Advisor",
				SystemPrompt:           "Monitor code health",
				Trigger:                TriggerBackground,
				BackgroundIntervalSecs: 300,
			},
		},
		{
			name: "with icon",
			config: Config{
				ID:           "testing",
				Name:         "Testing Advisor",
				Icon:         "check",
				SystemPrompt: "Improve test coverage",
				Trigger:      TriggerManual,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err != nil {
				t.Errorf("Validate() returned error: %v", err)
			}
		})
	}
}

func TestConfig_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError string
	}{
		{
			name:      "missing id",
			config:    Config{Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual},
			wantError: "id is required",
		},
		{
			name:      "missing name",
			config:    Config{ID: "test", SystemPrompt: "prompt", Trigger: TriggerManual},
			wantError: "name is required",
		},
		{
			name:      "missing system_prompt",
			config:    Config{ID: "test", Name: "Test", Trigger: TriggerManual},
			wantError: "system_prompt is required",
		},
		{
			name:      "missing trigger",
			config:    Config{ID: "test", Name: "Test", SystemPrompt: "prompt"},
			wantError: "trigger is required",
		},
		{
			name: "invalid trigger",
			config: Config{
				ID:           "test",
				Name:         "Test",
				SystemPrompt: "prompt",
				Trigger:      "invalid",
			},
			wantError: "invalid trigger",
		},
		{
			name: "background without interval",
			config: Config{
				ID:           "test",
				Name:         "Test",
				SystemPrompt: "prompt",
				Trigger:      TriggerBackground,
			},
			wantError: "background_interval_secs must be > 0",
		},
		{
			name: "background with zero interval",
			config: Config{
				ID:                     "test",
				Name:                   "Test",
				SystemPrompt:           "prompt",
				Trigger:                TriggerBackground,
				BackgroundIntervalSecs: 0,
			},
			wantError: "background_interval_secs must be > 0",
		},
		{
			name: "background with negative interval",
			config: Config{
				ID:                     "test",
				Name:                   "Test",
				SystemPrompt:           "prompt",
				Trigger:                TriggerBackground,
				BackgroundIntervalSecs: -1,
			},
			wantError: "background_interval_secs must be > 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Error("Validate() expected error, got nil")
				return
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Validate() error = %q, want error containing %q", err, tt.wantError)
			}
		})
	}
}

func TestConfig_MatchesFile(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		filePath string
		want     bool
	}{
		{
			name:     "empty patterns matches everything",
			patterns: nil,
			filePath: "src/main.go",
			want:     true,
		},
		{
			name:     "exact match",
			patterns: []string{"main.go"},
			filePath: "main.go",
			want:     true,
		},
		{
			name:     "wildcard extension",
			patterns: []string{"*.go"},
			filePath: "main.go",
			want:     true,
		},
		{
			name:     "no match different extension",
			patterns: []string{"*.ts"},
			filePath: "main.go",
			want:     false,
		},
		{
			name:     "matches base name for simple pattern",
			patterns: []string{"*.go"},
			filePath: "src/internal/main.go",
			want:     true,
		},
		{
			name:     "multiple patterns first matches",
			patterns: []string{"*.go", "*.ts"},
			filePath: "app.go",
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			patterns: []string{"*.go", "*.ts"},
			filePath: "app.ts",
			want:     true,
		},
		{
			name:     "multiple patterns none match",
			patterns: []string{"*.go", "*.ts"},
			filePath: "app.py",
			want:     false,
		},
		{
			name:     "pattern with directory",
			patterns: []string{"src/*.go"},
			filePath: "src/main.go",
			want:     true,
		},
		{
			name:     "pattern with directory no match",
			patterns: []string{"src/*.go"},
			filePath: "lib/main.go",
			want:     false,
		},
		{
			name:     "double star directory match",
			patterns: []string{"**/auth/**"},
			filePath: "src/auth/login.go",
			want:     true,
		},
		{
			name:     "double star extension match",
			patterns: []string{"**/*.go"},
			filePath: "src/internal/main.go",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				ID:            "test",
				Name:          "Test",
				FocusPatterns: tt.patterns,
			}
			got := config.MatchesFile(tt.filePath)
			if got != tt.want {
				t.Errorf("MatchesFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}

func TestParseConfig_Valid(t *testing.T) {
	json := `{
		"advisors": [
			{
				"id": "security",
				"name": "Security Advisor",
				"icon": "shield",
				"system_prompt": "Analyze for security vulnerabilities",
				"trigger": "on_file_change",
				"focus_patterns": ["**/auth/**", "**/*.env"]
			},
			{
				"id": "refactoring",
				"name": "Refactoring Advisor",
				"icon": "wrench",
				"system_prompt": "Find code smells",
				"trigger": "manual"
			}
		]
	}`

	configs, err := ParseConfig(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(configs) != 2 {
		t.Errorf("ParseConfig() returned %d configs, want 2", len(configs))
	}

	// Check first config
	if configs[0].ID != "security" {
		t.Errorf("configs[0].ID = %q, want security", configs[0].ID)
	}
	if configs[0].Trigger != TriggerOnFileChange {
		t.Errorf("configs[0].Trigger = %q, want on_file_change", configs[0].Trigger)
	}
	if len(configs[0].FocusPatterns) != 2 {
		t.Errorf("configs[0].FocusPatterns length = %d, want 2", len(configs[0].FocusPatterns))
	}

	// Check second config
	if configs[1].ID != "refactoring" {
		t.Errorf("configs[1].ID = %q, want refactoring", configs[1].ID)
	}
	if configs[1].Trigger != TriggerManual {
		t.Errorf("configs[1].Trigger = %q, want manual", configs[1].Trigger)
	}
}

func TestParseConfig_EmptyAdvisors(t *testing.T) {
	json := `{"advisors": []}`

	configs, err := ParseConfig(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(configs) != 0 {
		t.Errorf("ParseConfig() returned %d configs, want 0", len(configs))
	}
}

func TestParseConfig_InvalidJSON(t *testing.T) {
	json := `not valid json`

	_, err := ParseConfig(strings.NewReader(json))
	if err == nil {
		t.Error("ParseConfig() expected error for invalid JSON")
	}
}

func TestParseConfig_InvalidConfig(t *testing.T) {
	// Missing required field
	json := `{
		"advisors": [
			{
				"id": "test",
				"trigger": "manual"
			}
		]
	}`

	_, err := ParseConfig(strings.NewReader(json))
	if err == nil {
		t.Error("ParseConfig() expected error for invalid config")
		return
	}
	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("ParseConfig() error = %q, expected to mention 'name is required'", err)
	}
}

func TestParseConfig_DuplicateIDs(t *testing.T) {
	json := `{
		"advisors": [
			{
				"id": "duplicate",
				"name": "First",
				"system_prompt": "prompt1",
				"trigger": "manual"
			},
			{
				"id": "duplicate",
				"name": "Second",
				"system_prompt": "prompt2",
				"trigger": "manual"
			}
		]
	}`

	_, err := ParseConfig(strings.NewReader(json))
	if err == nil {
		t.Error("ParseConfig() expected error for duplicate IDs")
		return
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("ParseConfig() error = %q, expected to mention 'duplicate'", err)
	}
}

func TestParseConfig_BackgroundTrigger(t *testing.T) {
	json := `{
		"advisors": [
			{
				"id": "monitor",
				"name": "Monitor",
				"system_prompt": "Monitor health",
				"trigger": "background",
				"background_interval_secs": 300
			}
		]
	}`

	configs, err := ParseConfig(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if len(configs) != 1 {
		t.Fatalf("ParseConfig() returned %d configs, want 1", len(configs))
	}

	if configs[0].BackgroundIntervalSecs != 300 {
		t.Errorf("BackgroundIntervalSecs = %d, want 300", configs[0].BackgroundIntervalSecs)
	}
}

func TestDefaultConfigs(t *testing.T) {
	configs := DefaultConfigs()

	if len(configs) == 0 {
		t.Fatal("DefaultConfigs() returned empty slice")
	}

	// All default configs should be valid
	for i, cfg := range configs {
		if err := cfg.Validate(); err != nil {
			t.Errorf("DefaultConfigs()[%d] validation failed: %v", i, err)
		}
	}

	// Check for required advisors
	ids := make(map[string]bool)
	for _, cfg := range configs {
		ids[cfg.ID] = true
	}

	requiredIDs := []string{"security", "refactoring", "testing"}
	for _, id := range requiredIDs {
		if !ids[id] {
			t.Errorf("DefaultConfigs() missing required advisor %q", id)
		}
	}
}

func TestTriggerMode_Constants(t *testing.T) {
	// Verify TriggerMode constants are distinct and match expected values
	modes := map[TriggerMode]string{
		TriggerManual:       "manual",
		TriggerOnFileChange: "on_file_change",
		TriggerBackground:   "background",
	}

	for mode, expected := range modes {
		if string(mode) != expected {
			t.Errorf("TriggerMode %v = %q, want %q", mode, string(mode), expected)
		}
	}

	// Verify distinctness
	uniqueModes := make(map[TriggerMode]bool)
	uniqueModes[TriggerManual] = true
	uniqueModes[TriggerOnFileChange] = true
	uniqueModes[TriggerBackground] = true

	if len(uniqueModes) != 3 {
		t.Error("TriggerMode constants are not distinct")
	}
}

func TestContainsPathSeparator(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"*.go", false},
		{"src/*.go", true},
		{"a/b/c", true},
		{"", false},
		{"file.txt", false},
	}

	for _, tt := range tests {
		got := containsPathSeparator(tt.input)
		if got != tt.want {
			t.Errorf("containsPathSeparator(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
