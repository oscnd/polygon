package span

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Span struct {
	Name           *string         `json:"name,omitempty"`
	Layer          *string         `json:"layer,omitempty"`
	Context        *Context        `json:"context,omitempty"`
	Caller         *Caller         `json:"caller,omitempty"`
	Variables      map[string]any  `json:"variables,omitempty"`
	Started        *time.Time      `json:"started,omitempty"`
	Ended          *time.Time      `json:"ended,omitempty"`
	TracingSpan    trace.Span      `json:"-"`
	TracingContext context.Context `json:"-"`
}

func (r *Span) Variable(key string, value any) {
	r.Variables[key] = value
	r.TracingSpan.SetAttributes(attribute.String(fmt.Sprintf("var.%s", key), fmt.Sprintf("%v", value)))
}

func (r *Span) Fork(layer string) *Span {
	caller := NewCaller(2)
	now := time.Now()
	traceStr := caller.String()
	name := fmt.Sprintf("%s/%s", *r.Name, traceStr)

	tracingContext, tracingSpan := r.Context.Polygon.Tracer().Start(r.Context, name)
	tracingSpan.SetAttributes(attribute.String("span.layer", layer))
	tracingSpan.SetAttributes(attribute.String("span.caller", caller.String()))

	d2 := &Span{
		Name:           &name,
		Layer:          &layer,
		Context:        r.Context,
		Caller:         caller,
		Variables:      nil,
		Started:        &now,
		Ended:          nil,
		TracingSpan:    tracingSpan,
		TracingContext: tracingContext,
	}

	r.Context.Spans = append(r.Context.Spans, d2)
	return d2
}

func (r *Span) Error(message string, err error) error {
	return NewError(r, message, err)
}

func (r *Span) Tracing() trace.Span {
	return r.TracingSpan
}

func (r *Span) End() {
	end := time.Now()
	r.Ended = &end
	r.TracingSpan.End()
}
