package core

import (
	"context"

	"github.com/gofiber/fiber/v3"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/package/span"
	"go.scnd.dev/open/polygon/package/trace"
)

type Instance struct {
	config *polygon.Config
	trace  *trace.Trace
}

func New(config *polygon.Config) (_ polygon.Polygon, err error) {
	i := &Instance{
		config: config,
		trace:  nil,
	}

	i.trace, err = trace.New(i)
	if err != nil {
		return nil, err
	}

	return i, nil
}

func (r *Instance) Config() *polygon.Config {
	return r.config
}

func (r *Instance) Span(context context.Context, name string, layer string, arguments map[string]any) polygon.Span {
	return &span.Wrapper{
		Span: span.NewContext(r, context, name, layer, arguments),
	}
}

func (r *Instance) Tracer() oteltrace.Tracer {
	return r.trace.Tracer
}

func (r *Instance) TracerMiddleware() fiber.Handler {
	return r.trace.Middleware()
}
