package snapshot

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("snapshot-cli/internal/snapshot")

// spanHelper wraps a trace.Span for convenient tracing.
type spanHelper struct {
	span trace.Span
}

// End marks the span as ended with success status.
func (s *spanHelper) End() {
	s.span.SetStatus(codes.Ok, "")
	s.span.End()
}

// RecordError records an error on the span and sets error status.
func (s *spanHelper) RecordError(err error) {
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

// startCreateSpan begins a new span for snapshot creation operations.
func startCreateSpan(ctx context.Context, volumeID, shareID, name string) (context.Context, *spanHelper) {
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
	return ctx, &spanHelper{span}
}

// startDeleteSpan begins a new span for snapshot deletion operations.
func startDeleteSpan(ctx context.Context, snapshotID string) (context.Context, *spanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.delete")
	span.SetAttributes(attribute.String("snapshot.id", snapshotID))
	return ctx, &spanHelper{span}
}

// startListSpan begins a new span for snapshot list operations.
func startListSpan(ctx context.Context, volumeID, shareID string) (context.Context, *spanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.list")
	if volumeID != "" {
		span.SetAttributes(attribute.String("snapshot.volume_id", volumeID))
	}
	if shareID != "" {
		span.SetAttributes(attribute.String("snapshot.share_id", shareID))
	}
	return ctx, &spanHelper{span}
}

// startGetSpan begins a new span for snapshot get operations.
func startGetSpan(ctx context.Context, snapshotID string) (context.Context, *spanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.get")
	span.SetAttributes(attribute.String("snapshot.id", snapshotID))
	return ctx, &spanHelper{span}
}

// startCleanupSpan begins a new span for snapshot cleanup operations.
func startCleanupSpan(ctx context.Context, volumeID, shareID string, olderThanSeconds int64) (context.Context, *spanHelper) {
	ctx, span := tracer.Start(ctx, "snapshot.cleanup")
	if volumeID != "" {
		span.SetAttributes(attribute.String("snapshot.volume_id", volumeID))
	}
	if shareID != "" {
		span.SetAttributes(attribute.String("snapshot.share_id", shareID))
	}
	span.SetAttributes(attribute.Int64("snapshot.older_than_seconds", olderThanSeconds))
	return ctx, &spanHelper{span}
}
