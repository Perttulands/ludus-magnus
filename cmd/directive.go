package cmd

import "github.com/spf13/cobra"

func newDirectiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "directive",
		Short: "Manage lineage directives",
	}

	cmd.AddCommand(newDirectiveSetCmd())
	cmd.AddCommand(newDirectiveClearCmd())
	return cmd
}
