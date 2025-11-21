package cmd

import (
	"snapshot-cli/internal/snapshot"

	"github.com/spf13/cobra"
)

const (
	defaultTrue = "true"
)

func newSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Snapshot management commands",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newGetSnapshotCmd())
	cmd.AddCommand(newListSnapshotCmd())
	cmd.AddCommand(newCreateSnapshotCmd())
	cmd.AddCommand(newDeleteSnapshotCmd())
	doNotSortFlags(cmd)

	return cmd
}

func newGetSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get details of a snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				VolumeID: cmd.Flag("volume-id").Value.String(),
				ShareID:  cmd.Flag("share-id").Value.String(),
			}
			return snapshot.GetSnapshotCmd(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}

	cmd.Flags().String("share-id", "", "ID of the shared filesystem associated with the snapshot")
	cmd.Flags().String("volume-id", "", "ID of the volume associated with the snapshot")
	cmd.Flags().String("output", "json", "Output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume-id", "share-id")
	cmd.MarkFlagsMutuallyExclusive("volume-id", "share-id")
	doNotSortFlags(cmd)

	return cmd
}

func newListSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				Volume: cmd.Flag("volume").Value.String() == defaultTrue,
				Share:  cmd.Flag("share").Value.String() == defaultTrue,
			}
			return snapshot.ListSnapshotsCmd(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}

	cmd.Flags().Bool("share", false, "list shared filesystem snapshots")
	cmd.Flags().Bool("volume", false, "list volume snapshots")
	cmd.Flags().String("output", "json", "Output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	doNotSortFlags(cmd)

	return cmd
}

func newCreateSnapshotCmd() *cobra.Command {
	var DefaultOlderThan string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a snapshot of a volume or shared filesystem",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				VolumeID:    cmd.Flag("volume-id").Value.String(),
				ShareID:     cmd.Flag("share-id").Value.String(),
				Force:       cmd.Flag("force").Value.String() == defaultTrue,
				Name:        cmd.Flag("name").Value.String(),
				Description: cmd.Flag("description").Value.String(),
				Cleanup:     cmd.Flag("cleanup").Value.String() == defaultTrue,
				OlderThan:   cmd.Flag("older-than").Value.String(),
			}
			return snapshot.CreateSnapshotCmd(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}

	cmd.Flags().String("volume-id", "", "ID of the volume to snapshot")
	cmd.Flags().String("share-id", "", "ID of the shared filesystem to snapshot")
	cmd.Flags().Bool("force", false, "Force snapshot creation (block only)")
	cmd.Flags().Bool("cleanup", false, "Cleanup old snapshots after creation")
	cmd.Flags().Duration("older-than", snapshot.ParseDurationOrFallback(DefaultOlderThan), "Duration to identify old snapshots, e.g. 168h (7 days), 720h (30 days)")
	cmd.Flags().String("name", "", "Name of the snapshot")
	cmd.Flags().String("description", "", "Description of the snapshot")
	cmd.Flags().String("output", "json", "Output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume-id", "share-id")
	cmd.MarkFlagsMutuallyExclusive("volume-id", "share-id")
	doNotSortFlags(cmd)

	return cmd
}

func newDeleteSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				SnapshotID: cmd.Flag("snapshot-id").Value.String(),
				Volume:     cmd.Flag("volume").Value.String() == defaultTrue,
				Share:      cmd.Flag("share").Value.String() == defaultTrue,
			}
			return snapshot.DeleteSnapshotCmd(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}

	cmd.Flags().Bool("share", false, "list shared filesystem snapshots")
	cmd.Flags().Bool("volume", false, "list volume snapshots")
	cmd.Flags().String("snapshot-id", "", "ID of the snapshot to delete")
	cmd.Flags().String("output", "json", "Output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	_ = cmd.MarkFlagRequired("snapshot-id") //nolint:errcheck
	doNotSortFlags(cmd)

	return cmd
}
