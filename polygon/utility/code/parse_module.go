package code

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon"
)

func (r *Parser) ParseModule(ctx context.Context) error {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	if r.Module == nil || r.Module.Path == nil {
		return s.Error("parser not initialized or module path not set", nil)
	}

	// * walk through the module directory to find Go packages
	err := filepath.Walk(*r.Module.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("warning: error accessing path %s: %v", path, err)
			return nil
		}

		// * skip directories that should not be packages
		if info.IsDir() {
			dirName := filepath.Base(path)
			// * skip hidden directories and common non-package directories
			if strings.HasPrefix(dirName, ".") || dirName == "vendor" || dirName == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// * only process .go files, skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// * get the directory containing this Go file
		dirPath := filepath.Dir(path)

		// * get relative path from module root
		relPath, err := filepath.Rel(*r.Module.Path, dirPath)
		if err != nil {
			log.Printf("warning: failed to get relative path: %v", err)
			return nil
		}

		// * check if we've already processed this directory (O(1) lookup)
		if _, exists := r.Module.Packages[relPath]; exists {
			return nil
		}

		// * parse the package
		pkg, err := r.ParsePackage(ctx, dirPath)
		if err != nil {
			log.Printf("warning: failed to parse package at %s: %v", dirPath, err)
			return nil
		}

		// * add package to module maps
		if pkg != nil && pkg.Path != nil && pkg.PackageName != nil {
			r.Module.Packages[*pkg.Path] = pkg
			r.Module.PackageNames[*pkg.PackageName] = *pkg.Path
		}

		return nil
	})

	if err != nil {
		return s.Error("error walking module directory", err)
	}

	return nil
}
