package span

import (
	"context"

	"go.scnd.dev/open/polygon"
)

type ContextKey struct {
	Name string
}

var (
	ContextKeyPolygon = ContextKey{
		Name: "polygon",
	}
	ContextKeySpan = ContextKey{
		Name: "polygon.span",
	}
)

func NewContext(polygon polygon.Polygon, ctx context.Context) context.Context {
	return context.WithValue(ctx, ContextKeyPolygon, polygon)
}

func FromContext(ctx context.Context) polygon.Polygon {
	p, ok := ctx.Value(ContextKeyPolygon).(polygon.Polygon)
	if !ok {
		return nil
	}

	return p
}
