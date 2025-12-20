package polygon

import (
	"context"
)

type Span interface {
	Context() SpanContext
	Error(message string, err error) error
	Variable(key string, value any)
	Fork(layer string) Span
	End()
}

type SpanContext interface {
	context.Context
}
