package observability

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracerProvider initializes and returns an OpenTelemetry TracerProvider.
// It configures OTLP gRPC exporter if OTEL_EXPORTER_OTLP_ENDPOINT is set.
// If the endpoint is not set or connection fails, a no-op tracer provider is used.
func InitTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	// Check if OTEL is enabled
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// OTEL disabled; return no-op provider
		return trace.NewTracerProvider(), nil
	}

	// Create OTLP exporter
	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("snapshot-cli"),
			semconv.ServiceVersion(getVersion()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	// Set as global tracer provider
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

// getVersion returns the application version or "dev" if not set.
func getVersion() string {
	version := os.Getenv("SNAPSHOT_CLI_VERSION")
	if version == "" {
		version = "dev"
	}
	return version
}
