package polygon

import (
	"context"

	"go.scnd.dev/open/polygon/core"
	"go.scnd.dev/open/polygon/package/span"
)

type Polygon interface {
	Span(context context.Context, layer string, arguments map[string]any) Span
}

type Instance struct {
	Internal *core.Internal
}

func (r *Instance) Span(context context.Context, layer string, arguments map[string]any) Span {
	return &SpanWrapper{
		Span: span.NewContext(r.Internal, context, layer, arguments),
	}
}

func New(config *core.Config) Polygon {
	return &Instance{
		Internal: &core.Internal{
			Config: config,
		},
	}
}
