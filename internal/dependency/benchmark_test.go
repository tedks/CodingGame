package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkExtractGoSmall(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 2, 5, 3)
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

func BenchmarkExtractGoMedium(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 10, 10, 5)
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

func BenchmarkExtractGoLarge(b *testing.B) {
	tempDir := setupBenchmarkProject(b, 100, 10, 5)
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

func setupBenchmarkProject(b *testing.B, numPackages, filesPerPackage, importsPerFile int) string {
	b.Helper()

	tempDir, err := os.MkdirTemp("", "benchmark_extract")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}

	goModContent := "module benchmark.test\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		b.Fatalf("failed to write go.mod: %v", err)
	}

	packageNames := make([]string, numPackages)
	for i := 0; i < numPackages; i++ {
		pkgName := fmt.Sprintf("pkg%d", i)
		packageNames[i] = pkgName
		pkgDir := filepath.Join(tempDir, "internal", pkgName)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			os.RemoveAll(tempDir)
			b.Fatalf("failed to create package dir: %v", err)
		}

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

func generateGoFile(pkgName string, pkgIndex, fileIndex int, allPackages []string, numImports int) string {
	content := fmt.Sprintf("package %s\n\nimport (\n", pkgName)

	externals := []string{"fmt", "strings", "bytes", "io", "os"}
	importCount := 0

	for _, ext := range externals {
		if importCount >= numImports {
			break
		}
		content += fmt.Sprintf("\t\"%s\"\n", ext)
		importCount++
	}

	for i, pkg := range allPackages {
		if importCount >= numImports {
			break
		}
		if i == pkgIndex {
			continue
		}
		content += fmt.Sprintf("\t\"benchmark.test/internal/%s\"\n", pkg)
		importCount++
	}

	content += ")\n\n"
	content += fmt.Sprintf("func File%d() {\n", fileIndex)
	content += "\t_ = fmt.Sprint()\n"
	content += "\t_ = strings.ToLower(\"x\")\n"
	content += "}\n"

	return content
}
