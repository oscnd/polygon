package telemetry

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.scnd.dev/open/polygon/package/span"
)

func (r *Telemetry) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		// * ensure context
		if c.Context() == nil {
			c.SetContext(context.Background())
		}
		ctx := span.NewContext(r.Polygon, c.Context())

		// * start span
		s, ctx := r.Layer.With(ctx)
		defer s.End()

		// * set context
		c.SetContext(ctx)

		// * set attributes
		s.Trace().SetAttributes(
			attribute.String("http.method", c.Method()),
			attribute.String("http.url", c.OriginalURL()),
			attribute.String("http.user_agent", c.Get("User-Agent")),
		)

		// * count metric
		r.Instrument.HttpActiveRequestCounter(ctx, 1, c.OriginalURL())
		defer r.Instrument.HttpActiveRequestCounter(ctx, -1, c.OriginalURL())

		// * proceed to next
		err := c.Next()

		// * count metric
		r.Instrument.HttpDurationRecord(ctx, time.Now().Sub(*s.Started()).Milliseconds(), c.OriginalURL(), c.Response().StatusCode())
		s.Trace().SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))
		return err
	}
}
