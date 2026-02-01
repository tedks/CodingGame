package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegration_BazelWorkflow tests a complete Bazel workflow
func TestIntegration_BazelWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Bazel workspace
	workspaceContent := `workspace(name = "test_workspace")
`
	if err := os.WriteFile(filepath.Join(tmpDir, "WORKSPACE"), []byte(workspaceContent), 0644); err != nil {
		t.Fatalf("Failed to create WORKSPACE: %v", err)
	}

	// Test the workflow
	adapter := NewBazelAdapter()

	// 1. Detect should succeed
	detected, err := adapter.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true for directory with WORKSPACE")
	}

	// 2. GetTargets should work (though will fail without bazel installed)
	// We expect an error here since bazel isn't installed in test environment
	_, err = adapter.GetTargets(tmpDir)
	if err == nil {
		t.Log("GetTargets succeeded (bazel is installed)")
	} else {
		// Expected: bazel not found error
		if err.Error() == "" {
			t.Error("GetTargets error message is empty")
		}
	}

	// 3. Build should also fail gracefully without bazel
	opts := &BuildOptions{
		Timeout: 5 * time.Second,
	}
	_, err = adapter.Build(tmpDir, "//test:target", opts)
	if err == nil {
		t.Log("Build succeeded (bazel is installed)")
	} else {
		// Expected error
		if err.Error() == "" {
			t.Error("Build error message is empty")
		}
	}
}

// TestIntegration_NpmWorkflow tests a complete npm workflow
func TestIntegration_NpmWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal package.json
	packageJSON := `{
  "name": "test-package",
  "version": "1.0.0",
  "scripts": {
    "test": "echo 'test'",
    "build": "echo 'build'"
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	adapter := NewNpmAdapter()

	// 1. Detect should succeed
	detected, err := adapter.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true for directory with package.json")
	}

	// 2. GetTargets should parse the scripts
	targets, err := adapter.GetTargets(tmpDir)
	if err != nil {
		t.Fatalf("GetTargets() failed: %v", err)
	}
	if len(targets) != 2 {
		t.Errorf("GetTargets() returned %d targets, want 2", len(targets))
	}

	// Verify target names
	targetNames := make(map[string]bool)
	for _, target := range targets {
		targetNames[target.Name] = true
	}
	if !targetNames["test"] || !targetNames["build"] {
		t.Error("GetTargets() didn't return expected scripts")
	}

	// 3. Build with non-existent npm should fail gracefully
	adapter.npmBin = "nonexistent-npm-binary"
	_, err = adapter.Build(tmpDir, "test", nil)
	if err == nil {
		t.Error("Build() should fail when npm is not found")
	}
}

// TestIntegration_CargoWorkflow tests a complete cargo workflow
func TestIntegration_CargoWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal Cargo.toml
	cargoToml := `[package]
name = "test-package"
version = "0.1.0"
edition = "2021"

[dependencies]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatalf("Failed to create Cargo.toml: %v", err)
	}

	adapter := NewCargoAdapter()

	// 1. Detect should succeed
	detected, err := adapter.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}
	if !detected {
		t.Error("Detect() = false, want true for directory with Cargo.toml")
	}

	// 2. GetTargets will fail without cargo installed (expected)
	_, err = adapter.GetTargets(tmpDir)
	if err == nil {
		t.Log("GetTargets succeeded (cargo is installed)")
	} else {
		// Expected: cargo not found error
		if err.Error() == "" {
			t.Error("GetTargets error message is empty")
		}
	}
}

// TestIntegration_Registry tests the registry with multiple adapters
func TestIntegration_Registry(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files for multiple build systems
	os.WriteFile(filepath.Join(tmpDir, "WORKSPACE"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte("[package]\nname=\"test\""), 0644)

	// Create registry with all adapters
	registry := NewRegistry()
	registry.Register(NewBazelAdapter())
	registry.Register(NewNpmAdapter())
	registry.Register(NewCargoAdapter())

	// Detect should find all three
	detected, err := registry.Detect(tmpDir)
	if err != nil {
		t.Fatalf("Registry.Detect() failed: %v", err)
	}

	if len(detected) != 3 {
		t.Errorf("Registry.Detect() found %d adapters, want 3", len(detected))
	}

	// Verify we got the right adapters
	adapterNames := make(map[string]bool)
	for _, adapter := range detected {
		adapterNames[adapter.Name()] = true
	}

	expectedAdapters := []string{"bazel", "npm", "cargo"}
	for _, name := range expectedAdapters {
		if !adapterNames[name] {
			t.Errorf("Registry.Detect() didn't find %s adapter", name)
		}
	}

	// Test GetAdapter
	for _, name := range expectedAdapters {
		adapter, err := registry.GetAdapter(name)
		if err != nil {
			t.Errorf("GetAdapter(%q) failed: %v", name, err)
		}
		if adapter.Name() != name {
			t.Errorf("GetAdapter(%q) returned adapter with name %q", name, adapter.Name())
		}
	}
}

// TestIntegration_BuildOptionsTimeout tests that timeout is respected
func TestIntegration_BuildOptionsTimeout(t *testing.T) {
	// This test uses a mock adapter that sleeps longer than timeout
	adapter := NewBazelAdapter()
	// Point to a command that will hang for testing
	adapter.bazelBin = "sleep"

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "WORKSPACE"), []byte(""), 0644)

	opts := &BuildOptions{
		Timeout: 100 * time.Millisecond, // Very short timeout
	}

	start := time.Now()
	_, err := adapter.Build(tmpDir, "10", opts) // sleep 10 seconds
	duration := time.Since(start)

	// Should fail with timeout
	if err == nil {
		t.Error("Build() should timeout")
	}

	// Should not take longer than timeout + reasonable overhead
	if duration > 1*time.Second {
		t.Errorf("Build() took %v, expected to timeout around %v", duration, opts.Timeout)
	}
}

// TestIntegration_ConcurrentBuilds tests thread safety
func TestIntegration_ConcurrentBuilds(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
		"name": "test",
		"scripts": {"test": "echo test"}
	}`), 0644)

	adapter := NewNpmAdapter()
	adapter.npmBin = "echo" // Use echo so it succeeds quickly

	// Run multiple builds concurrently
	const numBuilds = 10
	done := make(chan bool, numBuilds)
	errors := make(chan error, numBuilds)

	for i := 0; i < numBuilds; i++ {
		go func() {
			_, err := adapter.Build(tmpDir, "test", nil)
			if err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < numBuilds; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent build failed: %v", err)
		}
	}
}

// TestIntegration_ResultMetricsChain tests that Result methods work together
func TestIntegration_ResultMetricsChain(t *testing.T) {
	result := &Result{
		Success:      true,
		Duration:     5 * time.Second,
		CacheHits:    75,
		CacheMisses:  25,
		TargetsBuilt: 100,
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(5 * time.Second),
	}

	// Test that cache hit rate is calculated correctly
	rate := result.CacheHitRate()
	expectedRate := 75.0
	if rate != expectedRate {
		t.Errorf("CacheHitRate() = %v, want %v", rate, expectedRate)
	}

	// Verify all fields are accessible
	if !result.Success {
		t.Error("Success field not set correctly")
	}
	if result.Duration != 5*time.Second {
		t.Error("Duration field not set correctly")
	}
	if result.TargetsBuilt != 100 {
		t.Error("TargetsBuilt field not set correctly")
	}
	if result.StartTime.After(result.EndTime) {
		t.Error("StartTime is after EndTime")
	}
}

// TestIntegration_TargetDependencies tests that target dependencies are preserved
func TestIntegration_TargetDependencies(t *testing.T) {
	target := Target{
		ID:           "//app:main",
		Name:         "main",
		Type:         TargetTypeBinary,
		Description:  "Main application",
		Dependencies: []string{"//lib:core", "//lib:utils"},
		Tags:         []string{"production", "critical"},
	}

	// Verify all fields are preserved
	if target.ID != "//app:main" {
		t.Error("ID not preserved")
	}
	if len(target.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(target.Dependencies))
	}
	if len(target.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(target.Tags))
	}

	// Verify dependencies can be iterated
	depMap := make(map[string]bool)
	for _, dep := range target.Dependencies {
		depMap[dep] = true
	}
	if !depMap["//lib:core"] || !depMap["//lib:utils"] {
		t.Error("Dependencies not accessible correctly")
	}
}

// TestIntegration_ErrorPropagation tests that errors are properly propagated
func TestIntegration_ErrorPropagation(t *testing.T) {
	adapter := NewBazelAdapter()
	adapter.bazelBin = "nonexistent-binary-that-does-not-exist"

	// Detect should succeed (only checks files)
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "WORKSPACE"), []byte(""), 0644)

	detected, err := adapter.Detect(tmpDir)
	if err != nil {
		t.Errorf("Detect should not fail on missing binary: %v", err)
	}
	if !detected {
		t.Error("Detect should succeed when WORKSPACE exists")
	}

	// GetTargets should fail with clear error
	_, err = adapter.GetTargets(tmpDir)
	if err == nil {
		t.Error("GetTargets should fail when binary doesn't exist")
	}
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}

	// Build should also fail with clear error
	_, err = adapter.Build(tmpDir, "//test:target", nil)
	if err == nil {
		t.Error("Build should fail when binary doesn't exist")
	}
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}
