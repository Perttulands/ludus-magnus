package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/Perttulands/ludus-magnus/internal/engine"
	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

var alternativeTrainingVariants = []trainingVariant{
	{name: "A", strategy: "fundamentally different methodology: deterministic rule-based workflow"},
	{name: "B", strategy: "fundamentally different methodology: retrieval-first evidence-driven workflow"},
	{name: "C", strategy: "fundamentally different methodology: planning-first decomposition workflow"},
	{name: "D", strategy: "fundamentally different methodology: critique-and-revise self-review workflow"},
}

func newPromoteCmd() *cobra.Command {
	var strategy string
	var providerName string
	var model string
	var baseURL string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "promote <session-id>",
		Short: "Promote a quickstart session into training mode",
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

			session, ok := st.Sessions[sessionID]
			if !ok {
				return fmt.Errorf("session %q not found", sessionID)
			}
			if session.Mode != "quickstart" {
				return fmt.Errorf("session %q is not in quickstart mode", sessionID)
			}

			_, mainLineage, found := findLineageByName(session, "main")
			if !found {
				return fmt.Errorf("lineage %q not found", "main")
			}
			baseAgent, ok := latestAgent(mainLineage)
			if !ok {
				return fmt.Errorf("lineage %q has no agents", "main")
			}

			variants, err := variantsForStrategy(strategy)
			if err != nil {
				return err
			}

			configProvider := strings.TrimSpace(providerName)
			if configProvider == "" {
				configProvider = strings.TrimSpace(baseAgent.GenerationMetadata.Provider)
			}

			adapter, err := provider.NewFactory(provider.Config{
				Provider: configProvider,
				Model:    modelOrDefault(model, baseAgent.Definition.Model),
				BaseURL:  baseURL,
				APIKey:   apiKey,
			})
			if err != nil {
				return err
			}

			now := time.Now().UTC().Format(time.RFC3339)
			lineages := make(map[string]state.Lineage, len(variants))
			for _, variant := range variants {
				lineageID := newPrefixedID("lin")
				agentID := newPrefixedID("agt")

				promotionPrompt := fmt.Sprintf(
					"%s\n\nOriginal system prompt:\n%s\n\nPromotion strategy: %s",
					session.Need,
					baseAgent.Definition.SystemPrompt,
					variant.strategy,
				)

				agentDef, generationMeta, err := engine.GenerateAgentDefinitionWithMetadata(promotionPrompt, nil, adapter)
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
			}

			session.Mode = "training"
			session.Lineages = lineages
			st.Sessions[sessionID] = session

			if err := state.Save("", st); err != nil {
				return err
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{
					"session_id": sessionID,
					"mode":       "training",
					"lineages":   []string{"A", "B", "C", "D"},
				})
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Session promoted to training mode with 4 lineages")
			return err
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "variations", "Promotion strategy: variations or alternatives")
	cmd.Flags().StringVar(&providerName, "provider", "", "Provider override for generation")
	cmd.Flags().StringVar(&model, "model", "", "Model override for generation")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL override for generation")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key override for generation")

	return cmd
}

func variantsForStrategy(strategy string) ([]trainingVariant, error) {
	switch strings.TrimSpace(strategy) {
	case "", "variations":
		return defaultTrainingVariants, nil
	case "alternatives":
		return alternativeTrainingVariants, nil
	default:
		return nil, fmt.Errorf("invalid --strategy %q (expected variations or alternatives)", strategy)
	}
}
