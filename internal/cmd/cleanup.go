package cmd

import (
	"snapshot-cli/internal/snapshot"

	"github.com/spf13/cobra"
)

func newClenaupCmd() *cobra.Command {
	var DefaultOlderThan string
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				Volume:    cmd.Flag("volume").Value.String() == "true",
				Share:     cmd.Flag("share").Value.String() == "true",
				OlderThan: cmd.Flag("older-than").Value.String(),
			}
			return snapshot.CleanupSnapshot(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}

	cmd.Flags().Bool("share", false, "list shared filesystem snapshots")
	cmd.Flags().Bool("volume", false, "list volume snapshots")
	cmd.Flags().Duration("older-than", snapshot.ParseDurationOrFallback(DefaultOlderThan), "Duration to identify old snapshots, e.g. 168h (7 days), 720h (30 days)")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	doNotSortFlags(cmd)

	return cmd
}
