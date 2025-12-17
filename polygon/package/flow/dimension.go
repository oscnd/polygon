package flow

import (
	"fmt"
	"time"
)

type Dimension struct {
	Name      *string    `json:"name,omitempty"`
	Context   *Context   `json:"context,omitempty"`
	Trace     *Trace     `json:"trace,omitempty"`
	Variables any        `json:"variables,omitempty"`
	Started   *time.Time `json:"started,omitempty"`
	Ended     *time.Time `json:"ended,omitempty"`
}

type DimensionGeneric[T any] struct {
	*Dimension
}

func (r *DimensionGeneric[T]) GetVariables() T {
	return r.Variables.(T)
}

func (r *Dimension) Fork() (*Dimension, func()) {
	trace := NewTrace(2)
	traceStr := trace.String()
	name := fmt.Sprintf("%s/%s", *r.Name, traceStr)
	now := time.Now()
	d2 := &Dimension{
		Name:      &name,
		Context:   r.Context,
		Trace:     trace,
		Variables: nil,
		Started:   &now,
		Ended:     nil,
	}

	f := func() {
		end := time.Now()
		d2.Ended = &end
	}

	r.Context.Dimensions = append(r.Context.Dimensions, d2)
	return d2, f
}
