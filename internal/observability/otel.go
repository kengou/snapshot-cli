package observability

import (
	"context"
	"fmt"
	"os"

	"github.com/sapcc/go-api-declarations/bininfo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracerProvider initializes and returns an OpenTelemetry TracerProvider.
// Tracing is enabled only when OTEL_EXPORTER_OTLP_ENDPOINT (or the traces-specific
// OTEL_EXPORTER_OTLP_TRACES_ENDPOINT) is set; otherwise a no-op provider is returned.
// The exporter itself reads the standard OTEL_EXPORTER_OTLP_* environment variables
// (endpoint, TLS/insecure via the URL scheme, headers, timeouts), so an endpoint
// like http://localhost:4317 exports without TLS and https://... with TLS.
func InitTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		// OTEL disabled; return no-op provider
		return trace.NewTracerProvider(), nil
	}

	exporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("snapshot-cli"),
			semconv.ServiceVersion(bininfo.VersionOr("dev")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

// Shutdown flushes any pending spans and closes the exporter.
func Shutdown(ctx context.Context, tp *trace.TracerProvider) error {
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}
