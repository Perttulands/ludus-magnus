package cmd

import "github.com/spf13/cobra"

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chiron",
		Short: "Chiron — train AI agents through iterative evaluation",
	}
	cmd.PersistentFlags().Bool("json", false, "Output JSON")

	cmd.AddCommand(newSessionCmd())
	cmd.AddCommand(newQuickstartCmd())
	cmd.AddCommand(newTrainingCmd())
	cmd.AddCommand(newLineageCmd())
	cmd.AddCommand(newIterateCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newEvaluateCmd())
	cmd.AddCommand(newArtifactCmd())
	cmd.AddCommand(newPromoteCmd())
	cmd.AddCommand(newDirectiveCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newDoctorCmd())

	return cmd
}
