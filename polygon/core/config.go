package core

import (
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
)

type Config struct {
	OtlpExporter *otlptrace.Exporter
}
