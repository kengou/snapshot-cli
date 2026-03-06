package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// VersionInfo holds build-time version metadata injected via ldflags or bininfo.
type VersionInfo struct {
	Version       string
	GitCommitHash string
	BuildDate     string
}

func Execute(ctx context.Context, v *VersionInfo) {
	if err := newRootCmd(v).ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newRootCmd(v *VersionInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot-cli",
		Short: "Command-line client for openstack-shares",
		Args:  cobra.NoArgs,
		Version: fmt.Sprintf("%s, Git commit %s, built at %s",
			v.Version, v.GitCommitHash, v.BuildDate),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	doNotSortFlags(cmd)

	cmd.AddCommand(newSharedCmd())
	cmd.AddCommand(newBlockCmd())
	cmd.AddCommand(newSnapshotCmd())
	cmd.AddCommand(newCleanupCmd())

	return cmd
}

func doNotSortFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false
	cmd.LocalFlags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	cmd.InheritedFlags().SortFlags = false
}
