package polygon

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
)

type Polygon interface {
	Config() *Config
	Tracer() trace.Tracer
	TracerMiddleware() fiber.Handler
	Span(context context.Context, name, layer string, arguments map[string]any) Span
}
