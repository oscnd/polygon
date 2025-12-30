package code

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Parser struct {
	Module *Module
}

func NewParser(modulePath string) (*Parser, error) {
	if modulePath == "" {
		return nil, fmt.Errorf("module path cannot be empty")
	}

	// Validate that the path exists and is a directory
	info, err := os.Stat(modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to access module path %s: %w", modulePath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("module path %s is not a directory", modulePath)
	}

	// Extract module name from go.mod file
	goModPath := filepath.Join(modulePath, "go.mod")
	moduleName := filepath.Base(modulePath) // Fallback

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

func (p *Parser) ParseModule() error {
	if p.Module == nil || p.Module.Path == nil {
		return fmt.Errorf("parser not initialized or module path not set")
	}

	// Walk through the module directory to find Go packages
	err := filepath.Walk(*p.Module.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("warning: error accessing path %s: %v", path, err)
			return nil // Continue walking
		}

		// Skip directories that should not be packages
		if info.IsDir() {
			dirName := filepath.Base(path)
			// Skip hidden directories and common non-package directories
			if strings.HasPrefix(dirName, ".") || dirName == "vendor" || dirName == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files, skip test files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Get the directory containing this Go file
		dirPath := filepath.Dir(path)

		// Check if we've already processed this directory
		for _, pkg := range p.Module.Packages {
			if pkg.Path != nil && *pkg.Path == dirPath {
				return nil // Already processed this package
			}
		}

		// Parse the package
		pkg, err := ParsePackage(p, dirPath)
		if err != nil {
			log.Printf("warning: failed to parse package at %s: %v", dirPath, err)
			return nil // Continue with other packages
		}

		// Add package to module
		if pkg != nil {
			p.Module.Packages = append(p.Module.Packages, pkg)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking module directory: %w", err)
	}

	return nil
}
