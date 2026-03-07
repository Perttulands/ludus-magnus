package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PiCLIProvider implements Provider using the `pi` CLI in print mode with Ollama.
// Mirrors ClaudeCLIProvider but targets Pi's CLI flags and JSON output format.
type PiCLIProvider struct {
	model   string
	binary  string
	baseURL string // Ollama endpoint (default: http://localhost:11434)
}

// NewPiCLIProvider creates a provider that delegates to the pi CLI with Ollama backend.
func NewPiCLIProvider(model, binary, baseURL string) *PiCLIProvider {
	if strings.TrimSpace(model) == "" {
		model = "qwen3.5:9b"
	}
	if strings.TrimSpace(binary) == "" {
		binary = "pi"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:11434"
	}
	return &PiCLIProvider{
		model:   model,
		binary:  binary,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (p *PiCLIProvider) GenerateAgent(ctx context.Context, need string, directives []string) (AgentDefinition, Metadata, error) {
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

func (p *PiCLIProvider) ExecuteAgent(ctx context.Context, agent AgentDefinition, input string) (string, Metadata, error) {
	start := time.Now()

	output, meta, err := p.call(ctx, agent.SystemPrompt, input)
	if err != nil {
		return "", Metadata{}, fmt.Errorf("execute agent: %w", err)
	}
	meta.DurationMs = int(time.Since(start).Milliseconds())

	return output, meta, nil
}

func (p *PiCLIProvider) GetMetadata() ProviderInfo {
	return ProviderInfo{Provider: "pi-cli", Model: p.model, BaseURL: p.baseURL}
}

// piJSONResult captures Pi's --mode json output.
// Pi outputs a JSON object with session metadata when run in json mode.
type piJSONResult struct {
	Response string `json:"response"`
	Result   string `json:"result"`
	Model    string `json:"model"`
	Turns    int    `json:"turns"`
	Usage    struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	DurationMs int  `json:"duration_ms"`
	IsError    bool `json:"is_error"`
}

func (p *PiCLIProvider) call(ctx context.Context, systemPrompt, userPrompt string) (string, Metadata, error) {
	binPath, err := exec.LookPath(p.binary)
	if err != nil {
		return "", Metadata{}, fmt.Errorf("pi binary not found: %w", err)
	}

	args := []string{
		"-p",
		"--provider", "ollama",
		"--model", p.model,
		"--mode", "json",
		"--no-session",
		"--no-extensions",
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
			return "", Metadata{}, fmt.Errorf("pi CLI error: %s: %w", trimmed, err)
		}
		return "", Metadata{}, fmt.Errorf("pi CLI error: %w", err)
	}

	// Try to parse JSON output
	var result piJSONResult
	if err := json.Unmarshal(output, &result); err != nil {
		// Fallback: raw output as text
		return strings.TrimSpace(string(output)), Metadata{
			CostUSD:   0,
			ToolCalls: []ToolCall{},
		}, nil
	}

	if result.IsError {
		return "", Metadata{}, fmt.Errorf("pi CLI returned error: %s", p.resultText(&result))
	}

	meta := Metadata{
		DurationMs:   result.DurationMs,
		TokensInput:  result.Usage.InputTokens,
		TokensOutput: result.Usage.OutputTokens,
		TokensUsed:   result.Usage.TotalTokens,
		CostUSD:      0, // Local inference, no cost
		ToolCalls:    []ToolCall{},
	}
	if meta.TokensUsed == 0 {
		meta.TokensUsed = meta.TokensInput + meta.TokensOutput
	}

	return p.resultText(&result), meta, nil
}

// resultText returns the response text, checking both fields Pi might use.
func (p *PiCLIProvider) resultText(r *piJSONResult) string {
	if r.Response != "" {
		return r.Response
	}
	return r.Result
}
