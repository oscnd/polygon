package code

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"go.scnd.dev/open/polygon"
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
			Path:         &modulePath,
			Name:         &moduleName,
			Packages:     make(map[string]*Package),
			PackageNames: make(map[string]string),
		},
	}

	return parser, nil
}

func (r *Parser) PackageByPath(ctx context.Context, path string) (*Package, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()
	s.Variable("path", path)

	if r.Module == nil {
		return nil, s.Error("parser not initialized", nil)
	}

	if path == "" {
		return nil, s.Error("path cannot be empty", nil)
	}

	// * check if already parsed
	if pkg, exists := r.Module.Packages[path]; exists {
		return pkg, nil
	}

	// * parse the package
	pkg, err := r.ParsePackage(ctx, path)
	if err != nil {
		return nil, s.Error("failed to parse package", err)
	}

	// * add to module maps
	if pkg != nil && pkg.Path != nil && pkg.PackageName != nil {
		r.Module.Packages[*pkg.Path] = pkg
		r.Module.PackageNames[*pkg.PackageName] = *pkg.Path
	}

	return pkg, nil
}

func (r *Parser) PackageByName(ctx context.Context, name string) (*Package, error) {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()
	s.Variable("name", name)

	if r.Module == nil {
		return nil, s.Error("parser not initialized", nil)
	}

	if name == "" {
		return nil, s.Error("name cannot be empty", nil)
	}

	// * lookup in PackageNames map
	relPath, exists := r.Module.PackageNames[name]
	if !exists {
		return nil, s.Error("package not found", nil)
	}

	// * use PackageByPath to get or parse
	return r.PackageByPath(ctx, relPath)
}
