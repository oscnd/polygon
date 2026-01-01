package telemetry

import (
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
)

type Wrapper struct {
	Trace *Telemetry
}

func (r *Wrapper) Tracer() trace.Tracer {
	return r.Trace.Tracer
}

func (r *Wrapper) Middleware() fiber.Handler {
	return r.Trace.Middleware()
}
