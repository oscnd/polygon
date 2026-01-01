package telemetry

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func (r *Telemetry) GetMeter() metric.Meter {
	return r.Meter
}

func (r *Telemetry) GetTracer() trace.Tracer {
	return r.Tracer
}
