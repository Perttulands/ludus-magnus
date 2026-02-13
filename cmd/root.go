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
	cmd.AddCommand(newTrainingCmd())
	cmd.AddCommand(newLineageCmd())
	cmd.AddCommand(newIterateCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newEvaluateCmd())
	cmd.AddCommand(newArtifactCmd())
	cmd.AddCommand(newPromoteCmd())

	return cmd
}
