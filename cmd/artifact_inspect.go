package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newArtifactInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <artifact-id>",
		Short: "Inspect one artifact in detail",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactID := strings.TrimSpace(args[0])
			if artifactID == "" {
				return fmt.Errorf("artifact id is required")
			}

			st, err := state.Load("")
			if err != nil {
				return err
			}

			for _, session := range st.Sessions {
				for _, lineage := range session.Lineages {
					for _, artifact := range lineage.Artifacts {
						if artifact.ID != artifactID {
							continue
						}

						payload, err := json.MarshalIndent(artifact, "", "  ")
						if err != nil {
							return err
						}
						_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", payload)
						return err
					}
				}
			}

			return fmt.Errorf("artifact %q not found", artifactID)
		},
	}

	return cmd
}
