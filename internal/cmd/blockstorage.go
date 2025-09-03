package cmd

import (
	"snapshot-cli/internal/blockstorage"

	"github.com/spf13/cobra"
)

func newBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volumes",
		Short: "Manage blockstorage volumes",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newGetBlockCmd())
	cmd.AddCommand(newListBlockCmd())
	cmd.AddCommand(newSnapshotBlockCmd())
	doNotSortFlags(cmd)

	return cmd
}

/*
Get command to retrieve block storage information.
*/
func newGetBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get block storage information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return blockstorage.RunGetBlockStorage(cmd.Context(), cmd.Flag("volume-id").Value.String(), cmd.Flag("output").Value.String())
		},
	}
	cmd.Flags().String("volume-id", "", "ID of the block storage volume to retrieve")
	cmd.MarkFlagRequired("volume-id")
	doNotSortFlags(cmd)

	return cmd
}

/*
List command to list block storage resources.
*/
func newListBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return blockstorage.RunListBlockStorage(cmd.Context(), cmd.Flag("output").Value.String())
		},
	}
	doNotSortFlags(cmd)

	return cmd
}

/*
Snapshot command to create a snapshot of a volume.
*/
func newSnapshotBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Create a snapshot of a block storage volume",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &blockstorage.SnapShotOpts{
				VolumeID:    cmd.Flag("volume-id").Value.String(),
				Force:       cmd.Flag("force").Value.String() == "true",
				Name:        cmd.Flag("name").Value.String(),
				Description: cmd.Flag("snapshot-dscr").Value.String(),
			}
			return blockstorage.CreateSnapshotBlockStorage(cmd.Context(), snapShotOpts)
		},
	}
	cmd.Flags().String("volume-id", "", "ID of the block storage volume to snapshot")
	cmd.Flags().String("snapshot-name", "", "Name of the snapshot (optional)")
	cmd.Flags().String("snapshot-dscr", "", "Description of the snapshot (optional)")
	cmd.Flags().Bool("force", false, "Force snapshot creation")
	cmd.MarkFlagRequired("volume-id")
	doNotSortFlags(cmd)

	return cmd
}
