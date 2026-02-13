package cmd

import (
	"fmt"
	"strings"

	exporter "github.com/Perttulands/ludus-magnus/internal/export"
	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export agents and evidence",
	}

	cmd.AddCommand(newExportAgentCmd())
	cmd.AddCommand(newExportEvidenceCmd())
	return cmd
}

func newExportAgentCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "agent <agent-id>",
		Short: "Export one agent definition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := strings.TrimSpace(args[0])
			if agentID == "" {
				return fmt.Errorf("agent id is required")
			}

			st, err := state.Load("")
			if err != nil {
				return err
			}

			payload, err := exporter.AgentDefinition(st, agentID, format)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), payload)
			return err
		},
	}

	cmd.Flags().StringVar(&format, "format", exporter.FormatJSON, "Export format: json, python, typescript")
	return cmd
}

func newExportEvidenceCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "evidence <session-id>",
		Short: "Export one session evidence pack",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := strings.TrimSpace(args[0])
			if sessionID == "" {
				return fmt.Errorf("session id is required")
			}

			st, err := state.Load("")
			if err != nil {
				return err
			}

			payload, err := exporter.EvidencePack(st, sessionID, format)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(cmd.OutOrStdout(), payload)
			return err
		},
	}

	cmd.Flags().StringVar(&format, "format", exporter.FormatJSON, "Export format: json")
	return cmd
}
