package polygon

import (
	"context"
)

type Span interface {
	Context() context.Context
	SetContext(context context.Context)
	Error(message string, err error) error
	Variable(key string, value any)
	Fork(layer string) Span
	End()
}
