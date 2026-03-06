package cmd

import (
	"snapshot-cli/internal/sharedfilesystem"

	"github.com/spf13/cobra"
)

func newSharedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nfs",
		Short: "Manage shared filesystems storage",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(newGetNfsCmd())
	cmd.AddCommand(newListNfsCmd())
	doNotSortFlags(cmd)

	return cmd
}

// newGetNfsCmd returns the "nfs get" subcommand.
func newGetNfsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get shared filesystem information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return sharedfilesystem.RunGetSharedFileSystem(cmd.Context(), cmd.Flag("share-id").Value.String(), cmd.Flag("output").Value.String())
		},
	}
	cmd.Flags().String("share-id", "", "ID of the shared filesystem to retrieve")
	cmd.Flags().String("output", "json", "output format: json (default), table")
	_ = cmd.MarkFlagRequired("share-id") //nolint:errcheck
	doNotSortFlags(cmd)

	return cmd
}

// newListNfsCmd returns the "nfs list" subcommand.
func newListNfsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List shared filesystem resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// L5: updated to renamed function RunListSharedFileSystems.
			return sharedfilesystem.RunListSharedFileSystems(cmd.Context(), cmd.Flag("output").Value.String())
		},
	}
	cmd.Flags().String("output", "json", "output format: json (default), table")
	doNotSortFlags(cmd)

	return cmd
}
