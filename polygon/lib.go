package polygon

import "context"

type Polygon interface {
	Span(context context.Context, layer string, arguments map[string]any) Span
}
