package cmd

import (
	"fmt"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/engine"
	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

type trainingVariant struct {
	name     string
	strategy string
}

var defaultTrainingVariants = []trainingVariant{
	{name: "A", strategy: "conservative approach, prioritize safety"},
	{name: "B", strategy: "balanced approach, equal priority to effectiveness and safety"},
	{name: "C", strategy: "creative approach, prioritize novel solutions"},
	{name: "D", strategy: "aggressive approach, prioritize speed and efficiency"},
}

func newTrainingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "training",
		Short: "Manage training mode flows",
	}

	cmd.AddCommand(newTrainingInitCmd())
	cmd.AddCommand(newTrainingIterateCmd())
	return cmd
}

func newTrainingInitCmd() *cobra.Command {
	var need string
	var providerName string
	var model string
	var baseURL string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a training session with lineages A/B/C/D",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := state.Load("")
			if err != nil {
				return err
			}

			now := time.Now().UTC().Format(time.RFC3339)
			sessionID := newPrefixedID("ses")

			adapter, err := provider.NewFactory(provider.Config{
				Provider: providerName,
				Model:    model,
				BaseURL:  baseURL,
				APIKey:   apiKey,
			})
			if err != nil {
				return err
			}

			lineages := make(map[string]state.Lineage, len(defaultTrainingVariants))
			lineageIDsByName := make(map[string]string, len(defaultTrainingVariants))

			for _, variant := range defaultTrainingVariants {
				lineageID := newPrefixedID("lin")
				agentID := newPrefixedID("agt")
				variantNeed := fmt.Sprintf("%s\n\nVariation strategy: %s", need, variant.strategy)

				agentDef, generationMeta, err := engine.GenerateAgentDefinitionWithMetadata(variantNeed, nil, adapter)
				if err != nil {
					return err
				}

				lineages[lineageID] = state.Lineage{
					ID:        lineageID,
					SessionID: sessionID,
					Name:      variant.name,
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
				lineageIDsByName[variant.name] = lineageID
			}

			st.Sessions[sessionID] = state.Session{
				ID:        sessionID,
				Mode:      "training",
				Need:      need,
				CreatedAt: now,
				Status:    "active",
				Lineages:  lineages,
			}

			if err := state.Save("", st); err != nil {
				return err
			}

			if isJSONOutput(cmd) {
				payload := map[string]any{"session_id": sessionID}
				for _, variant := range defaultTrainingVariants {
					payload[fmt.Sprintf("lineage_%s_id", variant.name)] = lineageIDsByName[variant.name]
				}
				return writeJSON(cmd, payload)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "session_id=%s\n", sessionID); err != nil {
				return err
			}
			for _, variant := range defaultTrainingVariants {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "lineage_%s_id=%s\n", variant.name, lineageIDsByName[variant.name]); err != nil {
					return err
				}
			}
			return nil
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
