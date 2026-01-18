// Package dependency provides extraction of dependency relationships from source code.
// It parses import statements and build files to create a connection graph.
//
// Currently supported:
// - Go: Parse import statements from .go files
//
// Future:
// - TypeScript: Parse import/require statements
// - Python: Parse import statements
// - LSP integration for more accurate relationships
package dependency

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tedks/CodingGame/internal/connection"
)

// Extractor extracts dependencies from source code.
type Extractor struct {
	projectRoot string
	modulePath  string // Go module path (e.g., "github.com/tedks/CodingGame")
}

// NewExtractor creates a new dependency extractor for a project.
//
// Parameters:
//   - projectRoot: Absolute path to the project root directory
//
// The extractor will attempt to detect the Go module path from go.mod.
func NewExtractor(projectRoot string) (*Extractor, error) {
	modulePath, _ := detectGoModulePath(projectRoot)
	return &Extractor{
		projectRoot: projectRoot,
		modulePath:  modulePath,
	}, nil
}

// detectGoModulePath reads the module path from go.mod.
func detectGoModulePath(projectRoot string) (string, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	// Simple parsing: find "module <path>" line
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", nil
}

// ExtractGo extracts Go import dependencies from the project.
//
// Returns a connection graph with:
// - TypeImport connections for direct imports
// - Relative paths matching MapView tile paths
//
// Assumptions:
// - Only parses .go files (not build constraints)
// - Only tracks imports within the same module (external deps are marked as external)
// - Self-imports (within the same package) are tracked but flagged
func (e *Extractor) ExtractGo() (*connection.Graph, error) {
	graph := connection.NewGraph()

	// Walk all Go files in the project
	err := filepath.Walk(e.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories we don't want to scan
		baseName := filepath.Base(path)
		if info.IsDir() {
			if baseName == ".git" || baseName == "vendor" || baseName == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files for dependency graph (they add noise)
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the file
		if err := e.extractFileImports(path, graph); err != nil {
			// Log but don't fail - allow partial graphs
			log.Printf("Warning: failed to parse %s: %v", path, err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Detect circular dependencies
	graph.DetectCircular()

	return graph, nil
}

// extractFileImports parses a single Go file and adds its imports to the graph.
func (e *Extractor) extractFileImports(filePath string, graph *connection.Graph) error {
	// Get relative path for this file
	relPath, err := filepath.Rel(e.projectRoot, filePath)
	if err != nil {
		return err
	}
	relPath = filepath.ToSlash(relPath)

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}

	// Extract imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check if this is an internal import
		if e.modulePath != "" && strings.HasPrefix(importPath, e.modulePath) {
			// Convert module import to relative path
			internalPath := strings.TrimPrefix(importPath, e.modulePath+"/")

			// Find the actual file(s) in this package
			// For simplicity, we use the package directory as the connection target
			targetDir := filepath.ToSlash(internalPath)

			conn := graph.AddNew(targetDir, relPath, connection.TypeImport)

			// Estimate coupling strength based on import usage
			// (A more sophisticated approach would count symbol usages)
			conn.SetStrength(1)
		} else {
			// External import - still track but mark as external
			conn := graph.AddNew(importPath, relPath, connection.TypeImport)
			conn.SetExternal(true)
		}
	}
	return nil
}

// ExtractGoWithSymbols extracts Go dependencies with symbol-level detail.
// This provides more accurate coupling strength by counting symbol usages.
//
// This is more expensive than ExtractGo as it does full AST parsing.
func (e *Extractor) ExtractGoWithSymbols() (*connection.Graph, error) {
	graph := connection.NewGraph()

	// Map from package path to list of files
	packageFiles := make(map[string][]string)

	// First pass: collect all files by package
	err := filepath.Walk(e.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		baseName := filepath.Base(path)
		if info.IsDir() {
			if baseName == ".git" || baseName == "vendor" || baseName == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relPath, _ := filepath.Rel(e.projectRoot, path)
		pkgDir := filepath.Dir(relPath)
		packageFiles[pkgDir] = append(packageFiles[pkgDir], path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: parse each package and analyze symbol usage
	fset := token.NewFileSet()
	for pkgDir, files := range packageFiles {
		// Parse all files in the package
		pkgs, err := parser.ParseDir(fset, filepath.Join(e.projectRoot, pkgDir), nil, parser.ParseComments)
		if err != nil {
			continue
		}

		for _, pkg := range pkgs {
			e.extractPackageSymbolUsage(pkg, pkgDir, graph)
		}
		_ = files // Used for reference
	}

	graph.DetectCircular()
	return graph, nil
}

// extractPackageSymbolUsage analyzes symbol usage within a package.
func (e *Extractor) extractPackageSymbolUsage(pkg *ast.Package, pkgDir string, graph *connection.Graph) {
	pkgPath := filepath.ToSlash(pkgDir)

	// Track imports and their aliases
	importAliases := make(map[string]string) // alias -> import path

	for fileName, file := range pkg.Files {
		// Clear aliases for each file
		for k := range importAliases {
			delete(importAliases, k)
		}

		// Collect imports
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				// Default alias is the last component of the import path
				parts := strings.Split(importPath, "/")
				alias = parts[len(parts)-1]
			}
			importAliases[alias] = importPath
		}

		// Count symbol usages per import
		usageCount := make(map[string]int)

		ast.Inspect(file, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if importPath, exists := importAliases[ident.Name]; exists {
						usageCount[importPath]++
					}
				}
			}
			return true
		})

		// Create connections with strength based on usage
		relPath, _ := filepath.Rel(e.projectRoot, fileName)
		relPath = filepath.ToSlash(relPath)

		for importPath, count := range usageCount {
			if e.modulePath != "" && strings.HasPrefix(importPath, e.modulePath) {
				internalPath := strings.TrimPrefix(importPath, e.modulePath+"/")
				conn := graph.AddNew(filepath.ToSlash(internalPath), relPath, connection.TypeImport)
				conn.SetStrength(count)
			} else {
				conn := graph.AddNew(importPath, relPath, connection.TypeImport)
				conn.SetExternal(true)
				conn.SetStrength(count)
			}
		}

		_ = pkgPath // Available for additional analysis
	}
}

// ProjectRoot returns the project root path.
func (e *Extractor) ProjectRoot() string {
	return e.projectRoot
}

// ModulePath returns the Go module path.
func (e *Extractor) ModulePath() string {
	return e.modulePath
}
