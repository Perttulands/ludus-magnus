package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newDirectiveSetCmd() *cobra.Command {
	var text string
	var oneshot bool
	var sticky bool

	cmd := &cobra.Command{
		Use:   "set <session-id> <lineage-name>",
		Short: "Add a one-shot or sticky directive to one lineage",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := strings.TrimSpace(args[0])
			lineageName := strings.TrimSpace(args[1])
			directiveText := strings.TrimSpace(text)
			if sessionID == "" || lineageName == "" {
				return fmt.Errorf("session id and lineage name are required")
			}
			if directiveText == "" {
				return fmt.Errorf("directive text is required")
			}
			if !oneshot && !sticky {
				return fmt.Errorf("must specify --oneshot or --sticky")
			}
			if oneshot && sticky {
				return fmt.Errorf("must specify exactly one of --oneshot or --sticky")
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

			directive := state.Directive{
				ID:        newPrefixedID("dir"),
				Text:      directiveText,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			}

			if oneshot {
				lineage.Directives.Oneshot = append(lineage.Directives.Oneshot, directive)
			} else {
				lineage.Directives.Sticky = append(lineage.Directives.Sticky, directive)
			}

			session.Lineages[lineageKey] = lineage
			st.Sessions[sessionID] = session

			if err := state.Save("", st); err != nil {
				return err
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{
					"directive_id": directive.ID,
					"lineage":      lineageName,
					"type":         map[bool]string{true: "oneshot", false: "sticky"}[oneshot],
				})
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "directive_id=%s\n", directive.ID)
			return err
		},
	}

	cmd.Flags().StringVar(&text, "text", "", "Directive instruction text")
	cmd.Flags().BoolVar(&oneshot, "oneshot", false, "Store as one-shot directive")
	cmd.Flags().BoolVar(&sticky, "sticky", false, "Store as sticky directive")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}
