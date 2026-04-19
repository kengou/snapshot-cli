package snapshot

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("snapshot-cli/internal/snapshot")

// SpanHelper wraps a trace.Span for convenient tracing.
type SpanHelper struct {
	span trace.Span
}

// End marks the span as ended with success status.
func (s *SpanHelper) End() {
	s.span.SetStatus(codes.Ok, "")
	s.span.End()
}

// RecordError records an error on the span and sets error status.
func (s *SpanHelper) RecordError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// startCreateSpan begins a new span for snapshot creation operations.
func startCreateSpan(ctx context.Context, volumeID, shareID, name string) (context.Context, *SpanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.create")
	if volumeID != "" {
		span.SetAttributes(attribute.String("snapshot.volume_id", volumeID))
	}
	if shareID != "" {
		span.SetAttributes(attribute.String("snapshot.share_id", shareID))
	}
	if name != "" {
		span.SetAttributes(attribute.String("snapshot.name", name))
	}
	return ctx, &SpanHelper{span}
}

// startDeleteSpan begins a new span for snapshot deletion operations.
//
//nolint:unused
func startDeleteSpan(ctx context.Context, snapshotID string) (context.Context, *SpanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.delete")
	span.SetAttributes(attribute.String("snapshot.id", snapshotID))
	return ctx, &SpanHelper{span}
}

// startListSpan begins a new span for snapshot list operations.
//
//nolint:unused
func startListSpan(ctx context.Context, volumeID, shareID string) (context.Context, *SpanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.list")
	if volumeID != "" {
		span.SetAttributes(attribute.String("snapshot.volume_id", volumeID))
	}
	if shareID != "" {
		span.SetAttributes(attribute.String("snapshot.share_id", shareID))
	}
	return ctx, &SpanHelper{span}
}

// startGetSpan begins a new span for snapshot get operations.
//
//nolint:unused
func startGetSpan(ctx context.Context, snapshotID string) (context.Context, *SpanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.get")
	span.SetAttributes(attribute.String("snapshot.id", snapshotID))
	return ctx, &SpanHelper{span}
}

// startCleanupSpan begins a new span for snapshot cleanup operations.
//
//nolint:unused
func startCleanupSpan(ctx context.Context, volumeID, shareID string, olderThanSeconds int64) (context.Context, *SpanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.cleanup")
	if volumeID != "" {
		span.SetAttributes(attribute.String("snapshot.volume_id", volumeID))
	}
	if shareID != "" {
		span.SetAttributes(attribute.String("snapshot.share_id", shareID))
	}
	span.SetAttributes(attribute.Int64("snapshot.older_than_seconds", olderThanSeconds))
	return ctx, &SpanHelper{span}
}
