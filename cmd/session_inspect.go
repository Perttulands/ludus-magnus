package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newSessionInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <session-id>",
		Short: "Inspect a session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load("")
			if err != nil {
				return err
			}

			sessionID := args[0]
			ses, ok := st.Sessions[sessionID]
			if !ok {
				return fmt.Errorf("session %q not found", sessionID)
			}

			data, err := json.MarshalIndent(ses, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal session: %w", err)
			}

			_, err = cmd.OutOrStdout().Write(append(data, '\n'))
			return err
		},
	}
}
