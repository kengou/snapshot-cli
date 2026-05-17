package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sapcc/go-api-declarations/bininfo"

	"snapshot-cli/internal/cmd"
	"snapshot-cli/internal/observability"
)

func main() {
	ctx := context.Background()

	tp, err := observability.InitTracerProvider(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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
	cmd.Execute(ctx, v)
}
