package code

import (
	"context"
	"strings"

	"go.scnd.dev/open/polygon"
)

type Module struct {
	Path     *string    `json:"path"`     // absolute path
	Name     *string    `json:"name"`     // module name
	Packages []*Package `json:"packages"` // circular pointer to packages
}

type File struct {
	Package    *Package     `json:"package"`
	Name       *string      `json:"name"`
	Import     *Import      `json:"import"`
	Interfaces []*Interface `json:"interfaces"`
	Structs    []*Struct    `json:"structs"`
	Receivers  []*Receiver  `json:"receivers"`
	Functions  []*Method    `json:"functions"`
}

type Interface struct {
	Name    *string   `json:"name"`
	Methods []*Method `json:"methods"`
}

type Method struct {
	Name       *string      `json:"name"`
	Parameters []*Parameter `json:"parameters"`
	Results    []*Parameter `json:"results"`
}

type Parameter struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

type Struct struct {
	Name      *string     `json:"name"`
	Fields    []*Field    `json:"fields"`
	Receivers []*Receiver `json:"receivers"`
}

type Field struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
	Tags []*Tag  `json:"tags"`
}

type Tag struct {
	Name  *string
	Value *string
}

type Receiver struct {
	Name   *string `json:"name"`
	Struct *Struct `json:"struct"`
	Method *Method `json:"method"`
}

type Import struct {
	Imports []*ImportItem `json:"imports,omitempty"`
}

type ImportItem struct {
	Alias *string `json:"alias,omitempty"`
	Path  *string `json:"path,omitempty"`
}

type Canvas struct {
	Import *Import `json:"import,omitempty"`
}

func New() *Canvas {
	return &Canvas{
		Import: new(Import),
	}
}

func (r *Import) AddImport(ctx context.Context, item *ImportItem) error {
	// * start span
	s, ctx := polygon.With(ctx)
	defer s.End()

	if item.Path == nil {
		return s.Error("path is nil", nil)
	}
	if item.Alias == nil {
		segments := strings.Split(*item.Path, "/")
		item.Alias = &segments[len(segments)-1]
	}
	for _, existing := range r.Imports {
		if *existing.Alias == *item.Alias {
			return s.Error("import alias already exists", nil)
		}
	}
	r.Imports = append(r.Imports, item)
	return nil
}
