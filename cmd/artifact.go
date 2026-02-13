package cmd

import "github.com/spf13/cobra"

func newArtifactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Inspect stored execution artifacts",
	}

	cmd.AddCommand(newArtifactListCmd())
	cmd.AddCommand(newArtifactInspectCmd())

	return cmd
}
