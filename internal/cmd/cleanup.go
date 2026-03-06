package cmd

import (
	"snapshot-cli/internal/snapshot"

	"github.com/spf13/cobra"
)

// newCleanupCmd returns the "cleanup" subcommand.
func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				Volume:    cmd.Flag("volume").Value.String() == "true",
				VolumeID:  cmd.Flag("volume-id").Value.String(),
				Share:     cmd.Flag("share").Value.String() == "true",
				ShareID:   cmd.Flag("share-id").Value.String(),
				OlderThan: cmd.Flag("older-than").Value.String(),
			}
			return snapshot.CleanupSnapshot(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}

	cmd.Flags().Bool("share", false, "clean up shared filesystem snapshots")
	cmd.Flags().Bool("volume", false, "clean up block storage snapshots")
	cmd.Flags().String("volume-id", "", "restrict cleanup to snapshots of this volume")
	cmd.Flags().String("share-id", "", "restrict cleanup to snapshots of this share")
	// M8: use the exported constant instead of an always-empty variable.
	cmd.Flags().Duration("older-than", snapshot.DefaultOlderThan, "delete snapshots older than this duration, e.g. 168h (7 days), 720h (30 days)")
	// H3: define --output so cmd.Flag("output") never returns nil.
	cmd.Flags().String("output", "json", "output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	doNotSortFlags(cmd)

	return cmd
}
