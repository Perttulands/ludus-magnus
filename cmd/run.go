package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/engine"
	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var input string
	var lineageName string
	var mode string
	var executor string
	var providerName string
	var model string
	var baseURL string
	var apiKey string

	cmd := &cobra.Command{
		Use:   "run <session-id>",
		Short: "Run latest agent on one input and store an artifact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]

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

			_, lineage, ok := findLineageByName(session, selectedLineage)
			if !ok {
				return fmt.Errorf("lineage %q not found", selectedLineage)
			}

			agent, ok := latestAgent(lineage)
			if !ok {
				return fmt.Errorf("lineage %q has no agents", selectedLineage)
			}

			request := engine.ExecuteRequest{
				Mode:       mode,
				Input:      input,
				Definition: agent.Definition,
				Executor:   executor,
			}

			if strings.TrimSpace(mode) == "" || strings.TrimSpace(mode) == engine.ExecutionModeAPI {
				configProvider := strings.TrimSpace(providerName)
				if configProvider == "" {
					configProvider = strings.TrimSpace(agent.GenerationMetadata.Provider)
				}
				adapter, err := provider.NewFactory(provider.Config{
					Provider: configProvider,
					Model:    modelOrDefault(model, agent.Definition.Model),
					BaseURL:  baseURL,
					APIKey:   apiKey,
				})
				if err != nil {
					return err
				}
				request.Provider = adapter
			}

			result, err := engine.Execute(context.Background(), request)
			if err != nil {
				return err
			}

			artifact := state.Artifact{
				AgentID:           agent.ID,
				Input:             input,
				Output:            result.Output,
				ExecutionMetadata: result.Metadata,
			}

			artifactID, err := state.AddArtifact(sessionID, lineage.ID, artifact)
			if err != nil {
				return err
			}

			if isJSONOutput(cmd) {
				return writeJSON(cmd, map[string]any{"artifact_id": artifactID})
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "artifact_id=%s\n", artifactID)
			return err
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Input for agent execution")
	cmd.Flags().StringVar(&lineageName, "lineage", "", "Lineage name (main, A, B, C, D)")
	cmd.Flags().StringVar(&mode, "mode", engine.ExecutionModeAPI, "Execution mode: api or cli")
	cmd.Flags().StringVar(&executor, "executor", "", "CLI executor for mode=cli: claude or codex")
	cmd.Flags().StringVar(&providerName, "provider", "", "Provider override for mode=api")
	cmd.Flags().StringVar(&model, "model", "", "Model override for mode=api")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL override for mode=api")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key override for mode=api")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func findLineageByName(session state.Session, target string) (string, state.Lineage, bool) {
	for key, lineage := range session.Lineages {
		if lineage.Name == target {
			return key, lineage, true
		}
	}
	return "", state.Lineage{}, false
}

func latestAgent(lineage state.Lineage) (state.Agent, bool) {
	if len(lineage.Agents) == 0 {
		return state.Agent{}, false
	}

	latest := lineage.Agents[0]
	for _, candidate := range lineage.Agents[1:] {
		if candidate.Version > latest.Version {
			latest = candidate
		}
	}
	return latest, true
}

func modelOrDefault(override, fallback string) string {
	trimmed := strings.TrimSpace(override)
	if trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(fallback)
}
