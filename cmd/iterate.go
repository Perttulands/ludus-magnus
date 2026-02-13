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

func newIterateCmd() *cobra.Command {
	var lineageName string
	var providerName string
	var model string
	var baseURL string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "iterate <session-id>",
		Short: "Generate the next agent version from lineage evolution feedback",
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

			selectedLineage := strings.TrimSpace(lineageName)
			if selectedLineage == "" {
				if session.Mode == "quickstart" {
					selectedLineage = "main"
				} else {
					return fmt.Errorf("--lineage is required for non-quickstart sessions")
				}
			}

			lineageKey, lineage, ok := findLineageByName(session, selectedLineage)
			if !ok {
				return fmt.Errorf("lineage %q not found", selectedLineage)
			}

			prevAgent, ok := latestAgent(lineage)
			if !ok {
				return fmt.Errorf("lineage %q has no agents", selectedLineage)
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
			st.Sessions[sessionID] = session

			if err := state.Save("", st); err != nil {
				return err
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{
					"agent_id": newAgent.ID,
					"version":  newAgent.Version,
				})
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "agent_id=%s\n", newAgent.ID); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "version=%d\n", newAgent.Version)
			return err
		},
	}

	cmd.Flags().StringVar(&lineageName, "lineage", "", "Lineage name (main, A, B, C, D)")
	cmd.Flags().StringVar(&providerName, "provider", "", "Provider override for generation")
	cmd.Flags().StringVar(&model, "model", "", "Model override for generation")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL override for generation")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key override for generation")

	return cmd
}
