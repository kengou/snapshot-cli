# Contributing to snapshot-cli

## Development setup

**Prerequisites:** Go 1.26+, Docker (optional, for container builds).

```bash
git clone https://github.com/kengou/snapshot-cli.git
cd snapshot-cli
go mod download
make build          # bin/snapshot-cli
```

## Project layout

```
snapshot-cli/
├── cmd/snapshot-cli/main.go        # entry point
├── internal/
│   ├── auth/                       # Keystone authentication
│   ├── cmd/                        # cobra command definitions
│   ├── config/                     # env-var based auth config
│   ├── blockstorage/               # Cinder (block storage) operations
│   ├── sharedfilesystem/           # Manila (NFS) operations
│   ├── snapshot/                   # cross-service snapshot operations
│   ├── observability/              # OpenTelemetry tracing setup
│   └── util/                       # shared output helpers
├── charts/                         # Helm chart
└── Dockerfile
```

Key design rules:
- All commands authenticate via `config.ReadAuthConfig()` (reads `OS_*` env vars)
- New commands belong in `internal/cmd/`; business logic in the relevant service package
- Output is always written via `util.Render(output, data, header)`; never `fmt.Print` in service functions

## Running tests

```bash
make test                           # run all tests
go test ./internal/... -v           # verbose, with test names
go test ./internal/... -run TestFoo # run a single test
```

Test files live next to the code they test (`auth_test.go` beside `auth.go`).
Tests that need env vars use `t.Setenv()`; no `.env` files or shared global state.

## Running the linter

```bash
make lint
```

The project uses [golangci-lint](https://golangci-lint.run/) v2 with the config in `.golangci.yaml`.
All 35 enabled linters must pass with zero issues before a PR can merge.

Common lint fixes:

| Error | Fix |
|-------|-----|
| `File is not properly formatted (gofmt)` | `gofmt -w <file>` |
| `Error return value is not checked (errcheck)` | Assign error or use `//nolint:errcheck` with justification |
| `declaration of "err" shadows declaration at` | Already excluded by config |

## Adding a new command

1. Create a `newXxxCmd()` function in `internal/cmd/`.
2. Register it in `newRootCmd()` (or the appropriate parent command) in `root.go`.
3. Put business logic in the relevant service package (`blockstorage/`, `snapshot/`, etc.).
4. Use `cmd.MarkFlagsOneRequired` / `cmd.MarkFlagsMutuallyExclusive` for exclusive flags.
5. Render output through `util.Render`, which switches between `"table"` and `"json"`:
   ```go
   return util.Render(output, result, header)
   ```
6. Add a godoc comment to every exported function.
7. Add or extend tests in `*_test.go` files.

## Service availability

Commands only require the OpenStack service they actually use: endpoint
resolution via gophercloud (`openstack.NewBlockStorageV3` / `NewSharedFileSystemV2`)
fails with a descriptive error when the service is missing from the catalog.
Do not add upfront checks that require unrelated services to be present.

## Observability & Tracing

snapshot-cli uses OpenTelemetry for distributed tracing. Add tracing to core operations to enable production observability.

**How tracing works:**
- Spans track operation duration, attributes, and error status
- Spans are exported to an OTEL collector (gRPC protocol) if `OTEL_EXPORTER_OTLP_ENDPOINT` is set
- No overhead when tracing is disabled

**Adding tracing to a module:**

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("github.com/kengou/snapshot-cli/internal/mymodule")

func MyOperation(ctx context.Context, param string) (err error) {
    ctx, span := tracer.Start(ctx, "my_operation")
    defer func() {
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        } else {
            span.SetStatus(codes.Ok, "")
        }
        span.End()
    }()

    // ... operation code ...
    return err
}
```

**Span conventions:**
- Operation names: lowercase, snake_case (e.g., `snapshot.create`, `auth.keystone.authenticate`)
- Attributes: use semantic attribute keys (e.g., `snapshot.id`, `snapshot.volume_id`)
- Error handling: record errors before returning

See [`internal/observability/`](internal/observability/) for example implementations.

## Adding tests

Unit tests for pure functions (no OpenStack dependency) go directly in the package.
Integration tests (requiring a real OpenStack) are not in scope for this repo's CI.

```go
func TestMyFunc_SomeScenario(t *testing.T) {
    got := MyFunc(input)
    if got != want {
        t.Errorf("MyFunc(%v) = %v, want %v", input, got, want)
    }
}
```

Avoid global state; use `t.Setenv()` for environment variables and `os.Pipe()` to capture stdout.

## Commit messages

Follow the existing style: lowercase imperative subject, 72-char limit, reference issues where relevant.

```
fix: use blockSnapshot.Delete for block storage cleanup
feat: add --dry-run flag to cleanup command
docs: update README with new flag names
```

## Pull request checklist

- [ ] `make build` succeeds
- [ ] `make lint` reports 0 issues
- [ ] `make test` passes
- [ ] Godoc comments added to any new exported symbols
- [ ] README updated if CLI flags or behaviour changed
