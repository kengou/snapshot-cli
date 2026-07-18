package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sapcc/go-api-declarations/bininfo"

	"github.com/kengou/snapshot-cli/internal/cmd"
	"github.com/kengou/snapshot-cli/internal/observability"
)

func main() {
	os.Exit(run())
}

// run wraps command execution so that deferred cleanup (trace flushing) runs
// before the process exits — calling os.Exit directly in main would skip it.
func run() int {
	ctx := context.Background()

	tp, err := observability.InitTracerProvider(ctx)
	if err != nil {
		// A broken tracing setup must not prevent the CLI from working.
		fmt.Fprintln(os.Stderr, "snapshot-cli: tracing disabled:", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := observability.Shutdown(shutdownCtx, tp); shutdownErr != nil {
			fmt.Fprintln(os.Stderr, shutdownErr)
		}
	}()

	v := &cmd.VersionInfo{
		Version:       bininfo.VersionOr("dev"),
		GitCommitHash: bininfo.CommitOr("unknown"),
		BuildDate:     bininfo.BuildDateOr("now"),
	}
	if execErr := cmd.Execute(ctx, v); execErr != nil {
		fmt.Fprintln(os.Stderr, "snapshot-cli:", execErr)
		return 1
	}
	return 0
}
