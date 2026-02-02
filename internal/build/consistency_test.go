package build

import (
	"os"
	"path/filepath"
	"testing"
)

// Consistency tests verify that all adapters behave consistently for
// common operations and edge cases.

// allAdapters returns all registered adapters for cross-adapter testing
func allAdapters() []Adapter {
	return []Adapter{
		NewBazelAdapter(),
		NewCargoAdapter(),
		NewNpmAdapter(),
	}
}

func TestConsistency_Detect_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			detected, err := adapter.Detect(tmpDir)

			// All adapters should return (false, nil) for empty directories
			if err != nil {
				t.Errorf("%s.Detect() returned error for empty dir: %v", adapter.Name(), err)
			}
			if detected {
				t.Errorf("%s.Detect() returned true for empty dir, want false", adapter.Name())
			}
		})
	}
}

func TestConsistency_Detect_NonExistentDirectory(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "this-does-not-exist")

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			detected, err := adapter.Detect(nonExistent)

			// All adapters should return (false, nil) for non-existent directories
			// This is documented behavior in the adapter interface
			if err != nil {
				t.Errorf("%s.Detect() returned error for non-existent dir: %v", adapter.Name(), err)
			}
			if detected {
				t.Errorf("%s.Detect() returned true for non-existent dir, want false", adapter.Name())
			}
		})
	}
}

func TestConsistency_Detect_PathIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "regular_file.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			detected, err := adapter.Detect(filePath)

			// All adapters should return an error when path is a file
			// The adapter documentation specifies this behavior
			if err == nil {
				t.Errorf("%s.Detect() should return error when path is a file, got nil", adapter.Name())
			}
			if detected {
				t.Errorf("%s.Detect() returned true for file path, want false", adapter.Name())
			}
		})
	}
}

func TestConsistency_Build_NilOptions(t *testing.T) {
	// We can't actually run builds without the tools installed,
	// but we can verify nil options don't cause panics

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s.Build() panicked with nil options: %v", adapter.Name(), r)
				}
			}()

			// Call Build with nil options - should not panic
			// We expect an error because the build tool isn't installed,
			// but that's fine - we're testing the nil handling
			_, err := adapter.Build(t.TempDir(), "//fake:target", nil)

			// Error is expected (tool not found), panic is not
			if err == nil {
				// This would be surprising but not necessarily wrong
				t.Log("Build succeeded unexpectedly (tool might be installed)")
			}
		})
	}
}

func TestConsistency_Name_StableAndNonEmpty(t *testing.T) {
	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			name1 := adapter.Name()
			name2 := adapter.Name()

			// Name should be non-empty
			if name1 == "" {
				t.Error("Name() returned empty string")
			}

			// Name should be stable (same value on repeated calls)
			if name1 != name2 {
				t.Errorf("Name() not stable: %q != %q", name1, name2)
			}

			// Name should be lowercase (convention)
			for _, c := range name1 {
				if c >= 'A' && c <= 'Z' {
					t.Errorf("Name() contains uppercase: %q", name1)
					break
				}
			}

			// Name should not contain whitespace
			for _, c := range name1 {
				if c == ' ' || c == '\t' || c == '\n' {
					t.Errorf("Name() contains whitespace: %q", name1)
					break
				}
			}
		})
	}
}

func TestConsistency_Detect_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			// Call Detect multiple times
			results := make([]bool, 5)
			errors := make([]error, 5)

			for i := 0; i < 5; i++ {
				results[i], errors[i] = adapter.Detect(tmpDir)
			}

			// All results should be the same
			for i := 1; i < 5; i++ {
				if results[i] != results[0] {
					t.Errorf("Detect() not idempotent: call %d returned %v, call 0 returned %v",
						i, results[i], results[0])
				}
				// Error consistency (both nil or both non-nil with same message)
				if (errors[i] == nil) != (errors[0] == nil) {
					t.Errorf("Detect() error not idempotent: call %d error=%v, call 0 error=%v",
						i, errors[i], errors[0])
				}
			}
		})
	}
}

func TestConsistency_Detect_DoesNotModifyFilesystem(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a marker file
	markerPath := filepath.Join(tmpDir, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	// Get initial directory state
	initialEntries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			_, _ = adapter.Detect(tmpDir)

			// Check directory wasn't modified
			afterEntries, err := os.ReadDir(tmpDir)
			if err != nil {
				t.Fatal(err)
			}

			if len(afterEntries) != len(initialEntries) {
				t.Errorf("%s.Detect() modified directory: had %d entries, now has %d",
					adapter.Name(), len(initialEntries), len(afterEntries))
			}

			// Check marker file wasn't modified
			content, err := os.ReadFile(markerPath)
			if err != nil {
				t.Errorf("Marker file disappeared after %s.Detect()", adapter.Name())
			} else if string(content) != "original" {
				t.Errorf("Marker file modified after %s.Detect(): %q", adapter.Name(), content)
			}
		})
	}
}

func TestConsistency_AllAdaptersHaveUniqueNames(t *testing.T) {
	adapters := allAdapters()
	seen := make(map[string]bool)

	for _, adapter := range adapters {
		name := adapter.Name()
		if seen[name] {
			t.Errorf("Duplicate adapter name: %q", name)
		}
		seen[name] = true
	}
}

func TestConsistency_Registry_FindsAllAdapters(t *testing.T) {
	registry := NewRegistry()
	adapters := allAdapters()

	for _, adapter := range adapters {
		registry.Register(adapter)
	}

	// Verify all adapters can be retrieved by name
	for _, adapter := range adapters {
		retrieved, err := registry.GetAdapter(adapter.Name())
		if err != nil {
			t.Errorf("GetAdapter(%q) failed: %v", adapter.Name(), err)
			continue
		}
		if retrieved.Name() != adapter.Name() {
			t.Errorf("GetAdapter(%q) returned wrong adapter: %q", adapter.Name(), retrieved.Name())
		}
	}
}

func TestConsistency_Registry_DetectOnEmptyDir(t *testing.T) {
	registry := NewRegistry()
	for _, adapter := range allAdapters() {
		registry.Register(adapter)
	}

	tmpDir := t.TempDir()
	detected, err := registry.Detect(tmpDir)

	if err != nil {
		t.Errorf("Registry.Detect() returned error: %v", err)
	}
	if len(detected) != 0 {
		t.Errorf("Registry.Detect() found %d adapters in empty dir, want 0", len(detected))
		for _, d := range detected {
			t.Logf("  Found: %s", d.Name())
		}
	}
}

func TestConsistency_Detect_WithOwnConfigFile(t *testing.T) {
	// Test that each adapter detects only its own config files
	tests := []struct {
		adapterName string
		configFile  string
		configDir   bool // true if config is a directory
	}{
		{"bazel", "WORKSPACE", false},
		{"bazel", "WORKSPACE.bazel", false},
		{"cargo", "Cargo.toml", false},
		{"npm", "package.json", false},
	}

	for _, tc := range tests {
		t.Run(tc.adapterName+"_"+tc.configFile, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create the config file
			configPath := filepath.Join(tmpDir, tc.configFile)
			if tc.configDir {
				if err := os.MkdirAll(configPath, 0755); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Test all adapters
			for _, adapter := range allAdapters() {
				detected, err := adapter.Detect(tmpDir)
				if err != nil {
					t.Errorf("%s.Detect() returned error: %v", adapter.Name(), err)
					continue
				}

				if adapter.Name() == tc.adapterName {
					// This adapter should detect its config
					if !detected {
						t.Errorf("%s.Detect() should detect %s, but didn't", adapter.Name(), tc.configFile)
					}
				} else {
					// Other adapters should NOT detect this config
					if detected {
						t.Errorf("%s.Detect() incorrectly detected %s (belongs to %s)",
							adapter.Name(), tc.configFile, tc.adapterName)
					}
				}
			}
		})
	}
}

func TestConsistency_ConfigFileIsDirectory_NotDetected(t *testing.T) {
	// When config file path is actually a directory, adapters should not detect
	tests := []struct {
		adapterName string
		configName  string
	}{
		{"bazel", "WORKSPACE"},
		{"cargo", "Cargo.toml"},
		{"npm", "package.json"},
	}

	for _, tc := range tests {
		t.Run(tc.adapterName+"_dir", func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create config name as directory instead of file
			configPath := filepath.Join(tmpDir, tc.configName)
			if err := os.MkdirAll(configPath, 0755); err != nil {
				t.Fatal(err)
			}

			// Find the matching adapter
			for _, adapter := range allAdapters() {
				if adapter.Name() != tc.adapterName {
					continue
				}

				detected, err := adapter.Detect(tmpDir)
				if err != nil {
					t.Errorf("%s.Detect() returned error: %v", adapter.Name(), err)
					continue
				}

				if detected {
					t.Errorf("%s.Detect() detected directory named %s as config file",
						adapter.Name(), tc.configName)
				}
			}
		})
	}
}

func TestConsistency_SymlinkToConfigFile(t *testing.T) {
	// Test that symlinks to config files are detected
	// Skip on Windows where symlinks require special permissions
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping symlink test on Windows")
	}

	tests := []struct {
		adapterName string
		configName  string
	}{
		{"bazel", "WORKSPACE"},
		{"cargo", "Cargo.toml"},
		{"npm", "package.json"},
	}

	for _, tc := range tests {
		t.Run(tc.adapterName+"_symlink", func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create real config file with different name
			realPath := filepath.Join(tmpDir, "real_config")
			if err := os.WriteFile(realPath, []byte("{}"), 0644); err != nil {
				t.Fatal(err)
			}

			// Create symlink with expected name
			linkPath := filepath.Join(tmpDir, tc.configName)
			if err := os.Symlink(realPath, linkPath); err != nil {
				t.Skip("Cannot create symlink:", err)
			}

			// Find the matching adapter
			for _, adapter := range allAdapters() {
				if adapter.Name() != tc.adapterName {
					continue
				}

				detected, err := adapter.Detect(tmpDir)
				if err != nil {
					t.Errorf("%s.Detect() returned error for symlinked config: %v", adapter.Name(), err)
					continue
				}

				if !detected {
					t.Errorf("%s.Detect() should detect symlinked %s", adapter.Name(), tc.configName)
				}
			}
		})
	}
}

func TestConsistency_EmptyConfigFile_StillDetected(t *testing.T) {
	// Adapters should detect empty config files (detection is about presence, not validity)
	tests := []struct {
		adapterName string
		configName  string
	}{
		{"bazel", "WORKSPACE"},
		{"cargo", "Cargo.toml"},
		{"npm", "package.json"},
	}

	for _, tc := range tests {
		t.Run(tc.adapterName+"_empty", func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create empty config file
			configPath := filepath.Join(tmpDir, tc.configName)
			if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
				t.Fatal(err)
			}

			for _, adapter := range allAdapters() {
				if adapter.Name() != tc.adapterName {
					continue
				}

				detected, err := adapter.Detect(tmpDir)
				if err != nil {
					t.Errorf("%s.Detect() returned error for empty config: %v", adapter.Name(), err)
					continue
				}

				if !detected {
					t.Errorf("%s.Detect() should detect empty %s", adapter.Name(), tc.configName)
				}
			}
		})
	}
}

func TestConsistency_BuildOptions_DefaultsApplied(t *testing.T) {
	// Verify that when BuildOptions is nil or has zero values,
	// adapters apply sensible defaults and don't panic

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name()+"_nil_opts", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Build with nil options panicked: %v", r)
				}
			}()

			// This will fail because tools aren't installed, but shouldn't panic
			_, _ = adapter.Build(t.TempDir(), "//fake:target", nil)
		})

		t.Run(adapter.Name()+"_zero_opts", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Build with zero options panicked: %v", r)
				}
			}()

			// Zero-value BuildOptions should work
			_, _ = adapter.Build(t.TempDir(), "//fake:target", &BuildOptions{})
		})
	}
}

func TestConsistency_GetTargets_EmptyDirectory(t *testing.T) {
	// GetTargets on a directory without proper config should error
	// (because Detect would return false first in normal use)

	for _, adapter := range allAdapters() {
		t.Run(adapter.Name(), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("GetTargets on empty dir panicked: %v", r)
				}
			}()

			tmpDir := t.TempDir()
			targets, err := adapter.GetTargets(tmpDir)

			// We expect either:
			// 1. An error (tool not found, or no config file)
			// 2. An empty target list
			// But NOT a panic
			_ = targets
			_ = err
		})
	}
}
