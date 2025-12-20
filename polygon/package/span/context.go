package span

import (
	"context"
	"time"
)

type Context struct {
	Context   context.Context `json:"context,omitempty"`
	Type      *string         `json:"type,omitempty"`
	Arguments map[string]any  `json:"arguments,omitempty"`
	Spans     []*Span         `json:"-"`
}

func NewContext(context context.Context, layer string, arguments map[string]any) *Span {
	c := &Context{
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
