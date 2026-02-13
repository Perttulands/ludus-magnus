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

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(tw, "ID\tAgent Version\tScore\tCreated At")
			for _, lineage := range session.Lineages {
				for _, artifact := range lineage.Artifacts {
					score := "-"
					if artifact.Evaluation != nil {
						score = strconv.Itoa(artifact.Evaluation.Score)
					}
					_, _ = fmt.Fprintf(
						tw,
						"%s\t%d\t%s\t%s\n",
						artifact.ID,
						agentVersionForArtifact(lineage, artifact.AgentID),
						score,
						artifact.CreatedAt,
					)
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
