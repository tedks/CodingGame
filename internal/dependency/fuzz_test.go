package dependency

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzExtractGo tests that the extractor doesn't panic on malformed Go source.
func FuzzExtractGo(f *testing.F) {
	f.Add([]byte("package x\nimport \"fmt\"\n"))
	f.Add([]byte(""))
	f.Add([]byte("\x00\x01\x02"))
	f.Add([]byte("package x\nimport (\n"))
	f.Add([]byte("package x\nimport \""))
	f.Add([]byte("package x\n/* unclosed comment"))
	f.Add([]byte("package x\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n"))
	f.Add([]byte("package x\nimport . \"fmt\"\n"))
	f.Add([]byte("package x\nimport _ \"image/png\"\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tempDir, err := os.MkdirTemp("", "fuzz_extract")
		if err != nil {
			t.Skip("cannot create temp dir")
		}
		defer os.RemoveAll(tempDir)

		goModContent := "module fuzz.test\n\ngo 1.21\n"
		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
			t.Skip("cannot write go.mod")
		}

		if err := os.WriteFile(filepath.Join(tempDir, "fuzz.go"), data, 0644); err != nil {
			t.Skip("cannot write go file")
		}

		extractor, err := NewExtractor(tempDir)
		if err != nil {
			return
		}

		_, _ = extractor.ExtractGo()
		_, _ = extractor.ExtractGoWithSymbols()
	})
}

// FuzzGoModParsing tests that go.mod parsing doesn't panic on malformed input.
func FuzzGoModParsing(f *testing.F) {
	f.Add([]byte("module foo\n"))
	f.Add([]byte(""))
	f.Add([]byte("module\n"))
	f.Add([]byte("module foo // comment\n"))
	f.Add([]byte("module foo\r\ngo 1.21\r\n"))
	f.Add([]byte("\xef\xbb\xbfmodule foo\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		tempDir, err := os.MkdirTemp("", "fuzz_gomod")
		if err != nil {
			t.Skip("cannot create temp dir")
		}
		defer os.RemoveAll(tempDir)

		if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), data, 0644); err != nil {
			t.Skip("cannot write go.mod")
		}

		_, _ = detectGoModulePath(tempDir)
	})
}
