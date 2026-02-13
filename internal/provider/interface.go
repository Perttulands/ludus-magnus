package provider

import "context"

// AgentDefinition is the prompt/model payload needed by providers.
type AgentDefinition struct {
	SystemPrompt string
	Model        string
	Temperature  float64
	MaxTokens    int
}

// Metadata captures provider call observability signals.
type Metadata struct {
	TokensUsed int
	DurationMs int
	CostUSD    float64
}

// ProviderInfo describes the provider instance identity.
type ProviderInfo struct {
	Provider string
	Model    string
	BaseURL  string
}

// Provider exposes generation and execution operations across LLM vendors.
type Provider interface {
	GenerateAgent(ctx context.Context, need string, directives []string) (AgentDefinition, Metadata, error)
	ExecuteAgent(ctx context.Context, agent AgentDefinition, input string) (string, Metadata, error)
	GetMetadata() ProviderInfo
}
