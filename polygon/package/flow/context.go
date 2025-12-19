package flow

import (
	"context"
	"time"
)

type Context struct {
	Context    context.Context `json:"context,omitempty"`
	Scope      *string         `json:"scope,omitempty"`
	Type       *string         `json:"type,omitempty"`
	Arguments  map[string]any  `json:"arguments,omitempty"`
	Parameters map[string]any  `json:"parameters,omitempty"`
	Dimensions []*Dimension    `json:"-"`
}

func NewContext(context context.Context, scope string, arguments map[string]any) (*Dimension, func()) {
	c := &Context{
		Context:    context,
		Scope:      &scope,
		Type:       nil,
		Arguments:  arguments,
		Dimensions: make([]*Dimension, 0),
	}

	trace := NewTrace(2)
	traceStr := trace.String()
	now := time.Now()
	d := &Dimension{
		Name:      &traceStr,
		Context:   c,
		Trace:     trace,
		Variables: nil,
		Started:   &now,
		Ended:     nil,
	}

	e := func() {
		// TODO: implement cleanup
	}

	c.Dimensions = append(c.Dimensions, d)

	return d, e
}

func (r *Context) Parameter(key string, value any) {
	r.Parameters[key] = value
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
