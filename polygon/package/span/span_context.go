package span

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.scnd.dev/open/polygon"
)

type Context struct {
	Polygon   polygon.Polygon
	Context   context.Context
	Type      *string
	Arguments map[string]any
	Spans     []*Span
}

func NewContext(polygon polygon.Polygon, context context.Context, name string, layer string, arguments map[string]any) *Span {
	c := &Context{
		Polygon:   polygon,
		Context:   context,
		Type:      nil,
		Arguments: arguments,
		Spans:     make([]*Span, 0),
	}

	caller := NewCaller(2)
	now := time.Now()

	tracingContext, tracingSpan := polygon.Tracer().Start(context, name)
	tracingSpan.SetAttributes(attribute.String("span.layer", layer))
	tracingSpan.SetAttributes(attribute.String("span.caller", caller.String()))

	s := &Span{
		Name:           &name,
		Layer:          &layer,
		Context:        c,
		Caller:         caller,
		Variables:      nil,
		Started:        &now,
		Ended:          nil,
		TracingSpan:    tracingSpan,
		TracingContext: tracingContext,
	}
	c.Spans = append(c.Spans, s)

	return s
}

func (r *Context) Deadline() (deadline time.Time, ok bool) {
	return r.Context.Deadline()
}

func (r *Context) Done() <-chan struct{} {
	return r.Context.Done()
}

func (r *Context) Err() error {
	return r.Context.Err()
}

func (r *Context) Value(key any) any {
	return r.Context.Value(key)
}
