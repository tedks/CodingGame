package dependency

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzExtractGo tests that the extractor doesn't panic on malformed Go source.
// This is the first fuzz test in the project.
//
// Run with: go test -fuzz=FuzzExtractGo -fuzztime=30s
func FuzzExtractGo(f *testing.F) {
	// Add seed corpus
	f.Add([]byte("package x\nimport \"fmt\"\n"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\x01\x02"))
	f.Add([]byte("package x\nimport (\n"))                                                // unclosed
	f.Add([]byte("package x\nimport \""))                                                 // unclosed string
	f.Add([]byte("package x\n/* unclosed comment"))                                       // unclosed comment
	f.Add([]byte("package x\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n"))                        // valid multi-import
	f.Add([]byte("package x\nimport . \"fmt\"\n"))                                        // dot import
	f.Add([]byte("package x\nimport _ \"image/png\"\n"))                                  // blank import
	f.Add([]byte("package x\nimport alias \"fmt\"\n"))                                    // aliased import
	f.Add([]byte("//go:build ignore\n\npackage x\nimport \"fmt\"\n"))                     // build ignore
	f.Add([]byte("// +build ignore\n\npackage x\nimport \"fmt\"\n"))                      // old-style build ignore
	f.Add([]byte("package 123invalid"))                                                   // invalid package name
	f.Add([]byte("package x\nfunc () {}"))                                                // invalid function
	f.Add([]byte("package x\nimport \"very/long/" + string(make([]byte, 1000)) + "\"\n")) // long import

	f.Fuzz(func(t *testing.T, data []byte) {
		tempDir, err := os.MkdirTemp("", "fuzz_extract")
		if err != nil {
			t.Skip("cannot create temp dir")
		}
		defer os.RemoveAll(tempDir)

		// Write go.mod
		goModContent := "module fuzz.test\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Skip("cannot write go.mod")
		}

		// Write the fuzzed data as a Go file
		goFilePath := filepath.Join(tempDir, "fuzz.go")
		if err := os.WriteFile(goFilePath, data, 0644); err != nil {
			t.Skip("cannot write go file")
		}

		// Create extractor - should not panic
		extractor, err := NewExtractor(tempDir)
		if err != nil {
			// Errors are acceptable, panics are not
			return
		}

		// Extract - should not panic regardless of input
		_, _ = extractor.ExtractGo()
		_, _ = extractor.ExtractGoWithSymbols()
	})
}

// FuzzGoModParsing tests that go.mod parsing doesn't panic on malformed input.
func FuzzGoModParsing(f *testing.F) {
	// Add seed corpus
	f.Add([]byte("module foo\n"))
	f.Add([]byte(""))
	f.Add([]byte("module\n"))
	f.Add([]byte("module "))
	f.Add([]byte("module foo // comment\n"))
	f.Add([]byte("module foo\r\ngo 1.21\r\n"))
	f.Add([]byte("\xef\xbb\xbfmodule foo\n")) // UTF-8 BOM
	f.Add([]byte("module foo\nreplace foo => ../bar\n"))
	f.Add([]byte("module foo\nrequire (\n\tbar v1.0.0\n)\n"))
	f.Add([]byte(string(make([]byte, 10000)))) // large junk

	f.Fuzz(func(t *testing.T, data []byte) {
		tempDir, err := os.MkdirTemp("", "fuzz_gomod")
		if err != nil {
			t.Skip("cannot create temp dir")
		}
		defer os.RemoveAll(tempDir)

		// Write fuzzed data as go.mod
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), data, 0644); err != nil {
			t.Skip("cannot write go.mod")
		}

		// Should not panic
		_, _ = detectGoModulePath(tempDir)
	})
}
