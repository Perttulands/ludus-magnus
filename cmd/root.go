package cmd

import "github.com/spf13/cobra"

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ludus-magnus",
		Short: "ludus-magnus CLI",
	}
}
