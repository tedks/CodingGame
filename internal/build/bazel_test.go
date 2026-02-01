package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBazelAdapter_Name(t *testing.T) {
	adapter := NewBazelAdapter()
	if adapter.Name() != "bazel" {
		t.Errorf("Name() = %q, want %q", adapter.Name(), "bazel")
	}
}

func TestBazelAdapter_Detect(t *testing.T) {
	// Create temporary test directories
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setupFunc   func() string // Returns path to test
		wantPresent bool
		wantErr     bool
	}{
		{
			name: "WORKSPACE file exists",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test1")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "WORKSPACE"), []byte(""), 0644)
				return dir
			},
			wantPresent: true,
			wantErr:     false,
		},
		{
			name: "WORKSPACE.bazel file exists",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test2")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "WORKSPACE.bazel"), []byte(""), 0644)
				return dir
			},
			wantPresent: true,
			wantErr:     false,
		},
		{
			name: "both WORKSPACE files exist",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test3")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "WORKSPACE"), []byte(""), 0644)
				os.WriteFile(filepath.Join(dir, "WORKSPACE.bazel"), []byte(""), 0644)
				return dir
			},
			wantPresent: true,
			wantErr:     false,
		},
		{
			name: "no WORKSPACE file",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test4")
				os.MkdirAll(dir, 0755)
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
			wantErr:     false, // Returns false, nil for nonexistent dirs
		},
		{
			name: "WORKSPACE is a directory (not a file)",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test5")
				os.MkdirAll(dir, 0755)
				os.MkdirAll(filepath.Join(dir, "WORKSPACE"), 0755)
				return dir
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

	adapter := NewBazelAdapter()

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

func TestBazelAdapter_parseQueryOutput(t *testing.T) {
	adapter := NewBazelAdapter()

	tests := []struct {
		name       string
		output     string
		wantLen    int
		wantErr    bool
		checkFirst func(*testing.T, Target) // Optional validation of first target
	}{
		{
			name:    "empty output",
			output:  "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "single go_binary target",
			output:  "go_binary rule //cmd/app:app\n",
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.ID != "//cmd/app:app" {
					t.Errorf("ID = %q, want //cmd/app:app", target.ID)
				}
				if target.Name != "app" {
					t.Errorf("Name = %q, want app", target.Name)
				}
				if target.Type != TargetTypeBinary {
					t.Errorf("Type = %q, want %q", target.Type, TargetTypeBinary)
				}
			},
		},
		{
			name: "multiple targets",
			output: `go_binary rule //cmd/app:app
go_library rule //internal/pkg:pkg
go_test rule //internal/pkg:pkg_test
`,
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "target with underscores",
			output:  "go_library rule //internal/my_package:my_package\n",
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.Name != "my_package" {
					t.Errorf("Name = %q, want my_package", target.Name)
				}
			},
		},
		{
			name:    "target with hyphens",
			output:  "go_binary rule //cmd/my-app:my-app\n",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "test target detection",
			output:  "go_test rule //tests:integration_test\n",
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.Type != TargetTypeTest {
					t.Errorf("Type = %q, want %q", target.Type, TargetTypeTest)
				}
			},
		},
		{
			name:    "library target detection",
			output:  "cc_library rule //lib:mylib\n",
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.Type != TargetTypeLibrary {
					t.Errorf("Type = %q, want %q", target.Type, TargetTypeLibrary)
				}
			},
		},
		{
			name: "malformed lines are skipped",
			output: `go_binary rule //cmd/app:app
this is not a valid line
random text
go_library rule //lib:lib
`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "whitespace variations",
			output:  "  go_binary rule //cmd/app:app  \n",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "target without colon separator",
			output:  "filegroup rule //some/path\n",
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.Name != "path" {
					t.Errorf("Name = %q, want path", target.Name)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := adapter.parseQueryOutput(tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseQueryOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(targets) != tt.wantLen {
				t.Errorf("parseQueryOutput() returned %d targets, want %d", len(targets), tt.wantLen)
				return
			}

			if tt.checkFirst != nil && len(targets) > 0 {
				tt.checkFirst(t, targets[0])
			}
		})
	}
}

func TestBazelAdapter_inferTargetType(t *testing.T) {
	adapter := NewBazelAdapter()

	tests := []struct {
		ruleType string
		want     TargetType
	}{
		// Explicit matches
		{"go_test", TargetTypeTest},
		{"java_test", TargetTypeTest},
		{"py_test", TargetTypeTest},
		{"cc_test", TargetTypeTest},

		{"go_binary", TargetTypeBinary},
		{"java_binary", TargetTypeBinary},
		{"py_binary", TargetTypeBinary},
		{"cc_binary", TargetTypeBinary},

		{"go_library", TargetTypeLibrary},
		{"java_library", TargetTypeLibrary},
		{"py_library", TargetTypeLibrary},
		{"cc_library", TargetTypeLibrary},

		// Substring matches
		{"my_test_rule", TargetTypeTest},
		{"custom_binary_rule", TargetTypeBinary},
		{"proto_library", TargetTypeLibrary},

		// Case insensitivity
		{"GO_TEST", TargetTypeTest},
		{"Go_Binary", TargetTypeBinary},
		{"GO_LIBRARY", TargetTypeLibrary},

		// Unknown types
		{"filegroup", TargetTypeUnknown},
		{"genrule", TargetTypeUnknown},
		{"custom_rule", TargetTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.ruleType, func(t *testing.T) {
			got := adapter.inferTargetType(tt.ruleType)
			if got != tt.want {
				t.Errorf("inferTargetType(%q) = %q, want %q", tt.ruleType, got, tt.want)
			}
		})
	}
}

func TestBazelAdapter_extractMetrics(t *testing.T) {
	adapter := NewBazelAdapter()

	tests := []struct {
		name            string
		output          string
		wantTargets     int
		wantCacheHits   int64
		wantCacheMisses int64
	}{
		{
			name:            "empty output",
			output:          "",
			wantTargets:     0,
			wantCacheHits:   0,
			wantCacheMisses: 0,
		},
		{
			name: "successful build with metrics",
			output: `INFO: Analyzed target //cmd/app:app (1 packages loaded, 10 targets configured).
INFO: Found 1 target...
INFO: From: Building cmd/app/main.go
INFO: Build completed successfully, 42 total actions
INFO: 20 remote cache hits
INFO: 15 processes
`,
			wantTargets:     42,
			wantCacheHits:   20,
			wantCacheMisses: 15,
		},
		{
			name: "build with local cache",
			output: `INFO: Build completed successfully, 100 total actions
INFO: 75 local cache hits
INFO: 25 processes
`,
			wantTargets:     100,
			wantCacheHits:   75,
			wantCacheMisses: 25,
		},
		{
			name: "build with no cache",
			output: `INFO: Build completed successfully, 50 total actions
INFO: 50 processes
`,
			wantTargets:     50,
			wantCacheHits:   0,
			wantCacheMisses: 50,
		},
		{
			name: "build with only total actions",
			output: `INFO: Analyzed target //lib:lib
INFO: Build completed successfully, 30 total actions
`,
			wantTargets:     30,
			wantCacheHits:   0,
			wantCacheMisses: 30, // Assumes all were cache misses
		},
		{
			name: "multiple cache hit lines (cumulative)",
			output: `INFO: Build completed successfully, 100 total actions
INFO: 40 remote cache hits
INFO: 30 local cache hits
INFO: 30 processes
`,
			wantTargets:     100,
			wantCacheHits:   70, // 40 + 30
			wantCacheMisses: 30,
		},
		{
			name: "build output with noise",
			output: `Loading: 0 packages loaded
Analyzing: 10 targets
INFO: Build completed successfully, 25 total actions
Some other info line
INFO: 10 cache hits
INFO: 15 processes
Build finished at 2024-01-15
`,
			wantTargets:     25,
			wantCacheHits:   10,
			wantCacheMisses: 15,
		},
		{
			name: "singular forms",
			output: `INFO: Build completed successfully, 1 total action
INFO: 1 cache hit
INFO: 0 processes
`,
			wantTargets:     1,
			wantCacheHits:   1,
			wantCacheMisses: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{}
			adapter.extractMetrics(tt.output, result)

			if result.TargetsBuilt != tt.wantTargets {
				t.Errorf("TargetsBuilt = %d, want %d", result.TargetsBuilt, tt.wantTargets)
			}
			if result.CacheHits != tt.wantCacheHits {
				t.Errorf("CacheHits = %d, want %d", result.CacheHits, tt.wantCacheHits)
			}
			if result.CacheMisses != tt.wantCacheMisses {
				t.Errorf("CacheMisses = %d, want %d", result.CacheMisses, tt.wantCacheMisses)
			}
		})
	}
}

func TestBazelAdapter_Build_NilOptions(t *testing.T) {
	adapter := NewBazelAdapter()
	// Set to a command that will fail fast
	adapter.bazelBin = "nonexistent-bazel-command"

	// Test that nil options doesn't cause panic
	_, err := adapter.Build("/tmp", "//test:target", nil)
	// We expect an error because bazel isn't installed, but no panic
	if err == nil {
		t.Error("Expected error when bazel not found, got nil")
	}
}

func TestBuildOptions_Defaults(t *testing.T) {
	opts := &BuildOptions{}

	// Verify zero values
	if opts.Timeout != 0 {
		t.Errorf("default Timeout = %v, want 0", opts.Timeout)
	}
	if opts.Clean {
		t.Error("default Clean = true, want false")
	}
	if opts.Jobs != 0 {
		t.Errorf("default Jobs = %d, want 0", opts.Jobs)
	}
}

func TestResult_CacheHitRate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		result      *Result
		wantRate    float64
		description string
	}{
		{
			name: "no cache operations",
			result: &Result{
				CacheHits:   0,
				CacheMisses: 0,
			},
			wantRate:    0,
			description: "should return 0 when no cache operations occurred",
		},
		{
			name: "only cache hits",
			result: &Result{
				CacheHits:   100,
				CacheMisses: 0,
			},
			wantRate:    100.0,
			description: "should return 100% when all operations were cache hits",
		},
		{
			name: "only cache misses",
			result: &Result{
				CacheHits:   0,
				CacheMisses: 100,
			},
			wantRate:    0,
			description: "should return 0% when all operations were cache misses",
		},
		{
			name: "balanced mix",
			result: &Result{
				CacheHits:   50,
				CacheMisses: 50,
			},
			wantRate:    50.0,
			description: "should return 50% for balanced hit/miss ratio",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := tt.result.CacheHitRate()
			if rate != tt.wantRate {
				t.Errorf("CacheHitRate() = %v, want %v (%s)", rate, tt.wantRate, tt.description)
			}
		})
	}
}

func TestTarget_Construction(t *testing.T) {
	// Test that Target can be constructed with all fields
	target := Target{
		ID:           "//cmd/app:app",
		Name:         "app",
		Type:         TargetTypeBinary,
		Description:  "Main application binary",
		Dependencies: []string{"//lib:lib1", "//lib:lib2"},
		Tags:         []string{"manual", "requires-network"},
	}

	// Verify all fields are accessible
	if target.ID != "//cmd/app:app" {
		t.Errorf("ID not set correctly")
	}
	if target.Name != "app" {
		t.Errorf("Name not set correctly")
	}
	if target.Type != TargetTypeBinary {
		t.Errorf("Type not set correctly")
	}
	if len(target.Dependencies) != 2 {
		t.Errorf("Dependencies not set correctly")
	}
	if len(target.Tags) != 2 {
		t.Errorf("Tags not set correctly")
	}
}

func TestResult_TimingFields(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Second)

	result := &Result{
		StartTime: now,
		EndTime:   later,
		Duration:  5 * time.Second,
	}

	if !result.StartTime.Equal(now) {
		t.Error("StartTime not set correctly")
	}
	if !result.EndTime.Equal(later) {
		t.Error("EndTime not set correctly")
	}
	if result.Duration != 5*time.Second {
		t.Error("Duration not set correctly")
	}

	// Verify Duration matches EndTime - StartTime
	calculatedDuration := result.EndTime.Sub(result.StartTime)
	if calculatedDuration != result.Duration {
		t.Errorf("Duration mismatch: calculated %v, stored %v", calculatedDuration, result.Duration)
	}
}

func TestBazelAdapter_parseQueryOutput_RealExample(t *testing.T) {
	// Test with real-looking Bazel query output
	output := `go_library rule //internal/build:build
go_test rule //internal/build:build_test
go_library rule //internal/tile:tile
go_test rule //internal/tile:tile_test
go_library rule //internal/game:game
go_test rule //internal/game:game_test
go_binary rule //:codinggame
`

	adapter := NewBazelAdapter()
	targets, err := adapter.parseQueryOutput(output)

	if err != nil {
		t.Fatalf("parseQueryOutput() failed: %v", err)
	}

	if len(targets) != 7 {
		t.Fatalf("Expected 7 targets, got %d", len(targets))
	}

	// Check that we got the right mix of types
	typeCounts := make(map[TargetType]int)
	for _, target := range targets {
		typeCounts[target.Type]++
	}

	if typeCounts[TargetTypeLibrary] != 3 {
		t.Errorf("Expected 3 library targets, got %d", typeCounts[TargetTypeLibrary])
	}
	if typeCounts[TargetTypeTest] != 3 {
		t.Errorf("Expected 3 test targets, got %d", typeCounts[TargetTypeTest])
	}
	if typeCounts[TargetTypeBinary] != 1 {
		t.Errorf("Expected 1 binary target, got %d", typeCounts[TargetTypeBinary])
	}

	// Verify the binary target specifically
	var binaryTarget *Target
	for i := range targets {
		if targets[i].Type == TargetTypeBinary {
			binaryTarget = &targets[i]
			break
		}
	}

	if binaryTarget == nil {
		t.Fatal("Binary target not found")
	}

	if binaryTarget.Name != "codinggame" {
		t.Errorf("Binary target name = %q, want 'codinggame'", binaryTarget.Name)
	}
	if !strings.Contains(binaryTarget.Description, "go_binary") {
		t.Errorf("Binary target description should mention go_binary, got %q", binaryTarget.Description)
	}
}
