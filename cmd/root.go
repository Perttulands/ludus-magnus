package cmd

import "github.com/spf13/cobra"

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ludus-magnus",
		Short: "ludus-magnus CLI",
	}

	cmd.AddCommand(newSessionCmd())
	cmd.AddCommand(newQuickstartCmd())
	cmd.AddCommand(newRunCmd())

	return cmd
}
