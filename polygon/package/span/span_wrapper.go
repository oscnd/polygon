package span

import (
	"time"
)

type Wrapper struct {
	Span *Span `json:"span"`
}

func (r *Wrapper) Started() *time.Time {
	return r.Span.Started
}

func (r *Wrapper) Variable(key string, value any) {
	r.Span.Variable(key, value)
}

func (r *Wrapper) Error(message string, err error) error {
	return r.Span.Error(message, err)
}

func (r *Wrapper) End() {
	r.Span.End()
}
