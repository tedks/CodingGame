package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkExtractGoSmall benchmarks extraction on a small project (10 files).
func BenchmarkExtractGoSmall(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 2, 5, 3) // 2 packages, 5 files each, 3 imports per file
	defer os.RemoveAll(tempDir)

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		b.Fatalf("NewExtractor failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractGo()
		if err != nil {
			b.Fatalf("ExtractGo failed: %v", err)
		}
	}
}

// BenchmarkExtractGoMedium benchmarks extraction on a medium project (100 files).
func BenchmarkExtractGoMedium(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 10, 10, 5) // 10 packages, 10 files each, 5 imports per file
	defer os.RemoveAll(tempDir)

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		b.Fatalf("NewExtractor failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractGo()
		if err != nil {
			b.Fatalf("ExtractGo failed: %v", err)
		}
	}
}

// BenchmarkExtractGoLarge benchmarks extraction on a large project (1000 files).
func BenchmarkExtractGoLarge(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 100, 10, 5) // 100 packages, 10 files each, 5 imports per file
	defer os.RemoveAll(tempDir)

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		b.Fatalf("NewExtractor failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractGo()
		if err != nil {
			b.Fatalf("ExtractGo failed: %v", err)
		}
	}
}

// BenchmarkExtractGoWithSymbolsSmall benchmarks symbol-level extraction on a small project.
func BenchmarkExtractGoWithSymbolsSmall(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 2, 5, 3)
	defer os.RemoveAll(tempDir)

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		b.Fatalf("NewExtractor failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractGoWithSymbols()
		if err != nil {
			b.Fatalf("ExtractGoWithSymbols failed: %v", err)
		}
	}
}

// BenchmarkExtractGoWithSymbolsMedium benchmarks symbol-level extraction on a medium project.
func BenchmarkExtractGoWithSymbolsMedium(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 10, 10, 5)
	defer os.RemoveAll(tempDir)

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		b.Fatalf("NewExtractor failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractGoWithSymbols()
		if err != nil {
			b.Fatalf("ExtractGoWithSymbols failed: %v", err)
		}
	}
}

// setupBenchmarkProject creates a synthetic project with the specified structure.
// Returns the temp directory path.
func setupBenchmarkProject(b *testing.B, numPackages, filesPerPackage, importsPerFile int) string {
	b.Helper()

	tempDir, err := os.MkdirTemp("", "benchmark_extract")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}

	// Write go.mod
	goModContent := "module benchmark.test\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("failed to write go.mod: %v", err)
	}

	// Create packages
	packageNames := make([]string, numPackages)
	for i := 0; i < numPackages; i++ {
		pkgName := fmt.Sprintf("pkg%d", i)
		packageNames[i] = pkgName
		pkgDir := filepath.Join(tempDir, "internal", pkgName)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			os.RemoveAll(tempDir)
			b.Fatalf("failed to create package dir: %v", err)
		}

		// Create files in each package
		for j := 0; j < filesPerPackage; j++ {
			fileName := fmt.Sprintf("file%d.go", j)
			content := generateGoFile(pkgName, i, j, packageNames, importsPerFile)
			if err := os.WriteFile(filepath.Join(pkgDir, fileName), []byte(content), 0644); err != nil {
				os.RemoveAll(tempDir)
				b.Fatalf("failed to write go file: %v", err)
			}
		}
	}

	return tempDir
}

// generateGoFile creates a synthetic Go file with imports.
func generateGoFile(pkgName string, pkgIndex, fileIndex int, allPackages []string, numImports int) string {
	content := fmt.Sprintf("package %s\n\nimport (\n", pkgName)

	// Add external imports
	externals := []string{"fmt", "strings", "bytes", "io", "os", "context", "sync", "time"}
	importCount := 0

	for _, ext := range externals {
		if importCount >= numImports {
			break
		}
		content += fmt.Sprintf("\t\"%s\"\n", ext)
		importCount++
	}

	// Add internal imports (from other packages, avoiding self-import)
	for i, pkg := range allPackages {
		if importCount >= numImports {
			break
		}
		if i == pkgIndex {
			continue // skip self
		}
		content += fmt.Sprintf("\t\"benchmark.test/internal/%s\"\n", pkg)
		importCount++
	}

	content += ")\n\n"

	// Generate functions that use the imports
	content += fmt.Sprintf("func File%d() {\n", fileIndex)
	content += "\t_ = fmt.Sprint()\n"
	content += "\t_ = strings.ToLower(\"x\")\n"

	// Reference internal packages if imported
	for i, pkg := range allPackages {
		if i == pkgIndex || i >= numImports-len(externals) {
			continue
		}
		content += fmt.Sprintf("\t_ = %s.File0()\n", pkg)
	}

	content += "}\n"

	return content
}

// BenchmarkMemoryExtractGoLarge measures memory allocation for large projects.
func BenchmarkMemoryExtractGoLarge(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 100, 10, 5)
	defer os.RemoveAll(tempDir)

	extractor, err := NewExtractor(tempDir)
	if err != nil {
		b.Fatalf("NewExtractor failed: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := extractor.ExtractGo()
		if err != nil {
			b.Fatalf("ExtractGo failed: %v", err)
		}
	}
}
