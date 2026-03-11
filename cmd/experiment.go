package cmd

import "github.com/spf13/cobra"

func newExperimentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiment",
		Short: "Experiment management commands",
	}

	cmd.AddCommand(newExperimentAnalyzeCmd())
	cmd.AddCommand(newExperimentScoreCmd())
	cmd.AddCommand(newExperimentRunCmd())

	return cmd
}
