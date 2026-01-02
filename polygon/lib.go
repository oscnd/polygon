package polygon

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/trace"
)

type Polygon interface {
	Config() *Config
	Layer(name string, typ string) Layer
	Tracer() trace.Tracer
	TracerMiddleware() fiber.Handler
	Instrument() Instrument
}

var With func(ctx context.Context) (Span, context.Context)
