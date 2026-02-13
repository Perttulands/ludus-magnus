package cmd

import (
	"fmt"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/engine"
	"github.com/Perttulands/ludus-magnus/internal/provider"
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
	var providerName string
	var model string
	var baseURL string
	var apiKey string

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
			agentID := newPrefixedID("agt")

			adapter, err := provider.NewFactory(provider.Config{
				Provider: providerName,
				Model:    model,
				BaseURL:  baseURL,
				APIKey:   apiKey,
			})
			if err != nil {
				return err
			}

			agentDef, generationMeta, err := engine.GenerateAgentDefinitionWithMetadata(need, nil, adapter)
			if err != nil {
				return err
			}

			mainLineage := state.Lineage{
				ID:        lineageID,
				SessionID: sessionID,
				Name:      "main",
				Locked:    false,
				Agents: []state.Agent{
					{
						ID:                 agentID,
						LineageID:          lineageID,
						Version:            1,
						Definition:         agentDef,
						CreatedAt:          now,
						GenerationMetadata: generationMeta,
					},
				},
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

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{
					"session_id": sessionID,
					"lineage_id": lineageID,
				})
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "session_id=%s\n", sessionID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "lineage_id=%s\n", lineageID)
			return err
		},
	}

	cmd.Flags().StringVar(&need, "need", "", "Intent for the session")
	cmd.Flags().StringVar(&providerName, "provider", "anthropic", "Provider name (anthropic or openai-compatible)")
	cmd.Flags().StringVar(&model, "model", "", "Override provider model")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Override provider base URL")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Override provider API key")
	_ = cmd.MarkFlagRequired("need")

	return cmd
}
