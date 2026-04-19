package observability

import (
	"context"
	"testing"
)

func TestInitTracerProvider_DisabledByDefault(t *testing.T) {
	// Ensure OTEL_EXPORTER_OTLP_ENDPOINT is not set
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	tp, err := InitTracerProvider(context.Background())
	if err != nil {
		t.Fatalf("InitTracerProvider() error = %v, want nil", err)
	}
	if tp == nil {
		t.Error("InitTracerProvider() returned nil tracer provider")
	}
	if tp != nil {
		if shutdownErr := tp.Shutdown(context.Background()); shutdownErr != nil {
			t.Logf("Shutdown error (ignored): %v", shutdownErr)
		}
	}
}

func TestInitTracerProvider_WithValidEndpoint(t *testing.T) {
	// Note: This test uses a loopback endpoint which will fail at export time, not init time.
	// The OTLP exporter is lazy-loaded and doesn't validate connectivity during initialization.
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	tp, err := InitTracerProvider(context.Background())
	if err != nil {
		t.Fatalf("InitTracerProvider() error = %v, want nil", err)
	}
	if tp == nil {
		t.Error("InitTracerProvider() returned nil tracer provider")
	}
	if tp != nil {
		if shutdownErr := tp.Shutdown(context.Background()); shutdownErr != nil {
			t.Logf("Shutdown error (ignored): %v", shutdownErr)
		}
	}
}

func TestShutdown_NilProvider(t *testing.T) {
	err := Shutdown(context.Background(), nil)
	if err != nil {
		t.Errorf("Shutdown(nil) error = %v, want nil", err)
	}
}

func TestShutdown_ValidProvider(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	tp, err := InitTracerProvider(context.Background())
	if err != nil {
		t.Fatalf("InitTracerProvider() error = %v", err)
	}

	err = Shutdown(context.Background(), tp)
	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

func TestGetVersion_Default(t *testing.T) {
	t.Setenv("SNAPSHOT_CLI_VERSION", "")
	version := getVersion()
	if version != "dev" {
		t.Errorf("getVersion() = %q, want %q", version, "dev")
	}
}

func TestGetVersion_Custom(t *testing.T) {
	t.Setenv("SNAPSHOT_CLI_VERSION", "v1.2.3")
	version := getVersion()
	if version != "v1.2.3" {
		t.Errorf("getVersion() = %q, want %q", version, "v1.2.3")
	}
}
