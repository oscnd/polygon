package polygon

import (
	"context"
	"time"
)

type Layer interface {
	With(ctx context.Context) (Span, context.Context)
}

type Span interface {
	Started() *time.Time
	Variable(key string, value any)
	Error(message string, err error) error
	End()
}

type Instrument interface {
	HttpDurationRecord(ctx context.Context, duration int64, path string, status int)
	HttpActiveRequestCounter(ctx context.Context, delta int64, path string)
}
