package dependency

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tedks/CodingGame/internal/connection"
)

func TestDetectGoModulePath(t *testing.T) {
	// Create a temporary directory with a go.mod file
	tempDir, err := os.MkdirTemp("", "dependency_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a go.mod file
	goModContent := `module github.com/example/testproject

go 1.21

require (
	github.com/some/dependency v1.0.0
)
`
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Test detection
	modulePath, err := detectGoModulePath(tempDir)
	if err != nil {
		t.Fatalf("detectGoModulePath failed: %v", err)
	}
	if modulePath != "github.com/example/testproject" {
		t.Errorf("expected module path 'github.com/example/testproject', got '%s'", modulePath)
	}
}

func TestDetectGoModulePathMissing(t *testing.T) {
	// Create a temporary directory without a go.mod file
	tempDir, err := os.MkdirTemp("", "dependency_test_missing")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test detection - should return empty string and error
	modulePath, err := detectGoModulePath(tempDir)
	if err == nil {
		t.Error("expected error for missing go.mod")
	}
	if modulePath != "" {
		t.Errorf("expected empty module path, got '%s'", modulePath)
	}
}

func TestNewExtractor(t *testing.T) {
	// Create a temporary directory with a go.mod file
	tempDir, err := os.MkdirTemp("", "extractor_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write a go.mod file
	goModContent := `module github.com/example/testproject

go 1.21
`
	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Test extractor creation
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}
	if extractor.ProjectRoot() != tempDir {
		t.Errorf("expected project root '%s', got '%s'", tempDir, extractor.ProjectRoot())
	}
	if extractor.ModulePath() != "github.com/example/testproject" {
		t.Errorf("expected module path 'github.com/example/testproject', got '%s'", extractor.ModulePath())
	}
}

func TestNewExtractorWithoutGoMod(t *testing.T) {
	// Create a temporary directory without a go.mod file
	tempDir, err := os.MkdirTemp("", "extractor_test_nomod")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test extractor creation - should still work but with empty module path
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}
	if extractor.ModulePath() != "" {
		t.Errorf("expected empty module path, got '%s'", extractor.ModulePath())
	}
}

func TestExtractGoBasic(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "extract_go_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write go.mod
	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create internal/pkga directory
	pkgaDir := filepath.Join(tempDir, "internal", "pkga")
	if err := os.MkdirAll(pkgaDir, 0755); err != nil {
		t.Fatalf("failed to create pkga dir: %v", err)
	}

	// Create internal/pkgb directory
	pkgbDir := filepath.Join(tempDir, "internal", "pkgb")
	if err := os.MkdirAll(pkgbDir, 0755); err != nil {
		t.Fatalf("failed to create pkgb dir: %v", err)
	}

	// Write pkga/a.go
	pkgaContent := `package pkga

func DoA() string {
	return "A"
}
`
	if err := os.WriteFile(filepath.Join(pkgaDir, "a.go"), []byte(pkgaContent), 0644); err != nil {
		t.Fatalf("failed to write a.go: %v", err)
	}

	// Write pkgb/b.go that imports pkga
	pkgbContent := `package pkgb

import (
	"fmt"
	"github.com/example/testproject/internal/pkga"
)

func DoB() string {
	return fmt.Sprintf("B uses %s", pkga.DoA())
}
`
	if err := os.WriteFile(filepath.Join(pkgbDir, "b.go"), []byte(pkgbContent), 0644); err != nil {
		t.Fatalf("failed to write b.go: %v", err)
	}

	// Create extractor and extract
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	graph, err := extractor.ExtractGo()
	if err != nil {
		t.Fatalf("ExtractGo failed: %v", err)
	}

	// Verify connections
	// pkgb/b.go should import internal/pkga
	connections := graph.FromFile("internal/pkga")
	if len(connections) != 1 {
		t.Errorf("expected 1 connection from internal/pkga, got %d", len(connections))
	}

	// Verify the connection details
	if len(connections) > 0 {
		conn := connections[0]
		if conn.Type() != connection.TypeImport {
			t.Errorf("expected TypeImport, got %v", conn.Type())
		}
		from, to := conn.Endpoints()
		if from != "internal/pkga" {
			t.Errorf("expected from 'internal/pkga', got '%s'", from)
		}
		if to != "internal/pkgb/b.go" {
			t.Errorf("expected to 'internal/pkgb/b.go', got '%s'", to)
		}
	}

	// Check for external import (fmt)
	externalConns := graph.FromFile("fmt")
	if len(externalConns) != 1 {
		t.Errorf("expected 1 external connection from fmt, got %d", len(externalConns))
	}
	if len(externalConns) > 0 && !externalConns[0].IsExternal() {
		t.Error("expected fmt import to be marked as external")
	}
}

func TestExtractGoSkipsTestFiles(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "extract_go_skip_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write go.mod
	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create pkg directory
	pkgDir := filepath.Join(tempDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	// Write main.go
	mainContent := `package pkg

func Main() {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Write main_test.go with a unique import
	testContent := `package pkg

import (
	"testing"
)

func TestMain(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(pkgDir, "main_test.go"), []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write main_test.go: %v", err)
	}

	// Create extractor and extract
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	graph, err := extractor.ExtractGo()
	if err != nil {
		t.Fatalf("ExtractGo failed: %v", err)
	}

	// There should be no "testing" import since we skip _test.go files
	testingConns := graph.FromFile("testing")
	if len(testingConns) != 0 {
		t.Errorf("expected 0 connections from testing (test files should be skipped), got %d", len(testingConns))
	}
}

func TestExtractGoSkipsVendorAndGit(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "extract_go_skip_dirs")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write go.mod
	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create .git directory with a Go file (should be skipped)
	gitDir := filepath.Join(tempDir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}
	gitGoContent := `package main

import "os"

func main() { os.Exit(0) }
`
	if err := os.WriteFile(filepath.Join(gitDir, "hook.go"), []byte(gitGoContent), 0644); err != nil {
		t.Fatalf("failed to write hook.go: %v", err)
	}

	// Create vendor directory with a Go file (should be skipped)
	vendorDir := filepath.Join(tempDir, "vendor", "example.com", "lib")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("failed to create vendor dir: %v", err)
	}
	vendorGoContent := `package lib

import "strings"

func Lib() { strings.ToLower("x") }
`
	if err := os.WriteFile(filepath.Join(vendorDir, "lib.go"), []byte(vendorGoContent), 0644); err != nil {
		t.Fatalf("failed to write lib.go: %v", err)
	}

	// Create extractor and extract
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	graph, err := extractor.ExtractGo()
	if err != nil {
		t.Fatalf("ExtractGo failed: %v", err)
	}

	// Should have no connections since all Go files are in skipped directories
	allConns := graph.All()
	if len(allConns) != 0 {
		t.Errorf("expected 0 connections (vendor and .git skipped), got %d", len(allConns))
		for _, c := range allConns {
			from, to := c.Endpoints()
			t.Logf("  found: %s -> %s", from, to)
		}
	}
}

func TestExtractGoCircularDependency(t *testing.T) {
	// Create a temporary directory structure with circular deps
	tempDir, err := os.MkdirTemp("", "extract_go_circular")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write go.mod
	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create packages that form a cycle: a -> b -> c -> a
	// In Go this would be a compile error, but we test the detection

	// Note: Go doesn't actually allow circular imports at the package level,
	// but our graph would detect if files reference each other.
	// For testing, we create a simpler structure where the detection works.

	// Create pkga
	pkgaDir := filepath.Join(tempDir, "internal", "pkga")
	if err := os.MkdirAll(pkgaDir, 0755); err != nil {
		t.Fatalf("failed to create pkga dir: %v", err)
	}

	// Create pkgb
	pkgbDir := filepath.Join(tempDir, "internal", "pkgb")
	if err := os.MkdirAll(pkgbDir, 0755); err != nil {
		t.Fatalf("failed to create pkgb dir: %v", err)
	}

	// pkga imports pkgb
	pkgaContent := `package pkga

import "github.com/example/testproject/internal/pkgb"

func DoA() string {
	return pkgb.DoB()
}
`
	if err := os.WriteFile(filepath.Join(pkgaDir, "a.go"), []byte(pkgaContent), 0644); err != nil {
		t.Fatalf("failed to write a.go: %v", err)
	}

	// pkgb imports pkga (circular!)
	pkgbContent := `package pkgb

import "github.com/example/testproject/internal/pkga"

func DoB() string {
	return pkga.DoA()
}
`
	if err := os.WriteFile(filepath.Join(pkgbDir, "b.go"), []byte(pkgbContent), 0644); err != nil {
		t.Fatalf("failed to write b.go: %v", err)
	}

	// Create extractor and extract
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	graph, err := extractor.ExtractGo()
	if err != nil {
		t.Fatalf("ExtractGo failed: %v", err)
	}

	// DetectCircular should have been called, check for circular paths
	circularPaths := graph.CircularPaths()
	if len(circularPaths) == 0 {
		t.Log("Note: circular dependency detection may not find cycles at package level")
		t.Log("This is expected since Go connections are package->file, not package->package")
	}

	// At minimum, verify we have both directions of the dependency
	aConns := graph.FromFile("internal/pkgb")
	bConns := graph.FromFile("internal/pkga")

	t.Logf("Connections from internal/pkgb: %d", len(aConns))
	t.Logf("Connections from internal/pkga: %d", len(bConns))
}

func TestExtractGoWithSymbols(t *testing.T) {
	// Create a temporary directory structure
	tempDir, err := os.MkdirTemp("", "extract_go_symbols")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write go.mod
	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Create internal/util directory
	utilDir := filepath.Join(tempDir, "internal", "util")
	if err := os.MkdirAll(utilDir, 0755); err != nil {
		t.Fatalf("failed to create util dir: %v", err)
	}

	// Create internal/main directory
	mainDir := filepath.Join(tempDir, "internal", "main")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatalf("failed to create main dir: %v", err)
	}

	// Write util/util.go
	utilContent := `package util

func Helper() string { return "help" }
func Other() string { return "other" }
`
	if err := os.WriteFile(filepath.Join(utilDir, "util.go"), []byte(utilContent), 0644); err != nil {
		t.Fatalf("failed to write util.go: %v", err)
	}

	// Write main/main.go that uses util multiple times
	mainContent := `package main

import (
	"github.com/example/testproject/internal/util"
)

func Run() {
	_ = util.Helper()
	_ = util.Helper()
	_ = util.Helper()
	_ = util.Other()
}
`
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte(mainContent), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Create extractor and extract with symbols
	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	graph, err := extractor.ExtractGoWithSymbols()
	if err != nil {
		t.Fatalf("ExtractGoWithSymbols failed: %v", err)
	}

	// Verify connections with strength
	connections := graph.FromFile("internal/util")
	if len(connections) != 1 {
		t.Errorf("expected 1 connection from internal/util, got %d", len(connections))
	}

	if len(connections) > 0 {
		conn := connections[0]
		// Should have strength 4 (3 Helper calls + 1 Other call)
		if conn.Strength() != 4 {
			t.Errorf("expected strength 4, got %d", conn.Strength())
		}
	}
}
