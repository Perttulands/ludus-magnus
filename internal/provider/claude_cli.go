package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClaudeCLIProvider implements Provider using the `claude` CLI in print mode.
// Uses Claude Code's existing OAuth auth — no API key required.
type ClaudeCLIProvider struct {
	model   string
	binary  string
}

// NewClaudeCLIProvider creates a provider that delegates to the claude CLI.
func NewClaudeCLIProvider(model, binary string) *ClaudeCLIProvider {
	if strings.TrimSpace(model) == "" {
		model = "sonnet"
	}
	if strings.TrimSpace(binary) == "" {
		binary = "claude"
	}
	return &ClaudeCLIProvider{model: model, binary: binary}
}

func (p *ClaudeCLIProvider) GenerateAgent(ctx context.Context, need string, directives []string) (AgentDefinition, Metadata, error) {
	start := time.Now()

	output, meta, err := p.call(ctx, "", need)
	if err != nil {
		return AgentDefinition{}, Metadata{}, fmt.Errorf("generate agent: %w", err)
	}
	meta.DurationMs = int(time.Since(start).Milliseconds())

	return AgentDefinition{
		SystemPrompt: strings.TrimSpace(output),
		Model:        p.model,
		Temperature:  1.0,
		MaxTokens:    4096,
	}, meta, nil
}

func (p *ClaudeCLIProvider) ExecuteAgent(ctx context.Context, agent AgentDefinition, input string) (string, Metadata, error) {
	start := time.Now()

	output, meta, err := p.call(ctx, agent.SystemPrompt, input)
	if err != nil {
		return "", Metadata{}, fmt.Errorf("execute agent: %w", err)
	}
	meta.DurationMs = int(time.Since(start).Milliseconds())

	return output, meta, nil
}

func (p *ClaudeCLIProvider) GetMetadata() ProviderInfo {
	return ProviderInfo{Provider: "claude-cli", Model: p.model, BaseURL: ""}
}

// claudeJSONResult is the JSON output from `claude -p --output-format json`.
type claudeJSONResult struct {
	Type       string `json:"type"`
	Result     string `json:"result"`
	DurationMs int    `json:"duration_ms"`
	IsError    bool   `json:"is_error"`
	TotalCost  float64 `json:"total_cost_usd"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	ModelUsage map[string]struct {
		InputTokens              int     `json:"inputTokens"`
		OutputTokens             int     `json:"outputTokens"`
		CacheReadInputTokens     int     `json:"cacheReadInputTokens"`
		CacheCreationInputTokens int     `json:"cacheCreationInputTokens"`
		CostUSD                  float64 `json:"costUSD"`
	} `json:"modelUsage"`
}

func (p *ClaudeCLIProvider) call(ctx context.Context, systemPrompt, userPrompt string) (string, Metadata, error) {
	binPath, err := exec.LookPath(p.binary)
	if err != nil {
		return "", Metadata{}, fmt.Errorf("claude binary not found: %w", err)
	}

	args := []string{
		"-p",
		"--output-format", "json",
		"--model", p.model,
		"--no-session-persistence",
	}

	if strings.TrimSpace(systemPrompt) != "" {
		args = append(args, "--system-prompt", systemPrompt)
	}

	args = append(args, userPrompt)

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return "", Metadata{}, fmt.Errorf("claude CLI error: %s: %w", trimmed, err)
		}
		return "", Metadata{}, fmt.Errorf("claude CLI error: %w", err)
	}

	// Parse JSON output
	var result claudeJSONResult
	if err := json.Unmarshal(output, &result); err != nil {
		// Fallback: if JSON parse fails, treat raw output as the result text
		return strings.TrimSpace(string(output)), Metadata{
			ToolCalls: []ToolCall{},
		}, nil
	}

	if result.IsError {
		return "", Metadata{}, fmt.Errorf("claude CLI returned error: %s", result.Result)
	}

	// Extract token usage from modelUsage map
	meta := Metadata{
		DurationMs: result.DurationMs,
		CostUSD:    result.TotalCost,
		ToolCalls:  []ToolCall{},
	}

	for _, usage := range result.ModelUsage {
		meta.TokensInput += usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
		meta.TokensOutput += usage.OutputTokens
		meta.CostUSD = usage.CostUSD
	}
	meta.TokensUsed = meta.TokensInput + meta.TokensOutput

	return result.Result, meta, nil
}
