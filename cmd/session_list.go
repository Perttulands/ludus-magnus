package cmd

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newSessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load("")
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, '\t', 0)
			if _, err := fmt.Fprintln(w, "ID\tMODE\tSTATUS\tCREATED_AT"); err != nil {
				return err
			}

			ids := make([]string, 0, len(st.Sessions))
			for id := range st.Sessions {
				ids = append(ids, id)
			}
			sort.Strings(ids)

			for _, id := range ids {
				ses := st.Sessions[id]
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ses.ID, ses.Mode, ses.Status, ses.CreatedAt); err != nil {
					return err
				}
			}

			return w.Flush()
		},
	}
}
