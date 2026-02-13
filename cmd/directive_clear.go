package cmd

import (
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newDirectiveClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear <session-id> <lineage-name> <directive-id>",
		Short: "Remove a directive from one lineage",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := strings.TrimSpace(args[0])
			lineageName := strings.TrimSpace(args[1])
			directiveID := strings.TrimSpace(args[2])
			if sessionID == "" || lineageName == "" || directiveID == "" {
				return fmt.Errorf("session id, lineage name, and directive id are required")
			}

			st, err := state.Load("")
			if err != nil {
				return err
			}

			session, ok := st.Sessions[sessionID]
			if !ok {
				return fmt.Errorf("session %q not found", sessionID)
			}

			lineageKey, lineage, ok := findLineageByName(session, lineageName)
			if !ok {
				return fmt.Errorf("lineage %q not found", lineageName)
			}

			var removed bool
			lineage.Directives.Sticky, removed = removeDirectiveByID(lineage.Directives.Sticky, directiveID)
			if !removed {
				lineage.Directives.Oneshot, removed = removeDirectiveByID(lineage.Directives.Oneshot, directiveID)
			}
			if !removed {
				return fmt.Errorf("directive %q not found", directiveID)
			}

			session.Lineages[lineageKey] = lineage
			st.Sessions[sessionID] = session

			if err := state.Save("", st); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "directive_id=%s cleared\n", directiveID)
			return err
		},
	}
}

func removeDirectiveByID(directives []state.Directive, directiveID string) ([]state.Directive, bool) {
	for i, directive := range directives {
		if directive.ID != directiveID {
			continue
		}

		updated := make([]state.Directive, 0, len(directives)-1)
		updated = append(updated, directives[:i]...)
		updated = append(updated, directives[i+1:]...)
		return updated, true
	}
	return directives, false
}
