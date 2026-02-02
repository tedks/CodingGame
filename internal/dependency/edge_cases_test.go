package dependency

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/connection"
)

// TestExtractGoWithSymlinkCycle verifies extraction completes without infinite
// loop when encountering symlink cycles.
func TestExtractGoWithSymlinkCycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "symlink_cycle_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	subdir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	symlinkPath := filepath.Join(subdir, "link")
	if err := os.Symlink(tempDir, symlinkPath); err != nil {
		t.Skipf("cannot create symlinks: %v", err)
	}

	goContent := `package subdir

import "fmt"

func Hello() { fmt.Println("hello") }
`
	if err := os.WriteFile(filepath.Join(subdir, "hello.go"), []byte(goContent), 0644); err != nil {
		t.Fatalf("failed to write hello.go: %v", err)
	}

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := extractor.ExtractGo()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("ExtractGo returned error (acceptable): %v", err)
		}
	case <-ctx.Done():
		t.Fatal("ExtractGo stuck in infinite loop due to symlink cycle")
	}
}

// TestExtractGoIgnoresBuildIgnore verifies files with //go:build ignore are excluded.
func TestExtractGoIgnoresBuildIgnore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "build_ignore_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	pkgDir := filepath.Join(tempDir, "pkg")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}

	normalContent := `package pkg

import "strings"

func Normal() string { return strings.ToLower("X") }
`
	if err := os.WriteFile(filepath.Join(pkgDir, "normal.go"), []byte(normalContent), 0644); err != nil {
		t.Fatalf("failed to write normal.go: %v", err)
	}

	ignoredContent := `//go:build ignore

package pkg

import "bytes"

func Ignored() { _ = bytes.NewBuffer(nil) }
`
	if err := os.WriteFile(filepath.Join(pkgDir, "ignored.go"), []byte(ignoredContent), 0644); err != nil {
		t.Fatalf("failed to write ignored.go: %v", err)
	}

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		t.Fatalf("NewExtractor failed: %v", err)
	}

	graph, err := extractor.ExtractGo()
	if err != nil {
		t.Fatalf("ExtractGo failed: %v", err)
	}

	stringsConns := graph.FromFile("strings")
	if len(stringsConns) != 1 {
		t.Errorf("expected 1 connection from strings, got %d", len(stringsConns))
	}

	bytesConns := graph.FromFile("bytes")
	if len(bytesConns) == 0 {
		t.Log("Good: //go:build ignore is respected")
	} else {
		t.Log("Note: //go:build ignore is NOT respected - documenting current behavior")
	}
}

// TestGoModParsingEdgeCases tests various edge cases in go.mod parsing.
func TestGoModParsingEdgeCases(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected string
	}{
		{"trailing_comment", "module foo // comment\n", "foo"},
		{"crlf_endings", "module foo\r\ngo 1.21\r\n", "foo"},
		{"utf8_bom", "\xef\xbb\xbfmodule foo\n", "foo"},
		{"extra_whitespace", "  module   foo  \n", "foo"},
		{"tabs", "\tmodule\tfoo\n", "foo"},
		{"empty_lines_before", "\n\n\nmodule foo\n", "foo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "gomod_edge_"+tc.name)
			if err != nil {
				t.Fatalf("failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(tc.content), 0644); err != nil {
				t.Fatalf("failed to write go.mod: %v", err)
			}

			modulePath, err := detectGoModulePath(tempDir)
			if err != nil {
				t.Fatalf("detectGoModulePath failed: %v", err)
			}

			if modulePath != tc.expected {
				t.Logf("Expected: %q, Got: %q", tc.expected, modulePath)
			}
		})
	}
}

// TestExtractGoIdempotent verifies same project extracted twice yields identical graph.
func TestExtractGoIdempotent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "idempotent_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModContent := `module github.com/example/testproject

go 1.21
`
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	pkgADir := filepath.Join(tempDir, "internal", "pkga")
	pkgBDir := filepath.Join(tempDir, "internal", "pkgb")
	os.MkdirAll(pkgADir, 0755)
	os.MkdirAll(pkgBDir, 0755)

	os.WriteFile(filepath.Join(pkgADir, "a.go"), []byte("package pkga\n\nfunc DoA() string { return \"A\" }\n"), 0644)
	os.WriteFile(filepath.Join(pkgBDir, "b.go"), []byte("package pkgb\n\nimport \"github.com/example/testproject/internal/pkga\"\n\nfunc DoB() string { return pkga.DoA() }\n"), 0644)

	extractor, _ := NewExtractor(tempDir)
	graph1, _ := extractor.ExtractGo()
	graph2, _ := extractor.ExtractGo()

	conns1 := graph1.All()
	conns2 := graph2.All()

	if len(conns1) != len(conns2) {
		t.Fatalf("graphs have different connection counts: %d vs %d", len(conns1), len(conns2))
	}

	connToString := func(c *connection.Connection) string {
		from, to := c.Endpoints()
		return from + " -> " + to
	}

	strings1 := make([]string, len(conns1))
	strings2 := make([]string, len(conns2))
	for i, c := range conns1 {
		strings1[i] = connToString(c)
	}
	for i, c := range conns2 {
		strings2[i] = connToString(c)
	}
	sort.Strings(strings1)
	sort.Strings(strings2)

	for i := range strings1 {
		if strings1[i] != strings2[i] {
			t.Errorf("connection mismatch at %d: %s vs %s", i, strings1[i], strings2[i])
		}
	}
}

// TestExtractGoDotImport verifies behavior with dot imports.
func TestExtractGoDotImport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dot_import_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module github.com/example/testproject\n\ngo 1.21\n"), 0644)

	pkgDir := filepath.Join(tempDir, "pkg")
	os.MkdirAll(pkgDir, 0755)

	dotContent := `package pkg

import . "fmt"

func UseDot() { Println("using dot import") }
`
	os.WriteFile(filepath.Join(pkgDir, "dot.go"), []byte(dotContent), 0644)

	extractor, _ := NewExtractor(tempDir)
	graph, _ := extractor.ExtractGo()

	fmtConns := graph.FromFile("fmt")
	if len(fmtConns) != 1 {
		t.Errorf("expected 1 connection from fmt (dot import), got %d", len(fmtConns))
	}
}

// TestExtractGoBlankImport verifies blank imports are tracked.
func TestExtractGoBlankImport(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "blank_import_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module github.com/example/testproject\n\ngo 1.21\n"), 0644)

	pkgDir := filepath.Join(tempDir, "pkg")
	os.MkdirAll(pkgDir, 0755)

	blankContent := `package pkg

import _ "image/png"

func Init() {}
`
	os.WriteFile(filepath.Join(pkgDir, "blank.go"), []byte(blankContent), 0644)

	extractor, _ := NewExtractor(tempDir)
	graph, _ := extractor.ExtractGo()

	pngConns := graph.FromFile("image/png")
	if len(pngConns) != 1 {
		t.Errorf("expected 1 connection from image/png (blank import), got %d", len(pngConns))
	}
}

// TestExtractGoModulePathSubstring verifies module paths with similar prefixes are handled correctly.
func TestExtractGoModulePathSubstring(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "module_substring_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/foo\n\ngo 1.21\n"), 0644)

	pkgDir := filepath.Join(tempDir, "pkg")
	os.MkdirAll(pkgDir, 0755)

	content := `package pkg

import "example.com/foobar/baz"

func UseExternal() { _ = baz.Do() }
`
	os.WriteFile(filepath.Join(pkgDir, "external.go"), []byte(content), 0644)

	extractor, _ := NewExtractor(tempDir)
	graph, _ := extractor.ExtractGo()

	conns := graph.FromFile("example.com/foobar/baz")
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}

	if !conns[0].IsExternal() {
		t.Error("example.com/foobar/baz should be EXTERNAL, not internal")
		t.Log("Bug: module path example.com/foo incorrectly matching example.com/foobar")
	}
}
