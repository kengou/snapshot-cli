# OpenTelemetry Integration

snapshot-cli includes built-in observability through [OpenTelemetry](https://opentelemetry.io/) distributed tracing.

## Overview

Observability is **disabled by default** and requires an OpenTelemetry collector to export traces. This guide covers setup and usage.

## Enabling Traces

Set the `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable to point to an OTEL collector:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
snapshot-cli snapshot list --volume
```

Traces will be exported to the collector at `localhost:4317` (gRPC protocol).

## Configuration

| Environment Variable | Default | Description |
|----------------------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `` (disabled) | gRPC endpoint for OTEL Collector |
| `OTEL_SDK_DISABLED` | `false` | Disable OTEL SDK entirely |
| `SNAPSHOT_CLI_VERSION` | `dev` | Version string attached to traces |

## Quick Start: Local Jaeger

Run Jaeger locally to visualize traces:

```bash
# Start Jaeger with OTEL receiver
docker run --rm \
  -p 4317:4317 \
  -p 6831:6831/udp \
  -p 16686:16686 \
  jaegertracing/all-in-one

# In another terminal, point snapshot-cli to Jaeger
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
snapshot-cli snapshot create --volume-id abc-123 --name test-snap

# View traces in Jaeger UI
open http://localhost:16686
```

## Instrumented Operations

The following operations emit distributed traces:

### Authentication
- `auth.keystone.authenticate` — Keystone v3 authentication with project scope
- `auth.block_storage.init` — Cinder v3 client initialization
- `auth.shared_filesystem.init` — Manila v2 client initialization

### Snapshot Operations
- `snapshot.create` — Create snapshot (attributes: volume_id/share_id, name)
- `snapshot.delete` — Delete snapshot (attributes: snapshot_id)
- `snapshot.list` — List snapshots (attributes: volume_id/share_id)
- `snapshot.get` — Get snapshot details (attributes: snapshot_id)
- `snapshot.cleanup` — Cleanup old snapshots (attributes: volume_id/share_id, older_than_seconds)

## Span Attributes

All spans include relevant context:

```
snapshot.create:
  - snapshot.volume_id: "abc-123" (if volume snapshot)
  - snapshot.share_id: "def-456" (if share snapshot)
  - snapshot.name: "snap-202603081200"

snapshot.cleanup:
  - snapshot.volume_id or snapshot.share_id (target resource)
  - snapshot.older_than_seconds: 604800 (7 days)
```

## Error Handling

Errors are recorded in spans with full error messages. Failed operations show span status as ERROR with error details.

Example error span:
```
snapshot.delete: ERROR
  error: "snapshot not found: xyz-789"
```

## Testing

Observability is compatible with existing test suites. Mock OTEL collectors are used in integration tests.

## Performance Impact

- When disabled: **Zero overhead** (no SDK initialization, no span creation)
- When enabled: <2% overhead for typical operations (depends on collector latency)

## Best Practices

1. **Always use external collector** — Don't expect traces to export directly from CLI
2. **Configure sampling in collector** — CLI traces all operations; filter in backend
3. **Use with JSON output** — Combine `--output json` with trace IDs for correlation
4. **Monitor collector health** — Slow collectors can impact CLI performance

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No traces appear | Verify `OTEL_EXPORTER_OTLP_ENDPOINT` is set and collector is reachable |
| CLI hangs on snapshot operations | Collector may be slow/unreachable; check network/firewall |
| Traces incomplete | Ensure collector has `OTLP gRPC` receiver enabled (default port 4317) |

## Further Reading

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Jaeger Getting Started](https://www.jaegertracing.io/docs/getting-started/)
- [OTEL Go Instrumentation](https://opentelemetry.io/docs/instrumentation/go/)
