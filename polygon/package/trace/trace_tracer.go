package trace

import "go.opentelemetry.io/otel/trace"

func (r *Trace) GetTracer() trace.Tracer {
	return r.Tracer
}
