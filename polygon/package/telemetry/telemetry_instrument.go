package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Instrument struct {
	HttpDurationHistogram          metric.Int64Histogram
	HttpActiveRequestUpDownCounter metric.Int64UpDownCounter
}

func NewInstrument(meter metric.Meter) (*Instrument, error) {
	httpDurationHistogram, err := meter.Int64Histogram(
		"app.http.duration",
		metric.WithDescription("Duration of HTTP requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	httpActiveRequestUpDownCounter, err := meter.Int64UpDownCounter(
		"app.http.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	return &Instrument{
		HttpDurationHistogram:          httpDurationHistogram,
		HttpActiveRequestUpDownCounter: httpActiveRequestUpDownCounter,
	}, nil
}

func (r *Instrument) HttpDurationRecord(ctx context.Context, duration int64, path string, status int) {
	r.HttpDurationHistogram.Record(
		ctx,
		duration,
		metric.WithAttributes(
			attribute.String("http.path", path),
			attribute.Int("http.status", status),
		),
	)
}

func (r *Instrument) HttpActiveRequestCounter(ctx context.Context, delta int64, path string) {
	r.HttpActiveRequestUpDownCounter.Add(
		ctx,
		delta,
		metric.WithAttributes(
			attribute.String("http.path", path),
		),
	)
}
