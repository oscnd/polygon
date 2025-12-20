package core

import (
	"context"

	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/package/span"
)

type Instance struct {
	Internal *polygon.Internal
}

func New(config *polygon.Config) polygon.Polygon {
	return &Instance{
		Internal: &polygon.Internal{
			Config: config,
		},
	}
}

func (r *Instance) Span(context context.Context, layer string, arguments map[string]any) polygon.Span {
	return &span.Wrapper{
		Span: span.NewContext(r.Internal, context, layer, arguments),
	}
}
