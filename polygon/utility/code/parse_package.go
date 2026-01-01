package code

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func ParsePackage(parser *Parser, path string) (*Package, error) {
	if parser == nil {
		return nil, fmt.Errorf("parser cannot be nil")
	}

	if path == "" {
		return nil, fmt.Errorf("package path cannot be empty")
	}

	// Validate that the path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to access package path %s: %w", path, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("package path %s is not a directory", path)
	}

	// Extract package name from first .go file in the directory
	packageName, err := ParsePackagePackageName(path)
	if err != nil {
		return nil, fmt.Errorf("failed to extract package name: %w", err)
	}

	// Get relative path from module root
	var relativePath *string
	if parser.Module != nil && parser.Module.Path != nil {
		rel, err := filepath.Rel(*parser.Module.Path, path)
		if err != nil {
			log.Printf("warning: failed to get relative path from module root: %v", err)
			relativePath = &path // Use absolute path as fallback
		} else {
			relativePath = &rel
		}
	} else {
		relativePath = &path // No module root available
	}

	// Extract directory name
	dirName := filepath.Base(path)

	// Extract package name (last part of full package name)
	packageNameLast := packageName
	if lastSlash := strings.LastIndex(packageName, "/"); lastSlash >= 0 {
		packageNameLast = packageName[lastSlash+1:]
	}

	// Create package struct
	pkg := &Package{
		Path:          relativePath,
		DirectoryName: &dirName,
		Package:       &packageName,
		PackageName:   &packageNameLast,
		Files:         []*File{},
		Module:        parser.Module,
	}

	// Read directory contents and parse Go files
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		// Skip directories and non-Go files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		// Skip test files
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		file, err := ParsePackageFile(pkg, filePath)
		if err != nil {
			log.Printf("warning: failed to parse file %s: %v", filePath, err)
			continue // Continue with other files
		}

		if file != nil {
			pkg.Files = append(pkg.Files, file)
		}
	}

	return pkg, nil
}

func ParsePackagePackageName(absolutePath string) (string, error) {
	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", absolutePath, err)
	}

	// Find the first .go file (excluding test files) to extract package name
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(absolutePath, entry.Name())

		// Parse the file to extract package name
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, filePath, nil, parser.PackageClauseOnly)
		if err != nil {
			log.Printf("warning: failed to parse file %s: %v", filePath, err)
			continue
		}

		if node.Name != nil {
			return node.Name.Name, nil
		}
	}

	return "", fmt.Errorf("no Go files found in directory %s", absolutePath)
}
