package main

import (
	"context"

	"github.com/sapcc/go-api-declarations/bininfo"

	"snapshot-cli/internal/cmd"
)

func main() {
	ctx := context.Background()

	v := &cmd.VersionInfo{
		Version:       bininfo.VersionOr("dev"),
		GitCommitHash: bininfo.CommitOr("unknown"),
		BuildDate:     bininfo.BuildDateOr("now"),
	}
	cmd.Execute(ctx, v)
}
