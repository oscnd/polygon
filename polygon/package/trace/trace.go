package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.scnd.dev/open/polygon"
	"go.scnd.dev/open/polygon/package/span"
)

type Trace struct {
	Polygon polygon.Polygon
	Tracer  trace.Tracer
}

func New(polygon polygon.Polygon) (*Trace, error) {
	// * construct exporter
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(*polygon.Config().TraceUrl),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, span.NewError(nil, "unable to intialize exporter", err)
	}

	// * construct resource
	attributes := make([]attribute.KeyValue, 0)
	if polygon.Config().AppName != nil {
		attributes = append(attributes, semconv.ServiceName(*polygon.Config().AppName))
	}
	if polygon.Config().AppVersion != nil {
		attributes = append(attributes, semconv.ServiceVersion(*polygon.Config().AppVersion))
	}
	if polygon.Config().AppNamespace != nil {
		attributes = append(attributes, semconv.ServiceNamespace(*polygon.Config().AppNamespace))
	}
	if polygon.Config().AppInstanceId != nil {
		attributes = append(attributes, semconv.ServiceInstanceID(*polygon.Config().AppInstanceId))
	}
	res, err := resource.New(context.Background(), resource.WithAttributes(attributes...))
	if err != nil {
		return nil, span.NewError(nil, "unable to initialize resource", err)
	}

	// * construct provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tt := otel.Tracer("polygon-tracer")

	return &Trace{
		Polygon: polygon,
		Tracer:  tt,
	}, nil
}
