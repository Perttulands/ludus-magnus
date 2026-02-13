package cmd

import (
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newLineageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lineage",
		Short: "Manage lineage lock state",
	}

	cmd.AddCommand(newLineageLockCmd())
	cmd.AddCommand(newLineageUnlockCmd())
	return cmd
}

func newLineageLockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lock <session-id> <lineage-name>",
		Short: "Lock one lineage",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setLineageLock(cmd, args[0], args[1], true)
		},
	}
}

func newLineageUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock <session-id> <lineage-name>",
		Short: "Unlock one lineage",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setLineageLock(cmd, args[0], args[1], false)
		},
	}
}

func setLineageLock(cmd *cobra.Command, sessionID, lineageName string, locked bool) error {
	trimmedSessionID := strings.TrimSpace(sessionID)
	trimmedLineageName := strings.TrimSpace(lineageName)
	if trimmedSessionID == "" || trimmedLineageName == "" {
		return fmt.Errorf("session id and lineage name are required")
	}

	st, err := state.Load("")
	if err != nil {
		return err
	}

	session, ok := st.Sessions[trimmedSessionID]
	if !ok {
		return fmt.Errorf("session %q not found", trimmedSessionID)
	}

	lineageKey, lineage, ok := findLineageByName(session, trimmedLineageName)
	if !ok {
		return fmt.Errorf("lineage %q not found", trimmedLineageName)
	}

	lineage.Locked = locked
	session.Lineages[lineageKey] = lineage
	st.Sessions[trimmedSessionID] = session

	if err := state.Save("", st); err != nil {
		return err
	}

	if isJSONOutput(cmd) {
		return writeJSON(cmd, map[string]any{
			"session_id": trimmedSessionID,
			"lineage":    trimmedLineageName,
			"locked":     locked,
		})
	}

	if locked {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Lineage %s locked\n", trimmedLineageName)
	} else {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Lineage %s unlocked\n", trimmedLineageName)
	}
	return err
}
