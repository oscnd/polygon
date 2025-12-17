package canvas

import (
	"context"
	"strings"

	"go.scnd.dev/open/polygon/package/flow"
)

type Import struct {
	Imports []*ImportItem `json:"imports,omitempty"`
}

type ImportItem struct {
	Alias *string `json:"alias,omitempty"`
	Path  *string `json:"path,omitempty"`
}

func (r *Import) AddImport(item *ImportItem) error {
	// * construct error dimension
	dimension := flow.NewContext(context.TODO(), "canvas", make(map[string]any))
	if item.Path == nil {
		return flow.NewError(dimension, "path is nil", nil)
	}
	if item.Alias == nil {
		segments := strings.Split(*item.Path, "/")
		item.Alias = &segments[len(segments)-1]
	}
	for _, existing := range r.Imports {
		if *existing.Alias == *item.Alias {
			dimension.Parameters["alias"] = *item.Alias
			return flow.NewError(dimension, "import alias already exists", nil)
		}
	}
	r.Imports = append(r.Imports, item)
	return nil
}
