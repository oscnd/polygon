package polygon

import (
	"context"
	"time"
)

type Span interface {
	Context() context.Context
	SetContext(context context.Context)
	Started() *time.Time
	Error(message string, err error) error
	Variable(key string, value any)
	Fork(layer string) Span
	End()
}

type Instrument interface {
	HttpDurationRecord(ctx context.Context, duration int64, path string, status int)
	HttpActiveRequestCounter(ctx context.Context, delta int64, path string)
}
