package cmd

import (
	"snapshot-cli/internal/blockstorage"
	"snapshot-cli/internal/snapshot"

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

// newGetBlockCmd returns the "volumes get" subcommand.
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
	// H3: define --output so cmd.Flag("output") never returns nil.
	cmd.Flags().String("output", "json", "output format: json (default), table")
	_ = cmd.MarkFlagRequired("volume-id") //nolint:errcheck
	doNotSortFlags(cmd)

	return cmd
}

// newListBlockCmd returns the "volumes list" subcommand.
func newListBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return blockstorage.RunListBlockStorage(cmd.Context(), cmd.Flag("output").Value.String())
		},
	}
	// H3: define --output so cmd.Flag("output") never returns nil.
	cmd.Flags().String("output", "json", "output format: json (default), table")
	doNotSortFlags(cmd)

	return cmd
}

// newSnapshotBlockCmd returns the "volumes snapshot" subcommand.
// H4: delegates to snapshot.CreateSnapshotCmd to eliminate duplicate logic.
// L7: uses --description flag name (consistent with "snapshot create").
func newSnapshotBlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Create a snapshot of a block storage volume",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapShotOpts := &snapshot.SnapShotOpts{
				VolumeID:    cmd.Flag("volume-id").Value.String(),
				Force:       cmd.Flag("force").Value.String() == "true",
				Name:        cmd.Flag("name").Value.String(),
				Description: cmd.Flag("description").Value.String(),
			}
			return snapshot.CreateSnapshotCmd(cmd.Context(), snapShotOpts, cmd.Flag("output").Value.String())
		},
	}
	cmd.Flags().String("volume-id", "", "ID of the block storage volume to snapshot")
	cmd.Flags().String("name", "", "Name of the snapshot (optional)")
	cmd.Flags().String("description", "", "Description of the snapshot (optional)")
	cmd.Flags().Bool("force", false, "Force snapshot creation")
	cmd.Flags().String("output", "json", "output format: json (default), table")
	_ = cmd.MarkFlagRequired("volume-id") //nolint:errcheck
	doNotSortFlags(cmd)

	return cmd
}
