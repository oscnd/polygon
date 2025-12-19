package span

import (
	"fmt"
	"time"
)

type Span struct {
	Name      *string        `json:"name,omitempty"`
	Layer     *string        `json:"layer,omitempty"`
	Context   *Context       `json:"context,omitempty"`
	Trace     *Caller        `json:"trace,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
	Started   *time.Time     `json:"started,omitempty"`
	Ended     *time.Time     `json:"ended,omitempty"`
}

func (r *Span) Variable(key string, value any) {
	r.Variables[key] = value
}

func (r *Span) Fork(layer string) *Span {
	trace := NewCaller(2)
	traceStr := trace.String()
	name := fmt.Sprintf("%s/%s", *r.Name, traceStr)
	now := time.Now()
	d2 := &Span{
		Name:      &name,
		Layer:     &layer,
		Context:   r.Context,
		Trace:     trace,
		Variables: nil,
		Started:   &now,
		Ended:     nil,
	}

	r.Context.Spans = append(r.Context.Spans, d2)
	return d2
}

func (r *Span) End() {
	end := time.Now()
	r.Ended = &end
}
