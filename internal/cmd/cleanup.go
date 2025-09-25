package cmd

import "github.com/spf13/cobra"

func newClenaupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup snapshots",
		Args:  cobra.NoArgs,
	}

	doNotSortFlags(cmd)

	return cmd
}
