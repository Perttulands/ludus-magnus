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

// OllamaProvider uses Ollama's native /api/chat endpoint.
type OllamaProvider struct {
	model      string
	baseURL    string
	httpClient *http.Client
}

// NewOllamaProvider creates a provider targeting Ollama's native API.
func NewOllamaProvider(model, baseURL string) *OllamaProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "http://localhost:11434"
	}
	if strings.TrimSpace(model) == "" {
		model = "qwen3:8b"
	}
	return &OllamaProvider{
		model:      model,
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	PromptEvalCount int   `json:"prompt_eval_count"`
	EvalCount       int   `json:"eval_count"`
	TotalDuration   int64 `json:"total_duration"`
	EvalDuration    int64 `json:"eval_duration"`
}

func (p *OllamaProvider) GenerateAgent(ctx context.Context, need string, directives []string) (AgentDefinition, Metadata, error) {
	opts := map[string]any{
		"num_ctx":     8192,
		"temperature": 1.0,
	}
	text, meta, err := p.chat(ctx, "", need, opts)
	if err != nil {
		return AgentDefinition{}, Metadata{}, fmt.Errorf("ollama generate: %w", err)
	}
	return AgentDefinition{
		SystemPrompt: strings.TrimSpace(text),
		Model:        p.model,
		Temperature:  1.0,
		MaxTokens:    4096,
	}, meta, nil
}

func (p *OllamaProvider) ExecuteAgent(ctx context.Context, agent AgentDefinition, input string) (string, Metadata, error) {
	opts := map[string]any{
		"num_ctx":     8192,
		"temperature": agent.Temperature,
	}
	if agent.InferenceOptions != nil {
		for k, v := range agent.InferenceOptions {
			opts[k] = v
		}
	}
	text, meta, err := p.chat(ctx, agent.SystemPrompt, input, opts)
	if err != nil {
		return "", Metadata{}, fmt.Errorf("ollama execute: %w", err)
	}
	return text, meta, nil
}

func (p *OllamaProvider) GetMetadata() ProviderInfo {
	return ProviderInfo{
		Provider: "ollama",
		Model:    p.model,
		BaseURL:  p.baseURL,
	}
}

func (p *OllamaProvider) chat(ctx context.Context, system, user string, opts map[string]any) (string, Metadata, error) {
	var msgs []ollamaMessage
	if system != "" {
		msgs = append(msgs, ollamaMessage{Role: "system", Content: system})
	}
	msgs = append(msgs, ollamaMessage{Role: "user", Content: user})

	req := ollamaChatRequest{
		Model:    p.model,
		Messages: msgs,
		Stream:   false,
		Options:  opts,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", Metadata{}, err
	}

	start := time.Now()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", Metadata{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return "", Metadata{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return "", Metadata{}, fmt.Errorf("ollama HTTP %d: %s", resp.StatusCode, errBody.String())
	}

	var result ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", Metadata{}, fmt.Errorf("decode response: %w", err)
	}

	durationMs := int(time.Since(start).Milliseconds())
	if result.TotalDuration > 0 {
		durationMs = int(result.TotalDuration / 1_000_000)
	}

	meta := Metadata{
		TokensInput:  result.PromptEvalCount,
		TokensOutput: result.EvalCount,
		TokensUsed:   result.PromptEvalCount + result.EvalCount,
		DurationMs:   durationMs,
		CostUSD:      0,
	}

	return result.Message.Content, meta, nil
}
