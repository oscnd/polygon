package polygon

import "go.scnd.dev/open/polygon/package/span"

type Span interface {
	Context() *span.Context
	Variable(key string, value any)
	Fork(layer string) Span
	End()
}

type SpanWrapper struct {
	Span *span.Span `json:"span"`
}

func (r *SpanWrapper) Context() *span.Context {
	return r.Span.Context
}

func (r *SpanWrapper) Variable(key string, value any) {
	r.Span.Variable(key, value)
}

func (r *SpanWrapper) Fork(layer string) Span {
	return &SpanWrapper{
		Span: r.Span.Fork(layer),
	}
}

func (r *SpanWrapper) End() {
	r.Span.End()
}
