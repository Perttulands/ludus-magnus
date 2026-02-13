package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/Perttulands/ludus-magnus/internal/provider"
	"github.com/Perttulands/ludus-magnus/internal/state"
)

const (
	defaultAgentModel       = "claude-sonnet-4-5"
	defaultAgentTemperature = 1.0
	defaultAgentMaxTokens   = 4096
)

// BuildGenerationPrompt builds the deterministic template used for agent generation.
func BuildGenerationPrompt(need string, directives []string) string {
	formattedDirectives := "(none)"
	if len(directives) > 0 {
		lines := make([]string, 0, len(directives))
		for _, directive := range directives {
			trimmed := strings.TrimSpace(directive)
			if trimmed == "" {
				continue
			}
			lines = append(lines, "- "+trimmed)
		}
		if len(lines) > 0 {
			formattedDirectives = strings.Join(lines, "\n")
		}
	}

	return fmt.Sprintf(`You are a master AI agent trainer. Generate a high-quality system prompt for an AI agent.

User Need: %s

Directives (constraints/guidance):
%s

Output a JSON object with the following structure:
{
  "system_prompt": "the complete system prompt for the agent",
  "reasoning": "brief explanation of your design choices"
}

Focus on clarity, specificity, and task alignment. The agent will use Claude Sonnet 4.5.`, strings.TrimSpace(need), formattedDirectives)
}

// GenerateAgentDefinition keeps the minimal API requested by the PRD.
func GenerateAgentDefinition(need string, directives []string, p provider.Provider) (state.AgentDefinition, error) {
	definition, _, err := GenerateAgentDefinitionWithMetadata(need, directives, p)
	return definition, err
}

// GenerateAgentDefinitionWithMetadata generates an agent definition plus provider metadata.
func GenerateAgentDefinitionWithMetadata(need string, directives []string, p provider.Provider) (state.AgentDefinition, state.GenerationMetadata, error) {
	if strings.TrimSpace(need) == "" {
		return state.AgentDefinition{}, state.GenerationMetadata{}, fmt.Errorf("need is required")
	}
	if p == nil {
		return state.AgentDefinition{}, state.GenerationMetadata{}, fmt.Errorf("provider is required")
	}

	prompt := BuildGenerationPrompt(need, directives)
	generated, meta, err := p.GenerateAgent(context.Background(), prompt, nil)
	if err != nil {
		return state.AgentDefinition{}, state.GenerationMetadata{}, fmt.Errorf("generate agent: %w", err)
	}

	systemPrompt := strings.TrimSpace(generated.SystemPrompt)
	if systemPrompt == "" {
		return state.AgentDefinition{}, state.GenerationMetadata{}, fmt.Errorf("provider returned empty system prompt")
	}

	model := strings.TrimSpace(generated.Model)
	if model == "" {
		model = defaultAgentModel
	}

	temperature := generated.Temperature
	if temperature == 0 {
		temperature = defaultAgentTemperature
	}

	maxTokens := generated.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultAgentMaxTokens
	}

	info := p.GetMetadata()
	metaModel := strings.TrimSpace(info.Model)
	if metaModel == "" {
		metaModel = model
	}

	metaProvider := strings.TrimSpace(info.Provider)
	if metaProvider == "" {
		metaProvider = "unknown"
	}

	return state.AgentDefinition{
			SystemPrompt: systemPrompt,
			Model:        model,
			Temperature:  temperature,
			MaxTokens:    maxTokens,
			Tools:        []any{},
		}, state.GenerationMetadata{
			Provider:   metaProvider,
			Model:      metaModel,
			TokensUsed: meta.TokensUsed,
			DurationMS: meta.DurationMs,
			CostUSD:    meta.CostUSD,
		}, nil
}
