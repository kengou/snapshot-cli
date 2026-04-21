package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"snapshot-cli/internal/config"
)

// VersionInfo holds build-time version metadata injected via ldflags or bininfo.
type VersionInfo struct {
	Version       string
	GitCommitHash string
	BuildDate     string
}

// skipVersionCheck is set by the persistent --skip-version-check flag and read
// by auth config construction to bypass Cinder/Manila version detection.
var skipVersionCheck bool

func Execute(ctx context.Context, v *VersionInfo) {
	if err := newRootCmd(v).ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "snapshot-cli:", err)
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

	cmd.PersistentFlags().BoolVar(&skipVersionCheck, "skip-version-check", false,
		"skip OpenStack Cinder v3 / Manila v2 endpoint detection")

	cmd.PersistentPreRunE = func(*cobra.Command, []string) error {
		config.SetSkipVersionCheck(skipVersionCheck)
		return nil
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
