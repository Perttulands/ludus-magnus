package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAnthropicProviderGenerateAgentParsesResponse(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "System prompt from anthropic"}},
			"usage":   map[string]any{"input_tokens": 100, "output_tokens": 40},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	agent, meta, err := p.GenerateAgent(context.Background(), "customer support", []string{"be brief"})
	if err != nil {
		t.Fatalf("GenerateAgent returned error: %v", err)
	}

	if gotAuth != "test-key" {
		t.Fatalf("expected x-api-key header, got %q", gotAuth)
	}
	if gotVersion == "" {
		t.Fatalf("expected anthropic-version header")
	}
	if agent.SystemPrompt != "System prompt from anthropic" {
		t.Fatalf("unexpected system prompt: %q", agent.SystemPrompt)
	}
	if meta.TokensUsed != 140 {
		t.Fatalf("expected 140 tokens used, got %d", meta.TokensUsed)
	}
	if meta.CostUSD <= 0 {
		t.Fatalf("expected positive cost, got %f", meta.CostUSD)
	}
}

func TestOpenAICompatibleProviderExecuteAgentParsesResponse(t *testing.T) {
	t.Parallel()

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "execution output"}}},
			"usage":   map[string]any{"prompt_tokens": 25, "completion_tokens": 15, "total_tokens": 40},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("open-key", "gpt-4o-mini", server.URL)
	out, meta, err := p.ExecuteAgent(context.Background(), AgentDefinition{SystemPrompt: "you are helpful"}, "hello")
	if err != nil {
		t.Fatalf("ExecuteAgent returned error: %v", err)
	}

	if gotAuth != "Bearer open-key" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if out != "execution output" {
		t.Fatalf("unexpected output: %q", out)
	}
	if meta.TokensUsed != 40 {
		t.Fatalf("expected 40 total tokens, got %d", meta.TokensUsed)
	}
	if meta.DurationMs < 0 {
		t.Fatalf("expected non-negative duration, got %d", meta.DurationMs)
	}
}

func TestFactoryDefaultsToAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	provider, err := NewFactory(Config{Model: "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("NewFactory returned error: %v", err)
	}

	if provider.GetMetadata().Provider != "anthropic" {
		t.Fatalf("expected anthropic provider, got %q", provider.GetMetadata().Provider)
	}
}

func TestFactoryValidatesCredentials(t *testing.T) {
	for _, k := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENAI_COMPATIBLE_API_KEY", "API_KEY"} {
		_ = os.Unsetenv(k)
	}

	_, err := NewFactory(Config{Provider: "anthropic"})
	if err == nil {
		t.Fatalf("expected error for missing anthropic credentials")
	}

	_, err = NewFactory(Config{Provider: "openai-compatible"})
	if err == nil {
		t.Fatalf("expected error for missing openai-compatible credentials")
	}
}
