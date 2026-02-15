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

var openAICompatiblePricing = map[string]struct {
	inputPerMillion  float64
	outputPerMillion float64
}{
	"gpt-4o-mini": {inputPerMillion: 0.15, outputPerMillion: 0.60},
}

type OpenAICompatibleProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewOpenAICompatibleProvider(apiKey, model, baseURL string) *OpenAICompatibleProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAICompatibleProvider{
		apiKey:     apiKey,
		model:      model,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *OpenAICompatibleProvider) GenerateAgent(ctx context.Context, need string, directives []string) (AgentDefinition, Metadata, error) {
	text, usage, meta, err := p.chatCompletionCall(ctx, "", need, 4096, 1.0)
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

func (p *OpenAICompatibleProvider) ExecuteAgent(ctx context.Context, agent AgentDefinition, input string) (string, Metadata, error) {
	maxTokens := agent.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	temp := agent.Temperature
	if temp == 0 {
		temp = 1.0
	}

	text, usage, meta, err := p.chatCompletionCall(ctx, agent.SystemPrompt, input, maxTokens, temp)
	if err != nil {
		return "", Metadata{}, err
	}
	return text, p.metadataFromUsage(usage, meta.DurationMs), nil
}

func (p *OpenAICompatibleProvider) GetMetadata() ProviderInfo {
	return ProviderInfo{Provider: "openai-compatible", Model: p.model, BaseURL: p.baseURL}
}

type openAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIChatMsg `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMsg `json:"message"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (p *OpenAICompatibleProvider) chatCompletionCall(ctx context.Context, system, user string, maxTokens int, temp float64) (string, openAIUsage, callMeta, error) {
	start := time.Now()

	messages := []openAIChatMsg{}
	if strings.TrimSpace(system) != "" {
		messages = append(messages, openAIChatMsg{Role: "system", Content: system})
	}
	messages = append(messages, openAIChatMsg{Role: "user", Content: user})

	reqBody := openAIChatRequest{
		Model:       p.model,
		Messages:    messages,
		Temperature: temp,
		MaxTokens:   maxTokens,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", openAIUsage{}, callMeta{}, fmt.Errorf("marshal openai-compatible request: %w", err)
	}

	url := p.completionsURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", openAIUsage{}, callMeta{}, fmt.Errorf("create openai-compatible request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", openAIUsage{}, callMeta{}, fmt.Errorf("call openai-compatible API: %w", err)
	}
	defer resp.Body.Close()

	var out openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", openAIUsage{}, callMeta{}, fmt.Errorf("decode openai-compatible response: %w", err)
	}
	if resp.StatusCode >= 300 {
		if out.Error != nil && out.Error.Message != "" {
			return "", openAIUsage{}, callMeta{}, fmt.Errorf("openai-compatible API error: %s", out.Error.Message)
		}
		return "", openAIUsage{}, callMeta{}, fmt.Errorf("openai-compatible API error: status %d", resp.StatusCode)
	}
	if len(out.Choices) == 0 {
		return "", openAIUsage{}, callMeta{}, fmt.Errorf("openai-compatible response missing choices")
	}

	return out.Choices[0].Message.Content, out.Usage, callMeta{DurationMs: int(time.Since(start).Milliseconds())}, nil
}

func (p *OpenAICompatibleProvider) completionsURL() string {
	if strings.HasSuffix(p.baseURL, "/v1") {
		return p.baseURL + "/chat/completions"
	}
	if strings.Contains(p.baseURL, "/v1/") {
		return strings.TrimRight(p.baseURL, "/") + "/chat/completions"
	}
	return p.baseURL + "/chat/completions"
}

func (p *OpenAICompatibleProvider) metadataFromUsage(usage openAIUsage, durationMs int) Metadata {
	tokensInput := usage.PromptTokens
	tokensOutput := usage.CompletionTokens
	tokens := usage.TotalTokens
	if tokens == 0 {
		tokens = tokensInput + tokensOutput
	}

	rate, ok := openAICompatiblePricing[p.model]
	if !ok {
		return Metadata{
			TokensInput:  tokensInput,
			TokensOutput: tokensOutput,
			TokensUsed:   tokens,
			DurationMs:   durationMs,
			CostUSD:      0,
			ToolCalls:    []ToolCall{},
		}
	}
	cost := (float64(tokensInput)*rate.inputPerMillion + float64(tokensOutput)*rate.outputPerMillion) / 1_000_000.0
	return Metadata{
		TokensInput:  tokensInput,
		TokensOutput: tokensOutput,
		TokensUsed:   tokens,
		DurationMs:   durationMs,
		CostUSD:      cost,
		ToolCalls:    []ToolCall{},
	}
}
