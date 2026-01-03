package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
	"go.opentelemetry.io/otel/trace"
	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/package/span"
)

type Telemetry struct {
	Polygon    polygon.Polygon
	Layer      polygon.Layer
	Meter      metric.Meter
	Tracer     trace.Tracer
	Instrument *Instrument
}

func New(polygon polygon.Polygon) (_ *Telemetry, err error) {
	// * construct telemetry
	telemetry := &Telemetry{
		Polygon:    polygon,
		Layer:      polygon.Layer("telemetry", "polygon"),
		Meter:      nil,
		Tracer:     nil,
		Instrument: nil,
	}

	// * construct resource
	attributes := make([]attribute.KeyValue, 0)
	if telemetry.Polygon.Config().AppName != nil {
		attributes = append(attributes, semconv.ServiceName(*telemetry.Polygon.Config().AppName))
	}
	if telemetry.Polygon.Config().AppVersion != nil {
		attributes = append(attributes, semconv.ServiceVersion(*telemetry.Polygon.Config().AppVersion))
	}
	if telemetry.Polygon.Config().AppNamespace != nil {
		attributes = append(attributes, semconv.ServiceNamespace(*telemetry.Polygon.Config().AppNamespace))
	}
	if telemetry.Polygon.Config().AppInstanceId != nil {
		attributes = append(attributes, semconv.ServiceInstanceID(*telemetry.Polygon.Config().AppInstanceId))
	}
	res, err := resource.New(context.Background(), resource.WithAttributes(attributes...))
	if err != nil {
		return nil, span.NewError(nil, "unable to initialize resource", err)
	}

	// * construct meter
	telemetry.Meter, err = NewMeter(telemetry, res)
	if err != nil {
		return nil, err
	}

	// * construct tracer
	telemetry.Tracer, err = NewTracer(telemetry, res)
	if err != nil {
		return nil, err
	}

	// * construct instrument
	telemetry.Instrument, err = NewInstrument(telemetry.Meter)

	return telemetry, nil
}

func NewMeter(telemetry *Telemetry, res *resource.Resource) (metric.Meter, error) {
	// * construct exporter
	exporter, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint(*telemetry.Polygon.Config().TelemetryUrl),
		otlpmetricgrpc.WithHeaders(map[string]string{
			"X-Scope-OrgID": *telemetry.Polygon.Config().TelemetryOrganization,
		}),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, span.NewError(nil, "unable to initialize metric exporter", err)
	}

	// * construct provider
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exporter,
			sdkmetric.WithInterval(time.Minute),
		)),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(provider)

	mm := otel.Meter("polygon-meter")

	return mm, nil
}

func NewTracer(telemetry *Telemetry, res *resource.Resource) (trace.Tracer, error) {
	// * construct exporter
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(*telemetry.Polygon.Config().TelemetryUrl),
		otlptracegrpc.WithHeaders(map[string]string{
			"X-Scope-OrgID": *telemetry.Polygon.Config().TelemetryOrganization,
		}),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, span.NewError(nil, "unable to intialize exporter", err)
	}

	// * construct provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tt := otel.Tracer("polygon-tracer")

	return tt, nil
}
