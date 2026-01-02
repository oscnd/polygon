package code

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/package/span"
)

type Parser struct {
	Module *Module
}

func NewParser(ctx context.Context, modulePath string) (*Parser, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()
	s.Variable("modulePath", modulePath)

	if modulePath == "" {
		return nil, s.Error("module path cannot be empty", nil)
	}

	// * validate that the path exists and is a directory
	info, err := os.Stat(modulePath)
	if err != nil {
		return nil, s.Error("failed to access module path", err)
	}

	if !info.IsDir() {
		return nil, s.Error("module path is not a directory", nil)
	}

	// * extract module name from go.mod file
	goModPath := filepath.Join(modulePath, "go.mod")
	moduleName := filepath.Base(modulePath)

	if file, err := os.Open(goModPath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "module ") {
				moduleName = strings.TrimPrefix(line, "module ")
				moduleName = strings.TrimSpace(moduleName)
				moduleName = strings.Trim(moduleName, `"`)
				break
			}
		}
	} else {
		log.Printf("warning: failed to open go.mod file at %s: %v", goModPath, err)
	}

	parser := &Parser{
		Module: &Module{
			Path:     &modulePath,
			Name:     &moduleName,
			Packages: []*Package{},
		},
	}

	return parser, nil
}

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

		// * check if we've already processed this directory
		for _, pkg := range r.Module.Packages {
			if pkg.Path != nil && *pkg.Path == dirPath {
				return nil
			}
		}

		// * parse the package
		pkg, err := ParsePackage(ctx, r, dirPath)
		if err != nil {
			log.Printf("warning: failed to parse package at %s: %v", dirPath, err)
			return nil
		}

		// * add package to module
		if pkg != nil {
			r.Module.Packages = append(r.Module.Packages, pkg)
		}

		return nil
	})

	if err != nil {
		return s.Error("error walking module directory", err)
	}

	return nil
}
