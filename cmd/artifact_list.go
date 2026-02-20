package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newArtifactListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <session-id>",
		Short: "List all artifacts for a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := strings.TrimSpace(args[0])
			st, err := state.Load("")
			if err != nil {
				return err
			}

			session, ok := st.Sessions[sessionID]
			if !ok {
				return fmt.Errorf("session not found: %s", sessionID)
			}

			type artifactSummary struct {
				ID           string `json:"id"`
				AgentVersion int    `json:"agent_version"`
				Score        *int   `json:"score,omitempty"`
				CreatedAt    string `json:"created_at"`
			}

			summaries := []artifactSummary{}
			for _, lineage := range session.Lineages {
				for _, artifact := range lineage.Artifacts {
					summary := artifactSummary{
						ID:           artifact.ID,
						AgentVersion: agentVersionForArtifact(lineage, artifact.AgentID),
						CreatedAt:    artifact.CreatedAt,
					}
					if artifact.Evaluation != nil {
						score := artifact.Evaluation.Score
						summary.Score = &score
					}
					summaries = append(summaries, summary)
				}
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{"artifacts": summaries})
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			if _, err := fmt.Fprintln(tw, "ID\tAgent Version\tScore\tCreated At"); err != nil {
				return fmt.Errorf("write artifact table header: %w", err)
			}
			for _, summary := range summaries {
				score := "-"
				if summary.Score != nil {
					score = strconv.Itoa(*summary.Score)
				}
				if _, err := fmt.Fprintf(
					tw,
					"%s\t%d\t%s\t%s\n",
					summary.ID,
					summary.AgentVersion,
					score,
					summary.CreatedAt,
				); err != nil {
					return fmt.Errorf("write artifact table row %q: %w", summary.ID, err)
				}
			}
			return tw.Flush()
		},
	}

	return cmd
}

func agentVersionForArtifact(lineage state.Lineage, agentID string) int {
	for _, agent := range lineage.Agents {
		if agent.ID == agentID {
			return agent.Version
		}
	}
	return 0
}
