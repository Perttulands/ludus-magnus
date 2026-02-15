package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const anthropicVersion = "2023-06-01"

var anthropicPricing = map[string]struct {
	inputPerMillion  float64
	outputPerMillion float64
}{
	"claude-sonnet-4-5": {inputPerMillion: 3.0, outputPerMillion: 15.0},
	"claude-3-5-sonnet": {inputPerMillion: 3.0, outputPerMillion: 15.0},
}

type AnthropicProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewAnthropicProvider(apiKey, model, baseURL string) *AnthropicProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.anthropic.com"
	}
	if strings.TrimSpace(model) == "" {
		model = "claude-sonnet-4-5"
	}
	return &AnthropicProvider{
		apiKey:     apiKey,
		model:      model,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *AnthropicProvider) GenerateAgent(ctx context.Context, need string, directives []string) (AgentDefinition, Metadata, error) {
	text, usage, meta, err := p.messagesCall(ctx, "", need, 4096)
	if err != nil {
		return AgentDefinition{}, Metadata{}, err
	}

	return AgentDefinition{
		SystemPrompt: strings.TrimSpace(text),
		Model:        p.model,
		Temperature:  1.0,
		MaxTokens:    4096,
	}, p.metadataFromUsage(usage, meta.DurationMs), nil
}

func (p *AnthropicProvider) ExecuteAgent(ctx context.Context, agent AgentDefinition, input string) (string, Metadata, error) {
	maxTokens := agent.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}

	text, usage, meta, err := p.messagesCall(ctx, agent.SystemPrompt, input, maxTokens)
	if err != nil {
		return "", Metadata{}, err
	}

	return text, p.metadataFromUsage(usage, meta.DurationMs), nil
}

func (p *AnthropicProvider) GetMetadata() ProviderInfo {
	return ProviderInfo{Provider: "anthropic", Model: p.model, BaseURL: p.baseURL}
}

type anthropicMessageRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessageResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage anthropicUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type callMeta struct {
	DurationMs int
}

func (p *AnthropicProvider) messagesCall(ctx context.Context, system, user string, maxTokens int) (string, anthropicUsage, callMeta, error) {
	start := time.Now()

	reqBody := anthropicMessageRequest{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages: []anthropicMessage{{
			Role:    "user",
			Content: user,
		}},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", anthropicUsage{}, callMeta{}, fmt.Errorf("marshal anthropic request: %w", err)
	}

	url := p.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", anthropicUsage{}, callMeta{}, fmt.Errorf("create anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", anthropicUsage{}, callMeta{}, fmt.Errorf("call anthropic API: %w", err)
	}
	defer resp.Body.Close()

	var out anthropicMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", anthropicUsage{}, callMeta{}, fmt.Errorf("decode anthropic response: %w", err)
	}
	if resp.StatusCode >= 300 {
		if out.Error != nil && out.Error.Message != "" {
			return "", anthropicUsage{}, callMeta{}, fmt.Errorf("anthropic API error: %s", out.Error.Message)
		}
		return "", anthropicUsage{}, callMeta{}, fmt.Errorf("anthropic API error: status %d", resp.StatusCode)
	}
	if len(out.Content) == 0 {
		return "", anthropicUsage{}, callMeta{}, fmt.Errorf("anthropic response missing content")
	}

	return out.Content[0].Text, out.Usage, callMeta{DurationMs: int(time.Since(start).Milliseconds())}, nil
}

func (p *AnthropicProvider) metadataFromUsage(usage anthropicUsage, durationMs int) Metadata {
	rate, ok := anthropicPricing[p.model]
	if !ok {
		return Metadata{
			TokensInput:  usage.InputTokens,
			TokensOutput: usage.OutputTokens,
			TokensUsed:   usage.InputTokens + usage.OutputTokens,
			DurationMs:   durationMs,
			CostUSD:      0,
			ToolCalls:    []ToolCall{},
		}
	}

	cost := (float64(usage.InputTokens)*rate.inputPerMillion + float64(usage.OutputTokens)*rate.outputPerMillion) / 1_000_000.0
	return Metadata{
		TokensInput:  usage.InputTokens,
		TokensOutput: usage.OutputTokens,
		TokensUsed:   usage.InputTokens + usage.OutputTokens,
		DurationMs:   durationMs,
		CostUSD:      cost,
		ToolCalls:    []ToolCall{},
	}
}
