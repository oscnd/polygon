package span

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.scnd.dev/open/polygon"
)

type Layer struct {
	Polygon polygon.Polygon `json:"polygon,omitempty"`
	Name    string          `json:"name,omitempty"`
	Type    string          `json:"type,omitempty"`
	Caller  *Caller         `json:"caller,omitempty"`
}

func NewLayer(polygon polygon.Polygon, name string, typ string) *Layer {
	caller := NewCaller(2)

	return &Layer{
		Polygon: polygon,
		Name:    name,
		Type:    typ,
		Caller:  caller,
	}
}

func (r *Layer) With(ctx context.Context) (polygon.Span, context.Context) {
	span, ok := ctx.Value(ContextKeySpan).(*Span)
	caller := NewCaller(2)
	name := caller.String()
	now := time.Now()

	ctx, tracingSpan := r.Polygon.Tracer().Start(ctx, caller.String())
	tracingSpan.SetAttributes(attribute.String("span.layer", fmt.Sprintf("%s/%s", r.Type, r.Name)))

	if !ok {
		s := &Span{
			Name:      &name,
			Path:      []*string{},
			Layer:     r,
			Caller:    caller,
			Variables: make(map[string]any),
			Started:   &now,
			Ended:     nil,
			Children:  []*Span{},
			TraceSpan: tracingSpan,
		}

		return &Wrapper{Span: s}, context.WithValue(ctx, ContextKeySpan, s)
	}

	s := &Span{
		Name:      &name,
		Path:      append(span.Path, span.Name),
		Layer:     r,
		Caller:    caller,
		Variables: make(map[string]any),
		Started:   &now,
		Ended:     nil,
		Children:  []*Span{},
		TraceSpan: tracingSpan,
	}
	span.Children = append(span.Children, s)

	return &Wrapper{Span: s}, context.WithValue(ctx, ContextKeySpan, s)
}
