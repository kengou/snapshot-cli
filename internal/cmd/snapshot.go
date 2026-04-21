package cmd

import (
	"context"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/spf13/cobra"

	"snapshot-cli/internal/snapshot"
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

// clientForKind returns the appropriate gophercloud client based on the
// volume/share boolean pair already validated by cobra's mutually-exclusive flags.
func clientForKind(ctx context.Context, volume, share bool) (*gophercloud.ServiceClient, error) {
	switch {
	case volume:
		return buildBlockClient(ctx)
	case share:
		return buildSharedClient(ctx)
	default:
		return nil, errors.New("must set --volume or --share")
	}
}

// newGetSnapshotCmd returns the "snapshot get" subcommand.
func newGetSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get details of a snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			volume, _ := cmd.Flags().GetBool("volume") //nolint:errcheck // flag defined below
			share, _ := cmd.Flags().GetBool("share")   //nolint:errcheck // flag defined below
			client, err := clientForKind(ctx, volume, share)
			if err != nil {
				return err
			}
			snapShotOpts := &snapshot.SnapShotOpts{
				SnapshotID: cmd.Flag("snapshot-id").Value.String(),
				Volume:     volume,
				Share:      share,
			}
			return snapshot.GetSnapshotCmd(ctx, snapShotOpts, cmd.Flag("output").Value.String(), client)
		},
	}

	cmd.Flags().String("snapshot-id", "", "ID of the snapshot to retrieve")
	cmd.Flags().Bool("volume", false, "retrieve a block storage snapshot")
	cmd.Flags().Bool("share", false, "retrieve a shared filesystem snapshot")
	cmd.Flags().String("output", "json", "output format: json (default), table")
	_ = cmd.MarkFlagRequired("snapshot-id") //nolint:errcheck
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	doNotSortFlags(cmd)

	return cmd
}

func newListSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			volume, _ := cmd.Flags().GetBool("volume") //nolint:errcheck // flag defined below
			share, _ := cmd.Flags().GetBool("share")   //nolint:errcheck // flag defined below
			client, err := clientForKind(ctx, volume, share)
			if err != nil {
				return err
			}
			snapShotOpts := &snapshot.SnapShotOpts{
				Volume: volume,
				Share:  share,
			}
			return snapshot.ListSnapshotsCmd(ctx, snapShotOpts, cmd.Flag("output").Value.String(), client)
		},
	}

	cmd.Flags().Bool("share", false, "list shared filesystem snapshots")
	cmd.Flags().Bool("volume", false, "list volume snapshots")
	cmd.Flags().String("output", "json", "output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	doNotSortFlags(cmd)

	return cmd
}

func newCreateSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a snapshot of a volume or shared filesystem",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			volumeID := cmd.Flag("volume-id").Value.String()
			shareID := cmd.Flag("share-id").Value.String()
			client, err := clientForKind(ctx, volumeID != "", shareID != "")
			if err != nil {
				return err
			}
			snapShotOpts := &snapshot.SnapShotOpts{
				VolumeID:    volumeID,
				ShareID:     shareID,
				Force:       cmd.Flag("force").Value.String() == defaultTrue,
				Name:        cmd.Flag("name").Value.String(),
				Description: cmd.Flag("description").Value.String(),
				Cleanup:     cmd.Flag("cleanup").Value.String() == defaultTrue,
				OlderThan:   cmd.Flag("older-than").Value.String(),
			}
			return snapshot.CreateSnapshotCmd(ctx, snapShotOpts, cmd.Flag("output").Value.String(), client)
		},
	}

	cmd.Flags().String("volume-id", "", "ID of the volume to snapshot")
	cmd.Flags().String("share-id", "", "ID of the shared filesystem to snapshot")
	cmd.Flags().Bool("force", false, "Force snapshot creation (block only)")
	cmd.Flags().Bool("cleanup", false, "Cleanup old snapshots after creation")
	cmd.Flags().Duration("older-than", snapshot.DefaultOlderThan, "delete snapshots older than this duration, e.g. 168h (7 days), 720h (30 days)")
	cmd.Flags().String("name", "", "Name of the snapshot")
	cmd.Flags().String("description", "", "Description of the snapshot")
	cmd.Flags().String("output", "json", "output format: json (default), table")
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
			ctx := cmd.Context()
			volume, _ := cmd.Flags().GetBool("volume") //nolint:errcheck // flag defined below
			share, _ := cmd.Flags().GetBool("share")   //nolint:errcheck // flag defined below
			client, err := clientForKind(ctx, volume, share)
			if err != nil {
				return err
			}
			snapShotOpts := &snapshot.SnapShotOpts{
				SnapshotID: cmd.Flag("snapshot-id").Value.String(),
				Volume:     volume,
				Share:      share,
			}
			return snapshot.DeleteSnapshotCmd(ctx, snapShotOpts, cmd.Flag("output").Value.String(), client)
		},
	}

	cmd.Flags().Bool("share", false, "delete a shared filesystem snapshot")
	cmd.Flags().Bool("volume", false, "delete a block storage snapshot")
	cmd.Flags().String("snapshot-id", "", "ID of the snapshot to delete")
	cmd.Flags().String("output", "json", "output format: json (default), table")
	cmd.MarkFlagsOneRequired("volume", "share")
	cmd.MarkFlagsMutuallyExclusive("volume", "share")
	_ = cmd.MarkFlagRequired("snapshot-id") //nolint:errcheck
	doNotSortFlags(cmd)

	return cmd
}
