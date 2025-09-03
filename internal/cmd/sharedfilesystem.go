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

/*
Get command to retrieve nfs storage information.
*/
func newGetNfsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get nfs storage information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return sharedfilesystem.RunGetSharedFileSystem(cmd.Context(), cmd.Flag("share-id").Value.String(), cmd.Flag("output").Value.String())
		},
	}
	cmd.Flags().String("share-id", "", "ID of the block storage volume to retrieve")
	cmd.MarkFlagRequired("share-id")
	doNotSortFlags(cmd)

	return cmd
}

/*
List command to list nfs storage resources.
*/
func newListNfsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage resources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return sharedfilesystem.RunListSharedFileSystemSnapshots(cmd.Context(), cmd.Flag("output").Value.String())
		},
	}
	doNotSortFlags(cmd)

	return cmd
}
