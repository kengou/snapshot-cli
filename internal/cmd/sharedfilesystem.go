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
			ctx := cmd.Context()
			client, err := buildSharedClient(ctx)
			if err != nil {
				return err
			}
			return sharedfilesystem.GetSharedFileSystem(ctx, cmd.Flag("share-id").Value.String(), cmd.Flag("output").Value.String(), client)
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
			ctx := cmd.Context()
			client, err := buildSharedClient(ctx)
			if err != nil {
				return err
			}
			return sharedfilesystem.ListSharedFileSystems(ctx, cmd.Flag("output").Value.String(), client)
		},
	}
	cmd.Flags().String("output", "json", "output format: json (default), table")
	doNotSortFlags(cmd)

	return cmd
}
