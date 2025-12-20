package span

import "go.scnd.dev/open/polygon"

type Wrapper struct {
	Span *Span `json:"span"`
}

func (r *Wrapper) Context() polygon.SpanContext {
	return r.Span.Context
}

func (r *Wrapper) Variable(key string, value any) {
	r.Span.Variable(key, value)
}

func (r *Wrapper) Fork(layer string) polygon.Span {
	return &Wrapper{
		Span: r.Span.Fork(layer),
	}
}

func (r *Wrapper) Error(message string, err error) error {
	return r.Span.Error(message, err)
}

func (r *Wrapper) End() {
	r.Span.End()
}
