package code

import (
	"context"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon"
)

func ParsePackage(ctx context.Context, parser *Parser, path string) (*Package, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()
	s.Variable("path", path)

	if parser == nil {
		return nil, s.Error("parser cannot be nil", nil)
	}

	if path == "" {
		return nil, s.Error("package path cannot be empty", nil)
	}

	// * validate that the path exists and is a directory
	info, err := os.Stat(path)
	if err != nil {
		return nil, s.Error("failed to access package path", err)
	}

	if !info.IsDir() {
		return nil, s.Error("package path is not a directory", nil)
	}

	// * extract package name from first .go file in the directory
	packageName, err := ParsePackagePackageName(ctx, path)
	if err != nil {
		return nil, s.Error("failed to extract package name", err)
	}

	// * get relative path from module root
	var relativePath *string
	if parser.Module != nil && parser.Module.Path != nil {
		rel, err := filepath.Rel(*parser.Module.Path, path)
		if err != nil {
			log.Printf("warning: failed to get relative path from module root: %v", err)
			relativePath = &path
		} else {
			relativePath = &rel
		}
	} else {
		relativePath = &path
	}

	// * extract directory name
	dirName := filepath.Base(path)

	// * extract package name (last part of full package name)
	packageNameLast := packageName
	if lastSlash := strings.LastIndex(packageName, "/"); lastSlash >= 0 {
		packageNameLast = packageName[lastSlash+1:]
	}

	// * create package struct
	pkg := &Package{
		Path:          relativePath,
		DirectoryName: &dirName,
		Package:       &packageName,
		PackageName:   &packageNameLast,
		Files:         []*File{},
		Module:        parser.Module,
	}

	// * read directory contents and parse Go files
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, s.Error("failed to read directory", err)
	}

	for _, entry := range entries {
		// * skip directories and non-Go files
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		// * skip test files
		if strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		file, err := ParsePackageFile(ctx, pkg, filePath)
		if err != nil {
			log.Printf("warning: failed to parse file %s: %v", filePath, err)
			continue
		}

		if file != nil {
			pkg.Files = append(pkg.Files, file)
		}
	}

	return pkg, nil
}

func ParsePackagePackageName(ctx context.Context, absolutePath string) (string, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()
	s.Variable("absolutePath", absolutePath)

	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return "", s.Error("failed to read directory", err)
	}

	// * find the first go file to extract package name
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		filePath := filepath.Join(absolutePath, entry.Name())

		// * parse the file to extract package name
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

	return "", s.Error("no Go files found in directory", nil)
}
