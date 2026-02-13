package cmd

import (
	"time"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newSessionCreateCmd() *cobra.Command {
	var mode string
	var need string

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new session",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load("")
			if err != nil {
				return err
			}

			sessionID := newPrefixedID("ses")
			now := time.Now().UTC().Format(time.RFC3339)
			st.Sessions[sessionID] = state.Session{
				ID:        sessionID,
				Mode:      mode,
				Need:      need,
				CreatedAt: now,
				Status:    "active",
				Lineages:  map[string]state.Lineage{},
			}

			if err := state.Save("", st); err != nil {
				return err
			}

			_, err = cmd.OutOrStdout().Write([]byte(sessionID + "\n"))
			return err
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "quickstart", "Session mode")
	cmd.Flags().StringVar(&need, "need", "", "Intent for the session")

	return cmd
}
