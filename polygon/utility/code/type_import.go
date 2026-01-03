package code

import (
	"context"
	"strings"

	"go.scnd.dev/open/polygon"
)

type Import struct {
	Imports []*ImportItem `json:"imports,omitempty"`
}

type ImportItem struct {
	Alias *string `json:"alias,omitempty"`
	Path  *string `json:"path,omitempty"`
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
