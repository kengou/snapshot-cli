package cmd

import (
	"fmt"
	"os"
	"time"

	"snapshot-cli/internal/snapshot"

	"github.com/spf13/cobra"
)

// minOlderThan is the smallest allowed value for --older-than to avoid a
// runaway deletion if someone passes 0 or a trivially small duration.
const minOlderThan = time.Hour

// newCleanupCmd returns the "cleanup" subcommand.
func newCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			olderThan, err := cmd.Flags().GetDuration("older-than")
			if err != nil {
				return fmt.Errorf("invalid --older-than: %w", err)
			}
			if olderThan < minOlderThan {
				return fmt.Errorf("--older-than must be at least %s to avoid accidental mass deletion", minOlderThan)
			}

			volume, _ := cmd.Flags().GetBool("volume")  //nolint:errcheck // flag defined below
			share, _ := cmd.Flags().GetBool("share")    //nolint:errcheck // flag defined below
			dryRun, _ := cmd.Flags().GetBool("dry-run") //nolint:errcheck // flag defined below

			ctx := cmd.Context()
			client, err := clientForKind(ctx, volume, share)
			if err != nil {
				return err
			}

			snapShotOpts := &snapshot.SnapShotOpts{
				Volume:    volume,
				VolumeID:  cmd.Flag("volume-id").Value.String(),
				Share:     share,
				ShareID:   cmd.Flag("share-id").Value.String(),
				OlderThan: olderThan.String(),
				DryRun:    dryRun,
			}

			scope := "all volumes"
			if snapShotOpts.VolumeID != "" {
				scope = "volume " + snapShotOpts.VolumeID
			} else if snapShotOpts.ShareID != "" {
				scope = "share " + snapShotOpts.ShareID
			}
			prefix := "cleanup: will delete"
			if dryRun {
				prefix = "cleanup [dry-run]: would delete"
			}
			if _, werr := fmt.Fprintf(os.Stderr, "%s snapshots older than %s in %s\n", prefix, olderThan, scope); werr != nil {
				return werr
			}

			return snapshot.CleanupSnapshot(ctx, snapShotOpts, cmd.Flag("output").Value.String(), client)
		},
	}

	cmd.Flags().Bool("share", false, "clean up shared filesystem snapshots")
	cmd.Flags().Bool("volume", false, "clean up block storage snapshots")
	cmd.Flags().String("volume-id", "", "restrict cleanup to snapshots of this volume")
	cmd.Flags().String("share-id", "", "restrict cleanup to snapshots of this share")
	cmd.Flags().Duration("older-than", snapshot.DefaultOlderThan, "delete snapshots older than this duration, e.g. 168h (7 days), 720h (30 days)")
	cmd.Flags().Bool("dry-run", false, "list snapshots that would be deleted without deleting them")
	cmd.Flags().String("output", "json", "output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	doNotSortFlags(cmd)

	return cmd
}
