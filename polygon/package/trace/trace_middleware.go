package trace

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.scnd.dev/open/polygon/package/span"
)

func (r *Trace) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		name := fmt.Sprintf("HTTP %s %s", c.Method(), c.OriginalURL())
		s := &span.Wrapper{
			Span: span.NewContext(r.Polygon, c.RequestCtx(), name, "trace", make(map[string]any)),
		}
		defer s.End()

		// * start tracer
		context, traceSpan := r.Tracer.Start(s.Context(), name)

		// * set context
		c.SetContext(context)

		// * set attributes
		traceSpan.SetAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.url", c.OriginalURL()),
			attribute.String("http.user_agent", c.Get("User-Agent")),
		)

		// * proceed to next
		err := c.Next()

		traceSpan.SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))
		return err
	}
}
