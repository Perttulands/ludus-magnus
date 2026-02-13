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
	TokensInput  int
	TokensOutput int
	TokensUsed   int
	DurationMs   int
	CostUSD      float64
	ToolCalls    []ToolCall
}

// ToolCall captures one provider-level tool invocation.
type ToolCall struct {
	Name       string
	Input      string
	Output     string
	DurationMs int
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
