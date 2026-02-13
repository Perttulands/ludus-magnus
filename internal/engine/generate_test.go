package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/Perttulands/ludus-magnus/internal/provider"
)

type mockProvider struct {
	gotNeed       string
	gotDirectives []string
	definition    provider.AgentDefinition
	metadata      provider.Metadata
	info          provider.ProviderInfo
	err           error
}

func (m *mockProvider) GenerateAgent(_ context.Context, need string, directives []string) (provider.AgentDefinition, provider.Metadata, error) {
	m.gotNeed = need
	m.gotDirectives = directives
	if m.err != nil {
		return provider.AgentDefinition{}, provider.Metadata{}, m.err
	}
	return m.definition, m.metadata, nil
}

func (m *mockProvider) ExecuteAgent(context.Context, provider.AgentDefinition, string) (string, provider.Metadata, error) {
	return "", provider.Metadata{}, nil
}

func (m *mockProvider) GetMetadata() provider.ProviderInfo {
	return m.info
}

func TestBuildGenerationPromptIncludesNeedAndDirectives(t *testing.T) {
	prompt := BuildGenerationPrompt("customer care agent", []string{"be concise", "ask clarifying questions"})
	if !strings.Contains(prompt, "User Need: customer care agent") {
		t.Fatalf("prompt missing need: %q", prompt)
	}
	if !strings.Contains(prompt, "- be concise") || !strings.Contains(prompt, "- ask clarifying questions") {
		t.Fatalf("prompt missing directives: %q", prompt)
	}
	if !strings.Contains(prompt, `"system_prompt"`) {
		t.Fatalf("prompt missing expected json output instructions: %q", prompt)
	}
}

func TestGenerateAgentDefinitionWithMetadataAppliesDefaults(t *testing.T) {
	p := &mockProvider{
		definition: provider.AgentDefinition{SystemPrompt: "You are a specialist support agent."},
		metadata: provider.Metadata{
			TokensUsed: 321,
			DurationMs: 45,
			CostUSD:    0.0123,
		},
		info: provider.ProviderInfo{Provider: "anthropic", Model: "claude-sonnet-4-5"},
	}

	definition, generationMeta, err := GenerateAgentDefinitionWithMetadata("customer care agent", []string{"be concise"}, p)
	if err != nil {
		t.Fatalf("GenerateAgentDefinitionWithMetadata returned error: %v", err)
	}
	if definition.SystemPrompt == "" {
		t.Fatalf("expected non-empty system prompt")
	}
	if definition.Model != "claude-sonnet-4-5" {
		t.Fatalf("expected default model, got %q", definition.Model)
	}
	if definition.Temperature != 1.0 {
		t.Fatalf("expected default temperature 1.0, got %f", definition.Temperature)
	}
	if definition.MaxTokens != 4096 {
		t.Fatalf("expected default max_tokens 4096, got %d", definition.MaxTokens)
	}
	if len(definition.Tools) != 0 {
		t.Fatalf("expected empty tools array, got %d entries", len(definition.Tools))
	}
	if generationMeta.Provider != "anthropic" {
		t.Fatalf("expected provider anthropic, got %q", generationMeta.Provider)
	}
	if generationMeta.TokensUsed != 321 {
		t.Fatalf("unexpected tokens used: %d", generationMeta.TokensUsed)
	}
	if len(p.gotDirectives) != 0 {
		t.Fatalf("expected engine to pass formatted prompt and no directives to provider")
	}
	if !strings.Contains(p.gotNeed, "User Need: customer care agent") {
		t.Fatalf("expected formatted prompt, got %q", p.gotNeed)
	}
}
