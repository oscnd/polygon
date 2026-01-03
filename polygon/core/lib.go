package core

import (
	"github.com/gofiber/fiber/v3"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/package/span"
	"go.scnd.dev/open/polygon/package/telemetry"
)

type Instance struct {
	config    *polygon.Config
	telemetry *telemetry.Telemetry
}

func New(config *polygon.Config) (_ polygon.Polygon, err error) {
	i := &Instance{
		config:    config,
		telemetry: nil,
	}

	i.telemetry, err = telemetry.New(i)
	if err != nil {
		return nil, err
	}

	polygon.With = span.NewLayer(i, "", "").With

	return i, nil
}

func (r *Instance) Config() *polygon.Config {
	return r.config
}

func (r *Instance) Layer(name string, typ string) polygon.Layer {
	return span.NewLayer(r, name, typ)
}

func (r *Instance) Tracer() oteltrace.Tracer {
	return r.telemetry.Tracer
}

func (r *Instance) TracerMiddleware() fiber.Handler {
	return r.telemetry.Middleware()
}

func (r *Instance) Instrument() polygon.Instrument {
	return r.telemetry.Instrument
}

func init() {
	polygon.With = span.NewLayer(new(Instance), "", "").With
}
