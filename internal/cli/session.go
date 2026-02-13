package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Perttulands/agent-academy/internal/session"
	"github.com/Perttulands/agent-academy/internal/store"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newSessionCmd() *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Session management commands",
	}

	sessionCmd.AddCommand(newSessionNewCmd())
	sessionCmd.AddCommand(newSessionListCmd())

	return sessionCmd
}

func newSessionNewCmd() *cobra.Command {
	var (
		need string
		mode string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new session",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, closeFn, err := newSessionManager()
			if err != nil {
				return err
			}
			defer closeFn()

			created, err := manager.Create(context.Background(), need, mode)
			if err != nil {
				return err
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Mode", "Need", "Created At"})
			table.Append([]string{created.ID, created.Mode, created.Need, created.CreatedAt.Format(time.RFC3339)})
			table.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&need, "need", "", "Description of the user need")
	cmd.Flags().StringVar(&mode, "mode", "quickstart", "Session mode")
	_ = cmd.MarkFlagRequired("need")

	return cmd
}

func newSessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List existing sessions",
		RunE: func(_ *cobra.Command, _ []string) error {
			manager, closeFn, err := newSessionManager()
			if err != nil {
				return err
			}
			defer closeFn()

			sessions, err := manager.List(context.Background())
			if err != nil {
				return err
			}

			if len(sessions) == 0 {
				fmt.Println("No sessions found.")
				return nil
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Mode", "Need", "Created At"})
			for _, item := range sessions {
				table.Append([]string{item.ID, item.Mode, item.Need, item.CreatedAt.Format(time.RFC3339)})
			}
			table.Render()
			return nil
		},
	}
}

func newSessionManager() (*session.Manager, func(), error) {
	dbPath := viper.GetString("db")
	if dbPath == "" {
		return nil, nil, fmt.Errorf("database path is empty")
	}

	st, err := store.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}

	return session.NewManager(st), func() {
		_ = st.Close()
	}, nil
}
