package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCargoAdapter_Name(t *testing.T) {
	adapter := NewCargoAdapter()
	if adapter.Name() != "cargo" {
		t.Errorf("Name() = %q, want %q", adapter.Name(), "cargo")
	}
}

func TestCargoAdapter_Detect(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setupFunc   func() string
		wantPresent bool
		wantErr     bool
	}{
		{
			name: "Cargo.toml exists",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test1")
				os.MkdirAll(dir, 0755)
				os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
				return dir
			},
			wantPresent: true,
			wantErr:     false,
		},
		{
			name: "Cargo.toml missing",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test2")
				os.MkdirAll(dir, 0755)
				return dir
			},
			wantPresent: false,
			wantErr:     false,
		},
		{
			name: "Cargo.toml is a directory",
			setupFunc: func() string {
				dir := filepath.Join(tmpDir, "test3")
				os.MkdirAll(dir, 0755)
				os.MkdirAll(filepath.Join(dir, "Cargo.toml"), 0755)
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

	adapter := NewCargoAdapter()

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

func TestCargoAdapter_parseMetadataOutput(t *testing.T) {
	adapter := NewCargoAdapter()

	tests := []struct {
		name       string
		output     string
		wantLen    int
		wantErr    bool
		checkFirst func(*testing.T, Target)
	}{
		{
			name:    "empty output",
			output:  "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "single binary target",
			output: `{
				"packages": [{
					"name": "myapp",
					"targets": [{
						"name": "myapp",
						"kind": ["bin"]
					}]
				}]
			}`,
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.Name != "myapp" {
					t.Errorf("Name = %q, want myapp", target.Name)
				}
				if target.Type != TargetTypeBinary {
					t.Errorf("Type = %q, want %q", target.Type, TargetTypeBinary)
				}
			},
		},
		{
			name: "library and binary targets",
			output: `{
				"packages": [{
					"targets": [{
						"name": "mylib",
						"kind": ["lib"]
					}, {
						"name": "mybin",
						"kind": ["bin"]
					}]
				}]
			}`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "test target",
			output: `{
				"targets": [{
					"name": "integration_test",
					"kind": ["test"]
				}]
			}`,
			wantLen: 1,
			wantErr: false,
			checkFirst: func(t *testing.T, target Target) {
				if target.Type != TargetTypeTest {
					t.Errorf("Type = %q, want %q", target.Type, TargetTypeTest)
				}
			},
		},
		{
			name: "multiple target types",
			output: `{
				"targets": [
					{"name": "lib1", "kind": ["lib"]},
					{"name": "bin1", "kind": ["bin"]},
					{"name": "test1", "kind": ["test"]},
					{"name": "bench1", "kind": ["bench"]}
				]
			}`,
			wantLen: 4,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := adapter.parseMetadataOutput(tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetadataOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(targets) != tt.wantLen {
				t.Errorf("parseMetadataOutput() returned %d targets, want %d", len(targets), tt.wantLen)
				return
			}

			if tt.checkFirst != nil && len(targets) > 0 {
				tt.checkFirst(t, targets[0])
			}
		})
	}
}

func TestCargoAdapter_inferTargetType(t *testing.T) {
	adapter := NewCargoAdapter()

	tests := []struct {
		kind string
		want TargetType
	}{
		// Binary types
		{"bin", TargetTypeBinary},

		// Library types
		{"lib", TargetTypeLibrary},
		{"rlib", TargetTypeLibrary},
		{"dylib", TargetTypeLibrary},
		{"cdylib", TargetTypeLibrary},
		{"staticlib", TargetTypeLibrary},
		{"proc-macro", TargetTypeLibrary},

		// Test types
		{"test", TargetTypeTest},
		{"bench", TargetTypeTest},

		// Unknown types
		{"example", TargetTypeUnknown},
		{"custom", TargetTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := adapter.inferTargetType(tt.kind)
			if got != tt.want {
				t.Errorf("inferTargetType(%q) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestCargoAdapter_extractMetrics(t *testing.T) {
	adapter := NewCargoAdapter()

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
			wantTargets:     1, // Defaults to 1
			wantCacheHits:   0,
			wantCacheMisses: 0,
		},
		{
			name: "all compiled",
			output: `   Compiling proc-macro2 v1.0.0
   Compiling unicode-ident v1.0.0
   Compiling serde v1.0.0
    Finished dev [unoptimized + debuginfo] target(s) in 12.34s`,
			wantTargets:     3,
			wantCacheHits:   0,
			wantCacheMisses: 3,
		},
		{
			name: "all cached",
			output: `       Fresh proc-macro2 v1.0.0
       Fresh unicode-ident v1.0.0
       Fresh serde v1.0.0
    Finished dev [unoptimized + debuginfo] target(s) in 0.05s`,
			wantTargets:     3,
			wantCacheHits:   3,
			wantCacheMisses: 0,
		},
		{
			name: "mixed compiled and cached",
			output: `       Fresh serde v1.0.0
   Compiling mylib v0.1.0
       Fresh tokio v1.0.0
   Compiling myapp v0.1.0
    Finished dev [unoptimized + debuginfo] target(s) in 5.67s`,
			wantTargets:     4,
			wantCacheHits:   2,
			wantCacheMisses: 2,
		},
		{
			name: "release build",
			output: `   Compiling myapp v0.1.0
    Finished release [optimized] target(s) in 45.12s`,
			wantTargets:     1,
			wantCacheHits:   0,
			wantCacheMisses: 1,
		},
		{
			name: "with warnings",
			output: `   Compiling mylib v0.1.0
warning: unused variable: foo
   Compiling myapp v0.1.0
    Finished dev [unoptimized + debuginfo] target(s) in 3.21s`,
			wantTargets:     2,
			wantCacheHits:   0,
			wantCacheMisses: 2,
		},
		{
			name: "whitespace variations",
			output: `     Fresh   serde v1.0.0
Compiling   myapp v0.1.0
  Finished dev [unoptimized + debuginfo] target(s) in 1.23s`,
			wantTargets:     2,
			wantCacheHits:   1,
			wantCacheMisses: 1,
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

func TestCargoAdapter_extractMetrics_RealExample(t *testing.T) {
	// Test with realistic cargo build output
	output := `   Compiling proc-macro2 v1.0.70
   Compiling unicode-ident v1.0.12
   Compiling serde v1.0.193
   Compiling syn v2.0.41
       Fresh quote v1.0.33
       Fresh serde_json v1.0.108
   Compiling mylib v0.1.0 (/home/user/myproject/mylib)
   Compiling myapp v0.1.0 (/home/user/myproject)
    Finished dev [unoptimized + debuginfo] target(s) in 8.42s
`

	adapter := NewCargoAdapter()
	result := &Result{}
	adapter.extractMetrics(output, result)

	// Should have 6 compiled + 2 fresh = 8 targets
	if result.TargetsBuilt != 8 {
		t.Errorf("TargetsBuilt = %d, want 8", result.TargetsBuilt)
	}

	// Should have 2 fresh (cache hits)
	if result.CacheHits != 2 {
		t.Errorf("CacheHits = %d, want 2", result.CacheHits)
	}

	// Should have 6 compiled (cache misses)
	if result.CacheMisses != 6 {
		t.Errorf("CacheMisses = %d, want 6", result.CacheMisses)
	}

	// Verify cache hit rate
	expectedRate := (2.0 / 8.0) * 100.0 // 25%
	rate := result.CacheHitRate()
	if rate != expectedRate {
		t.Errorf("CacheHitRate() = %v, want %v", rate, expectedRate)
	}
}

func TestCargoAdapter_Build_NilOptions(t *testing.T) {
	adapter := NewCargoAdapter()
	// Set to a command that will fail fast
	adapter.cargoBin = "nonexistent-cargo-command"

	// Test that nil options doesn't cause panic
	_, err := adapter.Build("/tmp", "test", nil)
	// We expect an error because cargo isn't installed, but no panic
	if err == nil {
		t.Error("Expected error when cargo not found, got nil")
	}
}

func TestCargoAdapter_parseMetadataOutput_EdgeCases(t *testing.T) {
	adapter := NewCargoAdapter()

	tests := []struct {
		name    string
		output  string
		wantLen int
	}{
		{
			name:    "malformed JSON is gracefully handled",
			output:  `{"name": "incomplete"`,
			wantLen: 0, // Parser should handle this gracefully
		},
		{
			name:    "name without kind",
			output:  `{"name": "myapp"}`,
			wantLen: 0, // Need both name and kind
		},
		{
			name:    "kind without name",
			output:  `{"kind": ["bin"]}`,
			wantLen: 0, // Need both name and kind
		},
		{
			name: "duplicate names",
			output: `{
				"targets": [
					{"name": "myapp", "kind": ["bin"]},
					{"name": "myapp", "kind": ["lib"]}
				]
			}`,
			wantLen: 2, // Both should be captured
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, _ := adapter.parseMetadataOutput(tt.output)
			if len(targets) != tt.wantLen {
				t.Errorf("parseMetadataOutput() returned %d targets, want %d", len(targets), tt.wantLen)
			}
		})
	}
}

func TestCargoAdapter_extractMetrics_EdgeCases(t *testing.T) {
	adapter := NewCargoAdapter()

	tests := []struct {
		name        string
		output      string
		description string
		check       func(*testing.T, *Result)
	}{
		{
			name:        "no finished line",
			output:      "Compiling myapp v0.1.0\n",
			description: "should still count compiled targets",
			check: func(t *testing.T, r *Result) {
				if r.CacheMisses != 1 {
					t.Errorf("CacheMisses = %d, want 1", r.CacheMisses)
				}
			},
		},
		{
			name:        "only finished line",
			output:      "Finished dev [unoptimized + debuginfo] target(s) in 0.01s\n",
			description: "should default to 1 target",
			check: func(t *testing.T, r *Result) {
				if r.TargetsBuilt != 1 {
					t.Errorf("TargetsBuilt = %d, want 1", r.TargetsBuilt)
				}
			},
		},
		{
			name: "with error messages",
			output: `   Compiling myapp v0.1.0
error[E0425]: cannot find value 'undefined_var' in this scope
error: could not compile 'myapp'`,
			description: "should count targets even with errors",
			check: func(t *testing.T, r *Result) {
				if r.CacheMisses != 1 {
					t.Errorf("CacheMisses = %d, want 1", r.CacheMisses)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &Result{}
			adapter.extractMetrics(tt.output, result)
			tt.check(t, result)
		})
	}
}

func TestCargoAdapter_BuildOptions_Clean(t *testing.T) {
	// Test that Clean option is handled without panic
	adapter := NewCargoAdapter()
	adapter.cargoBin = "nonexistent-cargo"

	opts := &BuildOptions{
		Clean: true,
	}

	_, err := adapter.Build("/tmp", "test", opts)
	// Should error because cargo doesn't exist, but shouldn't panic during clean
	if err == nil {
		t.Error("Expected error when cargo not found")
	}
}

func TestCargoAdapter_BuildOptions_Jobs(t *testing.T) {
	// Test that Jobs option is handled
	adapter := NewCargoAdapter()
	adapter.cargoBin = "echo" // Use echo to avoid needing real cargo

	opts := &BuildOptions{
		Jobs: 4,
	}

	// This will fail but won't panic
	_, _ = adapter.Build("/tmp", "test", opts)
}
