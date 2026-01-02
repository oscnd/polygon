package span

import (
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Span struct {
	Name      *string        `json:"name,omitempty"`
	Path      []*string      `json:"path,omitempty"`
	Layer     *Layer         `json:"layer,omitempty"`
	Caller    *Caller        `json:"caller,omitempty"`
	Variables map[string]any `json:"variables,omitempty"`
	Started   *time.Time     `json:"started,omitempty"`
	Ended     *time.Time     `json:"ended,omitempty"`
	Children  []*Span        `json:"children,omitempty"`
	TraceSpan trace.Span     `json:"-"`
}

func (r *Span) Variable(key string, value any) {
	r.Variables[key] = value
	// TODO: add json marshalling if value is struct or map, or keep if string
	r.TraceSpan.SetAttributes(attribute.String(fmt.Sprintf("var.%s", key), fmt.Sprintf("%v", value)))
}

func (r *Span) Error(message string, err error) error {
	return NewError(r, message, err)
}

func (r *Span) Tracing() trace.Span {
	return r.TraceSpan
}

func (r *Span) End() {
	end := time.Now()
	r.Ended = &end
	r.TraceSpan.End()
}
