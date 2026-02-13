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

func newTrainingIterateCmd() *cobra.Command {
	var providerName string
	var model string
	var baseURL string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "iterate <session-id>",
		Short: "Regenerate unlocked training lineages",
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
			if session.Mode != "training" {
				return fmt.Errorf("session %q is not in training mode", sessionID)
			}

			regenerated := []string{}
			locked := []string{}

			for _, variant := range defaultTrainingVariants {
				lineageKey, lineage, found := findLineageByName(session, variant.name)
				if !found {
					continue
				}

				if lineage.Locked {
					locked = append(locked, lineage.Name)
					continue
				}

				prevAgent, ok := latestAgent(lineage)
				if !ok {
					return fmt.Errorf("lineage %q has no agents", lineage.Name)
				}

				directives := append([]state.Directive{}, lineage.Directives.Sticky...)
				directives = append(directives, lineage.Directives.Oneshot...)
				evolutionPrompt := engine.GenerateEvolutionPrompt(lineage.Agents, lineage.Artifacts, directives)

				configProvider := strings.TrimSpace(providerName)
				if configProvider == "" {
					configProvider = strings.TrimSpace(prevAgent.GenerationMetadata.Provider)
				}

				adapter, err := provider.NewFactory(provider.Config{
					Provider: configProvider,
					Model:    modelOrDefault(model, prevAgent.Definition.Model),
					BaseURL:  baseURL,
					APIKey:   apiKey,
				})
				if err != nil {
					return err
				}

				newDefinition, generationMeta, err := engine.GenerateAgentDefinitionWithMetadata(evolutionPrompt, nil, adapter)
				if err != nil {
					return err
				}

				newAgent := state.Agent{
					ID:                 newPrefixedID("agt"),
					LineageID:          lineage.ID,
					Version:            prevAgent.Version + 1,
					Definition:         newDefinition,
					CreatedAt:          time.Now().UTC().Format(time.RFC3339),
					GenerationMetadata: generationMeta,
				}

				lineage.Agents = append(lineage.Agents, newAgent)
				lineage.Directives.Oneshot = []state.Directive{}
				session.Lineages[lineageKey] = lineage
				regenerated = append(regenerated, lineage.Name)
			}

			st.Sessions[sessionID] = session
			if err := state.Save("", st); err != nil {
				return err
			}

			regeneratedText := "none"
			if len(regenerated) > 0 {
				regeneratedText = strings.Join(regenerated, ", ")
			}

			lockedText := "none"
			if len(locked) > 0 {
				lockedText = strings.Join(locked, ", ")
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{
					"regenerated_count": len(regenerated),
					"regenerated":       regenerated,
					"locked":            locked,
				})
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"Regenerated %d lineages: %s. Locked: %s.\n",
				len(regenerated),
				regeneratedText,
				lockedText,
			)
			return err
		},
	}

	cmd.Flags().StringVar(&providerName, "provider", "", "Provider override for generation")
	cmd.Flags().StringVar(&model, "model", "", "Model override for generation")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL override for generation")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key override for generation")

	return cmd
}
