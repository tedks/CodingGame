package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNpmAdapter_Name(t *testing.T) {
	adapter := NewNpmAdapter()
	if adapter.Name() != "npm" {
		t.Errorf("Name() = %q, want %q", adapter.Name(), "npm")
	}
}

func TestNpmAdapter_Detect(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setupFunc   func() string
		wantPresent bool
		wantErr     bool
	}{
		{
			name: "package.json exists",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test1")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
				return dir
			},
			wantPresent: true,
			wantErr:     false,
		},
		{
			name: "package.json missing",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test2")
				os.MkdirAll(dir, 0755)
				return dir
			},
			wantPresent: false,
			wantErr:     false,
		},
		{
			name: "package.json is a directory",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test3")
				os.MkdirAll(dir, 0755)
				os.MkdirAll(filepath.Join(dir, "package.json"), 0755)
				return dir
			},
			wantPresent: false,
			wantErr:     false,
		},
		{
			name: "directory doesn't exist",
			setupFunc: func() string {
				return filepath.Join(tmpDir, "nonexistent")
			},
			wantPresent: false,
			wantErr:     false,
		},
		{
			name: "path is a file, not directory",
			setupFunc: func() string {
				file := filepath.Join(tmpDir, "testfile")
				os.WriteFile(file, []byte(""), 0644)
				return file
			},
			wantPresent: false,
			wantErr:     true,
		},
	}

	adapter := NewNpmAdapter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc()
			present, err := adapter.Detect(path)

			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if present != tt.wantPresent {
				t.Errorf("Detect() = %v, want %v", present, tt.wantPresent)
			}
		})
	}
}

func TestNpmAdapter_GetTargets(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		packageJSON  string
		wantLen      int
		wantErr      bool
		checkTargets func(*testing.T, []Target)
	}{
		{
			name: "empty package.json",
			packageJSON: `{
				"name": "test-package"
			}`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "no scripts field",
			packageJSON: `{
				"name": "test-package",
				"version": "1.0.0"
			}`,
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "single script",
			packageJSON: `{
				"name": "test-package",
				"scripts": {
					"build": "webpack"
				}
			}`,
			wantLen: 1,
			wantErr: false,
			checkTargets: func(t *testing.T, targets []Target) {
				if targets[0].Name != "build" {
					t.Errorf("Target name = %q, want 'build'", targets[0].Name)
				}
				if targets[0].Type != TargetTypeBinary {
					t.Errorf("Target type = %q, want %q", targets[0].Type, TargetTypeBinary)
				}
			},
		},
		{
			name: "multiple scripts",
			packageJSON: `{
				"name": "test-package",
				"scripts": {
					"build": "tsc",
					"test": "jest",
					"start": "node index.js",
					"lint": "eslint ."
				}
			}`,
			wantLen: 4,
			wantErr: false,
			checkTargets: func(t *testing.T, targets []Target) {
				// Verify we have the expected types
				hasTest := false
				hasBuild := false
				for _, target := range targets {
					if target.Name == "test" && target.Type == TargetTypeTest {
						hasTest = true
					}
					if target.Name == "build" && target.Type == TargetTypeBinary {
						hasBuild = true
					}
				}
				if !hasTest {
					t.Error("Expected to find test script with TargetTypeTest")
				}
				if !hasBuild {
					t.Error("Expected to find build script with TargetTypeBinary")
				}
			},
		},
		{
			name: "malformed JSON",
			packageJSON: `{
				"name": "test-package",
				"scripts": {
					"build": "tsc"
				` + "}", // Missing closing brace
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "scripts is null",
			packageJSON: `{
				"name": "test-package",
				"scripts": null
			}`,
			wantLen: 0,
			wantErr: false,
		},
	}

	adapter := NewNpmAdapter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory with package.json
			testDir := filepath.Join(tmpDir, tt.name)
			os.MkdirAll(testDir, 0755)
			os.WriteFile(filepath.Join(testDir, "package.json"), []byte(tt.packageJSON), 0644)

			targets, err := adapter.GetTargets(testDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTargets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(targets) != tt.wantLen {
				t.Errorf("GetTargets() returned %d targets, want %d", len(targets), tt.wantLen)
				return
			}

			if tt.checkTargets != nil && len(targets) > 0 {
				tt.checkTargets(t, targets)
			}
		})
	}
}

func TestNpmAdapter_GetTargets_FileNotFound(t *testing.T) {
	adapter := NewNpmAdapter()
	_, err := adapter.GetTargets("/nonexistent/directory")

	if err == nil {
		t.Error("Expected error when package.json doesn't exist, got nil")
	}
}

func TestNpmAdapter_inferTargetType(t *testing.T) {
	adapter := NewNpmAdapter()

	tests := []struct {
		scriptName    string
		scriptCommand string
		want          TargetType
	}{
		// Test scripts
		{"test", "jest", TargetTypeTest},
		{"test:unit", "jest --coverage", TargetTypeTest},
		{"test:integration", "mocha", TargetTypeTest},
		{"e2e", "jest e2e", TargetTypeTest},
		{"spec", "jasmine", TargetTypeTest},
		{"tests", "vitest", TargetTypeTest},

		// Build scripts
		{"build", "webpack", TargetTypeBinary},
		{"build:prod", "tsc -p tsconfig.prod.json", TargetTypeBinary},
		{"compile", "tsc", TargetTypeBinary},
		{"bundle", "vite build", TargetTypeBinary},
		{"dist", "esbuild src/index.ts", TargetTypeBinary},

		// Case insensitive matching
		{"TEST", "JEST", TargetTypeTest},
		{"BUILD", "TSC", TargetTypeBinary},

		// Unknown/utility scripts
		{"start", "node server.js", TargetTypeUnknown},
		{"dev", "nodemon", TargetTypeUnknown},
		{"lint", "eslint .", TargetTypeUnknown},
		{"format", "prettier --write", TargetTypeUnknown},
		{"clean", "rm -rf dist", TargetTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.scriptName, func(t *testing.T) {
			got := adapter.inferTargetType(tt.scriptName, tt.scriptCommand)
			if got != tt.want {
				t.Errorf("inferTargetType(%q, %q) = %q, want %q",
					tt.scriptName, tt.scriptCommand, got, tt.want)
			}
		})
	}
}

func TestNpmAdapter_Build_NilOptions(t *testing.T) {
	adapter := NewNpmAdapter()
	// Set to a command that will fail fast
	adapter.npmBin = "nonexistent-npm-command"

	// Test that nil options doesn't cause panic
	_, err := adapter.Build("/tmp", "test", nil)
	// We expect an error because npm isn't installed, but no panic
	if err == nil {
		t.Error("Expected error when npm not found, got nil")
	}
}

func TestNpmAdapter_Build_ResultStructure(t *testing.T) {
	// This test verifies the Result structure is populated correctly
	// without actually running npm (which may not be installed in test env)

	adapter := NewNpmAdapter()
	adapter.npmBin = "echo" // Use echo as a fake npm that succeeds

	tmpDir := t.TempDir()
	// Create a minimal package.json
	packageJSON := `{
		"name": "test",
		"scripts": {
			"test": "echo test"
		}
	}`
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)

	result, err := adapter.Build(tmpDir, "test", nil)

	// We're using echo, so this should succeed
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Build() returned nil result")
	}

	// Verify result structure
	if result.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if result.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
	if result.Duration <= 0 {
		t.Error("Duration should be positive")
	}
	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
	if result.TargetsBuilt != 1 {
		t.Errorf("TargetsBuilt = %d, want 1", result.TargetsBuilt)
	}
}

func TestNpmAdapter_packageJSON_Parsing(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name: "valid minimal package.json",
			json: `{
				"name": "test-package"
			}`,
			wantErr: false,
		},
		{
			name: "valid with scripts",
			json: `{
				"name": "test-package",
				"version": "1.0.0",
				"scripts": {
					"build": "tsc",
					"test": "jest"
				}
			}`,
			wantErr: false,
		},
		{
			name:    "empty JSON",
			json:    `{}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON - missing brace",
			json:    `{"name": "test"`,
			wantErr: true,
		},
		{
			name:    "invalid JSON - trailing comma",
			json:    `{"name": "test",}`,
			wantErr: true,
		},
		{
			name:    "not JSON at all",
			json:    `this is not json`,
			wantErr: true,
		},
	}

	adapter := NewNpmAdapter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			packagePath := filepath.Join(tmpDir, "package.json")
			os.WriteFile(packagePath, []byte(tt.json), 0644)

			_, err := adapter.GetTargets(tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTargets() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNpmAdapter_GetTargets_RealExample(t *testing.T) {
	// Test with a realistic package.json
	packageJSON := `{
		"name": "codinggame-ui",
		"version": "1.0.0",
		"description": "UI for CodingGame",
		"scripts": {
			"dev": "vite",
			"build": "tsc && vite build",
			"preview": "vite preview",
			"test": "vitest",
			"test:ui": "vitest --ui",
			"test:coverage": "vitest --coverage",
			"lint": "eslint . --ext ts,tsx",
			"format": "prettier --write \"src/**/*.{ts,tsx}\"",
			"typecheck": "tsc --noEmit"
		},
		"dependencies": {
			"react": "^18.0.0"
		},
		"devDependencies": {
			"vite": "^5.0.0",
			"vitest": "^1.0.0"
		}
	}`

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)

	adapter := NewNpmAdapter()
	targets, err := adapter.GetTargets(tmpDir)

	if err != nil {
		t.Fatalf("GetTargets() failed: %v", err)
	}

	if len(targets) != 9 {
		t.Fatalf("Expected 9 scripts, got %d", len(targets))
	}

	// Count target types
	typeCounts := make(map[TargetType]int)
	for _, target := range targets {
		typeCounts[target.Type]++
	}

	// Should have at least 3 test scripts
	if typeCounts[TargetTypeTest] < 3 {
		t.Errorf("Expected at least 3 test scripts, got %d", typeCounts[TargetTypeTest])
	}

	// Should have at least 1 build script
	if typeCounts[TargetTypeBinary] < 1 {
		t.Errorf("Expected at least 1 build script, got %d", typeCounts[TargetTypeBinary])
	}

	// Verify specific scripts exist
	scriptNames := make(map[string]bool)
	for _, target := range targets {
		scriptNames[target.Name] = true
	}

	expectedScripts := []string{"build", "test", "dev", "lint"}
	for _, expected := range expectedScripts {
		if !scriptNames[expected] {
			t.Errorf("Expected to find script %q", expected)
		}
	}
}
