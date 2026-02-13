package cmd

import (
	"fmt"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newQuickstartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quickstart",
		Short: "Manage quickstart flows",
	}

	cmd.AddCommand(newQuickstartInitCmd())
	return cmd
}

func newQuickstartInitCmd() *cobra.Command {
	var need string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a quickstart session",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load("")
			if err != nil {
				return err
			}

			now := time.Now().UTC().Format(time.RFC3339)
			sessionID := newPrefixedID("ses")
			lineageID := newPrefixedID("lin")

			mainLineage := state.Lineage{
				ID:         lineageID,
				SessionID:  sessionID,
				Name:       "main",
				Locked:     false,
				Agents:     []state.Agent{},
				Artifacts:  []state.Artifact{},
				Directives: state.Directives{Oneshot: []state.Directive{}, Sticky: []state.Directive{}},
			}

			st.Sessions[sessionID] = state.Session{
				ID:        sessionID,
				Mode:      "quickstart",
				Need:      need,
				CreatedAt: now,
				Status:    "active",
				Lineages:  map[string]state.Lineage{lineageID: mainLineage},
			}

			if err := state.Save("", st); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "session_id=%s\n", sessionID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "lineage_id=%s\n", lineageID)
			return err
		},
	}

	cmd.Flags().StringVar(&need, "need", "", "Intent for the session")
	_ = cmd.MarkFlagRequired("need")

	return cmd
}
