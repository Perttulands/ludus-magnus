package cmd

import (
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newEvaluateCmd() *cobra.Command {
	var score int
	var comment string

	cmd := &cobra.Command{
		Use:   "evaluate <artifact-id>",
		Short: "Evaluate one artifact with score and optional comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactID := strings.TrimSpace(args[0])
			if artifactID == "" {
				return fmt.Errorf("artifact id is required")
			}

			if err := state.EvaluateArtifact(artifactID, score, comment); err != nil {
				return err
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{
					"artifact_id": artifactID,
					"score":       score,
					"comment":     comment,
				})
			}

			_, err := fmt.Fprintf(cmd.OutOrStdout(), "Artifact %s evaluated: %d/10\n", artifactID, score)
			return err
		},
	}

	cmd.Flags().IntVar(&score, "score", 0, "Evaluation score (1-10)")
	cmd.Flags().StringVar(&comment, "comment", "", "Optional evaluation comment")
	_ = cmd.MarkFlagRequired("score")

	return cmd
}
