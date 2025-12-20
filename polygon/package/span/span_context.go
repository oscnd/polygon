package span

import (
	"context"
	"time"

	"go.scnd.dev/open/polygon/core"
)

type Context struct {
	Internal  *core.Internal
	Context   context.Context
	Type      *string
	Arguments map[string]any
	Spans     []*Span
}

func NewContext(internal *core.Internal, context context.Context, layer string, arguments map[string]any) *Span {
	c := &Context{
		Internal:  internal,
		Context:   context,
		Type:      nil,
		Arguments: arguments,
		Spans:     make([]*Span, 0),
	}

	trace := NewCaller(2)
	traceStr := trace.String()
	now := time.Now()
	s := &Span{
		Name:      &traceStr,
		Layer:     &layer,
		Context:   c,
		Trace:     trace,
		Variables: nil,
		Started:   &now,
		Ended:     nil,
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
